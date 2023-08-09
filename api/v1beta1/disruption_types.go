// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package v1beta1

import (
	"crypto/md5"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"time"

	chaosapi "github.com/DataDog/chaos-controller/api"
	eventtypes "github.com/DataDog/chaos-controller/eventnotifier/types"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"github.com/DataDog/chaos-controller/utils"
	"github.com/hashicorp/go-multierror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/intstr"
	goyaml "sigs.k8s.io/yaml"
)

// DisruptionSpec defines the desired state of Disruption
// +ddmark:validation:ExclusiveFields={ContainerFailure,CPUPressure,DiskPressure,NodeFailure,Network,DNS,DiskFailure}
// +ddmark:validation:ExclusiveFields={NodeFailure,CPUPressure,DiskPressure,ContainerFailure,Network,DNS,DiskFailure}
// +ddmark:validation:LinkedFieldsValueWithTrigger={NodeFailure,Level}
// +ddmark:validation:AtLeastOneOf={DNS,CPUPressure,Network,NodeFailure,ContainerFailure,DiskPressure,GRPC,DiskFailure}
// +ddmark:validation:AtLeastOneOf={Selector,AdvancedSelector}
type DisruptionSpec struct {
	// +kubebuilder:validation:Required
	// +ddmark:validation:Required=true
	Count *intstr.IntOrString `json:"count"` // number of pods to target in either integer form or percent form appended with a %
	// AllowDisruptedTargets allow pods with one or several other active disruptions, with disruption kinds that does not intersect
	// with this disruption kinds, to be returned as part of eligible targets for this disruption
	// - e.g. apply a CPU pressure and later, apply a container failure for a short duration
	// NB: it's ALWAYS forbidden to apply the same disruption kind to the same target to avoid unreliable effects due to competing interactions
	AllowDisruptedTargets bool `json:"allowDisruptedTargets,omitempty"`
	// +nullable
	Selector labels.Set `json:"selector,omitempty"` // label selector
	// +nullable
	AdvancedSelector []metav1.LabelSelectorRequirement `json:"advancedSelector,omitempty"` // advanced label selector
	// +nullable
	Filter          *DisruptionFilter `json:"filter,omitempty"`
	DryRun          bool              `json:"dryRun,omitempty"`          // enable dry-run mode
	OnInit          bool              `json:"onInit,omitempty"`          // enable disruption on init
	Unsafemode      *UnsafemodeSpec   `json:"unsafeMode,omitempty"`      // unsafemode spec used to turn off safemode safety nets
	StaticTargeting bool              `json:"staticTargeting,omitempty"` // enable dynamic targeting and cluster observation
	// +nullable
	Triggers DisruptionTriggers `json:"triggers,omitempty"` // alter the pre-injection lifecycle
	// +nullable
	Pulse    *DisruptionPulse   `json:"pulse,omitempty"`    // enable pulsing diruptions and specify the duration of the active state and the dormant state of the pulsing duration
	Duration DisruptionDuration `json:"duration,omitempty"` // time from disruption creation until chaos pods are deleted and no more are created
	// Level defines what the disruption will target, either a pod or a node
	// +kubebuilder:default=pod
	// +kubebuilder:validation:Enum=pod;node
	// +ddmark:validation:Enum=pod;node
	Level      chaostypes.DisruptionLevel `json:"level,omitempty"`
	Containers []string                   `json:"containers,omitempty"`
	// +nullable
	Network *NetworkDisruptionSpec `json:"network,omitempty"`
	// +nullable
	NodeFailure *NodeFailureSpec `json:"nodeFailure,omitempty"`
	// +nullable
	ContainerFailure *ContainerFailureSpec `json:"containerFailure,omitempty"`
	// +nullable
	CPUPressure *CPUPressureSpec `json:"cpuPressure,omitempty"`
	// +nullable
	DiskPressure *DiskPressureSpec `json:"diskPressure,omitempty"`
	// +nullable
	DiskFailure *DiskFailureSpec `json:"diskFailure,omitempty"`
	// +nullable
	DNS DNSDisruptionSpec `json:"dns,omitempty"`
	// +nullable
	GRPC *GRPCDisruptionSpec `json:"grpc,omitempty"`
	// +nullable
	Reporting *Reporting `json:"reporting,omitempty"`
}

// DisruptionTriggers holds the options for changing when injector pods are created, and the timing of when the injection occurs
type DisruptionTriggers struct {
	Inject     DisruptionTrigger `json:"inject,omitempty"`
	CreatePods DisruptionTrigger `json:"createPods,omitempty"`
}

func (dt DisruptionTriggers) IsZero() bool {
	return dt.Inject.IsZero() && dt.CreatePods.IsZero()
}

// +ddmark:validation:ExclusiveFields={NotBefore,Offset}
type DisruptionTrigger struct {
	// inject.notBefore: Normal reconciliation and chaos pod creation will occur, but chaos pods will wait to inject until NotInjectedBefore. Must be after NoPodsBefore if both are specified
	// createPods.notBefore: Will skip reconciliation until this time, no chaos pods will be created until after NoPodsBefore
	// +nullable
	NotBefore metav1.Time `json:"notBefore,omitempty"`
	// inject.offset: Identical to NotBefore, but specified as an offset from max(CreationTimestamp, NoPodsBefore) instead of as a metav1.Time
	// pods.offset: Identical to NotBefore, but specified as an offset from CreationTimestamp instead of as a metav1.Time
	// +nullable
	Offset DisruptionDuration `json:"offset,omitempty"`
}

func (dt DisruptionTrigger) IsZero() bool {
	return dt.NotBefore.IsZero() && dt.Offset.Duration() == 0
}

// Reporting provides additional reporting options in order to send a message to a custom slack channel
// it expects the main controller to have the slack notifier enabled
// it expects a slack bot to be added to the defined slack channel
type Reporting struct {
	// SlackChannel is the destination slack channel to send reporting informations to.
	// It's expected to follow slack naming conventions https://api.slack.com/methods/conversations.create#naming or slack channel ID format
	// +kubebuilder:validation:MaxLength=80
	// +kubebuilder:validation:Pattern=(^[a-z0-9-_]+$)|(^C[A-Z0-9]+$)
	// +kubebuilder:validation:Required
	// +ddmark:validation:Required=true
	SlackChannel string `json:"slackChannel,omitempty"`
	// Purpose determines contextual informations about the disruption
	// a brief context to determines disruption goal
	// +kubebuilder:validation:MinLength=10
	// +kubebuilder:validation:Required
	// +ddmark:validation:Required=true
	Purpose string `json:"purpose,omitempty"`
	// MinNotificationType is the minimal notification type we want to receive informations for
	// In order of importance it's Info, Success, Warning, Error
	// Default level is considered Success, meaning all info will be ignored
	MinNotificationType eventtypes.NotificationType `json:"minNotificationType,omitempty"`
}

// EmbeddedChaosAPI includes the library so it can be statically exported to chaosli
//
//go:embed *.go
var EmbeddedChaosAPI embed.FS

type DisruptionDuration string

func (dd DisruptionDuration) MarshalJSON() ([]byte, error) {
	if dd == "" {
		return json.Marshal("")
	}

	d, err := time.ParseDuration(string(dd))
	if err != nil {
		return nil, err
	}

	return json.Marshal(d.String())
}

func (dd *DisruptionDuration) UnmarshalJSON(data []byte) error {
	var v interface{}

	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	switch value := v.(type) {
	case float64:
		*dd = DisruptionDuration(time.Duration(value).String())
		return nil
	case string:
		d, err := time.ParseDuration(value)
		*dd = DisruptionDuration(d.String())

		if err != nil {
			return err
		}

		return nil
	default:
		return errors.New("invalid duration")
	}
}

func (dd DisruptionDuration) Duration() time.Duration {
	// This was validated at unmarshal time, so we can ignore the error
	d, _ := time.ParseDuration(string(dd))

	return d
}

type TargetInjection struct {
	InjectorPodName string `json:"injectorPodName,omitempty"`
	// +kubebuilder:validation:Enum=NotInjected;Injected;IsStuckOnRemoval
	// +ddmark:validation:Enum=NotInjected;Injected;IsStuckOnRemoval
	InjectionStatus chaostypes.DisruptionTargetInjectionStatus `json:"injectionStatus,omitempty"`
	// since when this status is in place
	Since metav1.Time `json:"since,omitempty"`
}

// TargetInjections map of target injection
type TargetInjections map[string]TargetInjection

// GetTargetNames return the name of targets
func (in TargetInjections) GetTargetNames() []string {
	names := make([]string, 0, len(in))

	for targetName := range in {
		names = append(names, targetName)
	}

	return names
}

// DisruptionStatus defines the observed state of Disruption
type DisruptionStatus struct {
	IsStuckOnRemoval bool `json:"isStuckOnRemoval,omitempty"`
	IsInjected       bool `json:"isInjected,omitempty"`
	// +kubebuilder:validation:Enum=NotInjected;PartiallyInjected;PausedPartiallyInjected;Injected;PausedInjected;PreviouslyNotInjected;PreviouslyPartiallyInjected;PreviouslyInjected
	// +ddmark:validation:Enum=NotInjected;PartiallyInjected;PausedPartiallyInjected;Injected;PausedInjected;PreviouslyNotInjected;PreviouslyPartiallyInjected;PreviouslyInjected
	InjectionStatus chaostypes.DisruptionInjectionStatus `json:"injectionStatus,omitempty"`
	// +nullable
	TargetInjections TargetInjections `json:"targetInjections,omitempty"`
	// Actual targets selected by the disruption
	SelectedTargetsCount int `json:"selectedTargetsCount"`
	// Targets ignored by the disruption, (not in a ready state, already targeted, not in the count percentage...)
	IgnoredTargetsCount int `json:"ignoredTargetsCount"`
	// Number of targets with a chaos pod ready
	InjectedTargetsCount int `json:"injectedTargetsCount"`
	// Number of targets we want to target (count)
	DesiredTargetsCount int `json:"desiredTargetsCount"`
}

type DisruptionFilter struct {
	Annotations labels.Set `json:"annotations,omitempty"`
}

//+kubebuilder:object:root=true

// Disruption is the Schema for the disruptions API
// +kubebuilder:resource:shortName=dis
// +kubebuilder:subresource:status
type Disruption struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DisruptionSpec   `json:"spec,omitempty"`
	Status DisruptionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DisruptionList contains a list of Disruption
type DisruptionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Disruption `json:"items"`
}

// DisruptionPulse contains the active disruption duration and the dormant disruption duration
// +ddmark:validation:LinkedFields={ActiveDuration,DormantDuration}
type DisruptionPulse struct {
	ActiveDuration  DisruptionDuration `json:"activeDuration,omitempty"`
	DormantDuration DisruptionDuration `json:"dormantDuration,omitempty"`
	InitialDelay    DisruptionDuration `json:"initialDelay,omitempty"`
}

func init() {
	SchemeBuilder.Register(&Disruption{}, &DisruptionList{})
}

func (s *DisruptionSpec) UnmarshalJSON(data []byte) error {
	type innerSpec DisruptionSpec

	// Unmarshalling does not consider tag default
	// as we want all manifest to have a default value if undefined, we hence set it here
	inner := &innerSpec{Level: chaostypes.DisruptionLevelPod}
	if err := json.Unmarshal(data, &inner); err != nil {
		return err
	}

	*s = DisruptionSpec(*inner)

	return nil
}

// Hash returns the disruption spec JSON hash
func (s DisruptionSpec) Hash() (string, error) {
	// serialize instance spec to JSON
	specBytes, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("error serializing instance spec: %w", err)
	}

	// compute bytes hash
	return fmt.Sprintf("%x", md5.Sum(specBytes)), nil
}

func (s DisruptionSpec) HashNoCount() (string, error) {
	s.Count = nil

	return s.Hash()
}

// Validate applies rules for disruption global scope and all subsequent disruption specifications
func (s DisruptionSpec) Validate() (retErr error) {
	if err := s.validateGlobalDisruptionScope(); err != nil {
		retErr = multierror.Append(retErr, err)
	}

	for _, kind := range chaostypes.DisruptionKindNames {
		disruptionKind := s.DisruptionKindPicker(kind)
		if reflect.ValueOf(disruptionKind).IsNil() {
			continue
		}

		if err := disruptionKind.Validate(); err != nil {
			retErr = multierror.Append(retErr, err)
		}
	}

	return multierror.Prefix(retErr, "Spec:")
}

func AdvancedSelectorsToRequirements(advancedSelectors []metav1.LabelSelectorRequirement) ([]labels.Requirement, error) {
	reqs := []labels.Requirement{}
	for _, req := range advancedSelectors {
		var op selection.Operator

		// parse the operator to convert it from one package to another
		switch req.Operator {
		case metav1.LabelSelectorOpIn:
			op = selection.In
		case metav1.LabelSelectorOpNotIn:
			op = selection.NotIn
		case metav1.LabelSelectorOpExists:
			op = selection.Exists
		case metav1.LabelSelectorOpDoesNotExist:
			op = selection.DoesNotExist
		default:
			return nil, fmt.Errorf("error parsing advanced selector operator %s: must be either In, NotIn, Exists or DoesNotExist", req.Operator)
		}

		// generate and add the requirement to the selector
		parsedReq, err := labels.NewRequirement(req.Key, op, req.Values)
		if err != nil {
			return nil, fmt.Errorf("error parsing given advanced selector to requirements: %w", err)
		}

		reqs = append(reqs, *parsedReq)
	}

	return reqs, nil
}

// Validate applies rules for disruption global scope
func (s DisruptionSpec) validateGlobalDisruptionScope() (retErr error) {
	// Rule: at least one kind of selector is set
	if s.Selector.AsSelector().Empty() && len(s.AdvancedSelector) == 0 {
		retErr = multierror.Append(retErr, errors.New("either selector or advancedSelector field must be set"))
	}

	// Rule: selectors must be valid
	if !s.Selector.AsSelector().Empty() {
		_, err := labels.ParseToRequirements(s.Selector.AsSelector().String())
		if err != nil {
			retErr = multierror.Append(retErr, err)
		}
	}

	if len(s.AdvancedSelector) > 0 {
		_, err := AdvancedSelectorsToRequirements(s.AdvancedSelector)
		if err != nil {
			retErr = multierror.Append(retErr, err)
		}
	}

	// Rule: no targeted container if disruption is node-level
	if len(s.Containers) > 0 && s.Level == chaostypes.DisruptionLevelNode {
		retErr = multierror.Append(retErr, errors.New("cannot target specific containers because the level configuration is set to node"))
	}

	// Rule: container failure not possible if disruption is node-level
	if s.ContainerFailure != nil && s.Level == chaostypes.DisruptionLevelNode {
		retErr = multierror.Append(retErr, errors.New("cannot execute a container failure because the level configuration is set to node"))
	}

	// Rule: on init compatibility
	if s.OnInit {
		if s.CPUPressure != nil ||
			s.NodeFailure != nil ||
			s.ContainerFailure != nil ||
			s.DiskPressure != nil ||
			s.GRPC != nil ||
			s.DiskFailure != nil {
			retErr = multierror.Append(retErr, errors.New("OnInit is only compatible with network and dns disruptions"))
		}

		if s.DNS != nil && len(s.Containers) > 0 {
			retErr = multierror.Append(retErr, errors.New("OnInit is only compatible on dns disruptions with no subset of targeted containers"))
		}

		if s.Level != chaostypes.DisruptionLevelPod {
			retErr = multierror.Append(retErr, errors.New("OnInit is only compatible with pod level disruptions"))
		}

		if len(s.Containers) > 0 {
			retErr = multierror.Append(retErr, errors.New("OnInit is not compatible with containers scoping"))
		}
	}

	// Rule: No specificity of containers on a disk disruption
	if len(s.Containers) != 0 && s.DiskPressure != nil {
		retErr = multierror.Append(retErr, errors.New("disk pressure disruptions apply to all containers, specifying certain containers does not isolate the disruption"))
	}

	// Rule: DisruptionTrigger
	if !s.Triggers.IsZero() {
		if !s.Triggers.Inject.IsZero() && !s.Triggers.CreatePods.IsZero() {
			if !s.Triggers.Inject.NotBefore.IsZero() && !s.Triggers.CreatePods.NotBefore.IsZero() && s.Triggers.Inject.NotBefore.Before(&s.Triggers.CreatePods.NotBefore) {
				retErr = multierror.Append(retErr, fmt.Errorf("spec.trigger.inject.notBefore is %s, which is before your spec.trigger.createPods.notBefore of %s. inject.notBefore must come after createPods.notBefore if both are specified", s.Triggers.Inject.NotBefore, s.Triggers.CreatePods.NotBefore))
			}
		}
	}

	// Rule: pulse compatibility
	if s.Pulse != nil {
		if s.Pulse.ActiveDuration.Duration() > 0 || s.Pulse.DormantDuration.Duration() > 0 {
			if s.NodeFailure != nil || s.ContainerFailure != nil {
				retErr = multierror.Append(retErr, errors.New("pulse is only compatible with network, cpu pressure, disk pressure, dns and grpc disruptions"))
			}
		}

		if s.Pulse.ActiveDuration.Duration() != 0 && s.Pulse.ActiveDuration.Duration() < chaostypes.PulsingDisruptionMinimumDuration {
			retErr = multierror.Append(retErr, fmt.Errorf("pulse activeDuration of %s should be greater than %s", s.Pulse.ActiveDuration.Duration(), chaostypes.PulsingDisruptionMinimumDuration))
		}

		if s.Pulse.DormantDuration.Duration() != 0 && s.Pulse.DormantDuration.Duration() < chaostypes.PulsingDisruptionMinimumDuration {
			retErr = multierror.Append(retErr, fmt.Errorf("pulse dormantDuration of %s should be greater than %s", s.Pulse.DormantDuration.Duration(), chaostypes.PulsingDisruptionMinimumDuration))
		}
	}

	if s.GRPC != nil && s.Level != chaostypes.DisruptionLevelPod {
		retErr = multierror.Append(retErr, errors.New("GRPC disruptions can only be applied at the pod level"))
	}

	// Rule: count must be valid
	if err := ValidateCount(s.Count); err != nil {
		retErr = multierror.Append(retErr, err)
	}

	return retErr
}

// DisruptionKindPicker returns this DisruptionSpec's instance of a DisruptionKind based on given kind name
func (s DisruptionSpec) DisruptionKindPicker(kind chaostypes.DisruptionKindName) chaosapi.DisruptionKind {
	var disruptionKind chaosapi.DisruptionKind

	switch kind {
	case chaostypes.DisruptionKindNodeFailure:
		disruptionKind = s.NodeFailure
	case chaostypes.DisruptionKindContainerFailure:
		disruptionKind = s.ContainerFailure
	case chaostypes.DisruptionKindNetworkDisruption:
		disruptionKind = s.Network
	case chaostypes.DisruptionKindCPUPressure:
		disruptionKind = s.CPUPressure
	case chaostypes.DisruptionKindDiskPressure:
		disruptionKind = s.DiskPressure
	case chaostypes.DisruptionKindDNSDisruption:
		disruptionKind = s.DNS
	case chaostypes.DisruptionKindGRPCDisruption:
		disruptionKind = s.GRPC
	case chaostypes.DisruptionKindDiskFailure:
		disruptionKind = s.DiskFailure
	}

	return disruptionKind
}

// KindNames returns the non-nil disruption kind names for the given disruption
func (s DisruptionSpec) KindNames() []chaostypes.DisruptionKindName {
	kinds := []chaostypes.DisruptionKindName{}

	for _, kind := range chaostypes.DisruptionKindNames {
		subspec := s.DisruptionKindPicker(kind)
		if reflect.ValueOf(subspec).IsNil() {
			continue
		}

		kinds = append(kinds, kind)
	}

	return kinds
}

func ReadUnmarshal(path string) (*Disruption, error) {
	fullPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("error finding absolute path: %w", err)
	}

	yaml, err := os.Open(filepath.Clean(fullPath))
	if err != nil {
		return nil, fmt.Errorf("could not open yaml file at %s: %w", fullPath, err)
	}

	yamlBytes, err := io.ReadAll(yaml)
	if err != nil {
		return nil, fmt.Errorf("could not read yaml file: %w", err)
	}

	parsedSpec := Disruption{}

	if err = goyaml.UnmarshalStrict(yamlBytes, &parsedSpec); err != nil {
		return nil, fmt.Errorf("could not unmarshal yaml file to Disruption: %w", err)
	}

	return &parsedSpec, nil
}

// DisruptionCount get the number of disruption types per disruption
func (s DisruptionSpec) DisruptionCount() int {
	count := 0

	if s.CPUPressure != nil {
		count++
	}

	if s.ContainerFailure != nil {
		count++
	}

	if s.DNS != nil {
		count++
	}

	if s.DiskPressure != nil {
		count++
	}

	if s.GRPC != nil {
		count++
	}

	if s.Network != nil {
		count++
	}

	if s.NodeFailure != nil {
		count++
	}

	if s.DiskFailure != nil {
		count++
	}

	return count
}

// RemoveDeadTargets removes targets not found in matchingTargets from the targets list
func (status *DisruptionStatus) RemoveDeadTargets(matchingTargets []string) {
	desiredTargets := TargetInjections{}

	targetNames := status.TargetInjections.GetTargetNames()

	for _, matchingTarget := range matchingTargets {
		if utils.Contains(targetNames, matchingTarget) {
			desiredTargets[matchingTarget] = status.TargetInjections[matchingTarget]
		}
	}

	status.TargetInjections = desiredTargets
}

// AddTargets adds newTargetsCount random targets from the eligibleTargets list to the Target List
// - eligibleTargets should be previously filtered to not include current targets
func (status *DisruptionStatus) AddTargets(newTargetsCount int, eligibleTargets TargetInjections) {
	if len(eligibleTargets) == 0 || newTargetsCount <= 0 {
		return
	}

	parseRandomTargets(newTargetsCount, eligibleTargets, func(targetName string) {
		status.TargetInjections[targetName] = eligibleTargets[targetName]
		delete(eligibleTargets, targetName)
	})
}

// RemoveTargets removes toRemoveTargetsCount random targets from the Target List
func (status *DisruptionStatus) RemoveTargets(toRemoveTargetsCount int) {
	parseRandomTargets(toRemoveTargetsCount, status.TargetInjections, func(targetName string) {
		delete(status.TargetInjections, targetName)
	})
}

func parseRandomTargets(targetLimit int, targetInjections TargetInjections, callback func(targetName string)) {
	targetNames := targetInjections.GetTargetNames()

	for i := 0; i < targetLimit && len(targetNames) > 0; i++ {
		index := rand.Intn(len(targetNames)) //nolint:gosec
		targetName := targetNames[index]
		targetNames[len(targetNames)-1], targetNames[index] = targetNames[index], targetNames[len(targetNames)-1]
		targetNames = targetNames[:len(targetNames)-1]

		callback(targetName)
	}
}

// HasTarget returns true when a target exists in the Target List or returns false.
func (status *DisruptionStatus) HasTarget(searchTarget string) bool {
	_, exists := status.TargetInjections[searchTarget]
	return exists
}

var NonReinjectableDisruptions = map[chaostypes.DisruptionKindName]struct{}{
	chaostypes.DisruptionKindGRPCDisruption: {},
	chaostypes.DisruptionKindNodeFailure:    {},
}

func DisruptionIsNotReinjectable(kind chaostypes.DisruptionKindName) bool {
	_, found := NonReinjectableDisruptions[kind]

	return found
}

// NoSideEffectDisruptions is the list of all disruption kinds where the lifecycle of the failure matches the lifecycle of
// the chaos pod. So once the chaos pod is gone, there's nothing left for us to clean.
var NoSideEffectDisruptions = map[chaostypes.DisruptionKindName]struct{}{
	chaostypes.DisruptionKindNodeFailure:      {},
	chaostypes.DisruptionKindContainerFailure: {},
}

func DisruptionHasNoSideEffects(kind string) bool {
	_, found := NoSideEffectDisruptions[chaostypes.DisruptionKindName(kind)]

	return found
}

// ShouldSkipNodeFailureInjection returns true if we are attempting to inject a node failure that has already been injected for this given target
// If we're using staticTargeting, we should never re-select a target whose InjectionStatus is anything other than NotInjected, as we may be
// injecting into a pod that has been rescheduled onto a new node
func ShouldSkipNodeFailureInjection(disKind chaostypes.DisruptionKindName, instance *Disruption, injection TargetInjection) bool {
	// we should never re-inject a static node failure, as it may be targeting the same pod on a new node
	return disKind == chaostypes.DisruptionKindNodeFailure && instance.Spec.StaticTargeting && injection.InjectionStatus != chaostypes.DisruptionTargetInjectionStatusNotInjected
}

// TargetedContainers returns a map of containers with containerName as a key and containerID in the format '<type>://<container_id>' as a value
func TargetedContainers(pod corev1.Pod, containerNames []string) (map[string]string, error) {
	if len(pod.Status.ContainerStatuses) < 1 {
		return nil, fmt.Errorf("missing container ids for pod '%s'", pod.Name)
	}

	podContainers := make([]corev1.ContainerStatus, 0, len(pod.Status.ContainerStatuses)+len(pod.Status.InitContainerStatuses))
	podContainers = append(podContainers, pod.Status.ContainerStatuses...)
	podContainers = append(podContainers, pod.Status.InitContainerStatuses...)

	if len(containerNames) == 0 {
		allContainers := make(map[string]string, len(podContainers))

		for _, c := range podContainers {
			if c.State.Running != nil {
				allContainers[c.Name] = c.ContainerID
			}
		}

		return allContainers, nil
	}

	allContainers := make(map[string]string, len(podContainers))
	targetedContainers := make(map[string]string, len(containerNames))

	for _, c := range podContainers {
		allContainers[c.Name] = c.ContainerID
	}

	// look for the target in the map
	for _, containerName := range containerNames {
		if containerID, existsInPod := allContainers[containerName]; existsInPod {
			targetedContainers[containerName] = containerID
		} else {
			return nil, fmt.Errorf("could not find specified container in pod (pod: %s, target: %s)", pod.ObjectMeta.Name, containerName)
		}
	}

	return targetedContainers, nil
}
