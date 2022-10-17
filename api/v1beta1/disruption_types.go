// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package v1beta1

import (
	"crypto/md5"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"time"

	chaosapi "github.com/DataDog/chaos-controller/api"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"github.com/DataDog/chaos-controller/utils"
	"github.com/hashicorp/go-multierror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	goyaml "sigs.k8s.io/yaml"
)

// DisruptionSpec defines the desired state of Disruption
// +ddmark:validation:ExclusiveFields={ContainerFailure,CPUPressure,DiskPressure,NodeFailure,Network,DNS}
// +ddmark:validation:ExclusiveFields={NodeFailure,CPUPressure,DiskPressure,ContainerFailure,Network,DNS}
// +ddmark:validation:AtLeastOneOf={DNS,CPUPressure,Network,NodeFailure,ContainerFailure,DiskPressure,GRPC}
// +ddmark:validation:AtLeastOneOf={Selector,AdvancedSelector}
type DisruptionSpec struct {
	// +kubebuilder:validation:Required
	// +ddmark:validation:Required=true
	Count *intstr.IntOrString `json:"count"` // number of pods to target in either integer form or percent form appended with a %
	// +nullable
	Selector labels.Set `json:"selector,omitempty"` // label selector
	// +nullable
	AdvancedSelector []metav1.LabelSelectorRequirement `json:"advancedSelector,omitempty"` // advanced label selector
	DryRun           bool                              `json:"dryRun,omitempty"`           // enable dry-run mode
	OnInit           bool                              `json:"onInit,omitempty"`           // enable disruption on init
	Unsafemode       *UnsafemodeSpec                   `json:"unsafeMode,omitempty"`       // unsafemode spec used to turn off safemode safety nets
	StaticTargeting  bool                              `json:"staticTargeting,omitempty"`  // enable dynamic targeting and cluster observation
	// +nullable
	Pulse    *DisruptionPulse   `json:"pulse,omitempty"`    // enable pulsing diruptions and specify the duration of the active state and the dormant state of the pulsing duration
	Duration DisruptionDuration `json:"duration,omitempty"` // time from disruption creation until chaos pods are deleted and no more are created
	// +kubebuilder:validation:Enum=pod;node;""
	// +ddmark:validation:Enum=pod;node;""
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
	DNS DNSDisruptionSpec `json:"dns,omitempty"`
	// +nullable
	GRPC *GRPCDisruptionSpec `json:"grpc,omitempty"`
}

// EmbeddedChaosAPI includes the library so it can be statically exported to chaosli
//
//go:embed *
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

// DisruptionStatus defines the observed state of Disruption
type DisruptionStatus struct {
	IsStuckOnRemoval bool `json:"isStuckOnRemoval,omitempty"`
	IsInjected       bool `json:"isInjected,omitempty"`
	// +kubebuilder:validation:Enum=NotInjected;PartiallyInjected;Injected;PreviouslyInjected
	// +ddmark:validation:Enum=NotInjected;PartiallyInjected;Injected;PreviouslyInjected
	InjectionStatus chaostypes.DisruptionInjectionStatus `json:"injectionStatus,omitempty"`
	// +nullable
	Targets []string `json:"targets,omitempty"`
	// Actual targets selected by the disruption
	SelectedTargetsCount int `json:"selectedTargetsCount"`
	// Targets ignored by the disruption, (not in a ready state, already targeted, not in the count percentage...)
	IgnoredTargetsCount int `json:"ignoredTargetsCount"`
	// Number of targets with a chaos pod ready
	InjectedTargetsCount int `json:"injectedTargetsCount"`
	// Number of targets we want to target (count)
	DesiredTargetsCount int `json:"desiredTargetsCount"`
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
type DisruptionPulse struct {
	ActiveDuration  DisruptionDuration `json:"activeDuration"`
	DormantDuration DisruptionDuration `json:"dormantDuration"`
}

func init() {
	SchemeBuilder.Register(&Disruption{}, &DisruptionList{})
}

// Hash returns the disruption spec JSON hash
func (s *DisruptionSpec) Hash() (string, error) {
	// serialize instance spec to JSON
	specBytes, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("error serializing instance spec: %w", err)
	}

	// compute bytes hash
	return fmt.Sprintf("%x", md5.Sum(specBytes)), nil
}

func (s *DisruptionSpec) HashNoCount() (string, error) {
	sCopy := s.DeepCopy()
	sCopy.Count = nil

	return sCopy.Hash()
}

// Validate applies rules for disruption global scope and all subsequent disruption specifications
func (s *DisruptionSpec) Validate() (retErr error) {
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

// Validate applies rules for disruption global scope
func (s *DisruptionSpec) validateGlobalDisruptionScope() (retErr error) {
	// Rule: at least one kind of selector is set
	if s.Selector.AsSelector().Empty() && len(s.AdvancedSelector) == 0 {
		retErr = multierror.Append(retErr, errors.New("either selector or advancedSelector field must be set"))
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
			s.GRPC != nil {
			retErr = multierror.Append(retErr, errors.New("OnInit is only compatible with network and dns disruptions"))
		}

		if s.DNS != nil && len(s.Containers) > 0 {
			retErr = multierror.Append(retErr, errors.New("OnInit is only compatible on dns disruptions with no subset of targeted containers"))
		}

		if s.Level != chaostypes.DisruptionLevelPod && s.Level != chaostypes.DisruptionLevelUnspecified {
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

	// Rule: pulse compatibility
	if s.Pulse != nil {
		if s.NodeFailure != nil || s.ContainerFailure != nil {
			retErr = multierror.Append(retErr, errors.New("pulse is only compatible with network, cpu pressure, disk pressure, dns and grpc disruptions"))
		}

		if s.Pulse.ActiveDuration.Duration() < chaostypes.PulsingDisruptionMinimumDuration {
			retErr = multierror.Append(retErr, fmt.Errorf("pulse activeDuration should be greater than %s", chaostypes.PulsingDisruptionMinimumDuration))
		}

		if s.Pulse.DormantDuration.Duration() < chaostypes.PulsingDisruptionMinimumDuration {
			retErr = multierror.Append(retErr, fmt.Errorf("pulse dormantDuration should be greater than %s", chaostypes.PulsingDisruptionMinimumDuration))
		}
	}

	if s.GRPC != nil && s.Level != chaostypes.DisruptionLevelPod && s.Level != chaostypes.DisruptionLevelUnspecified {
		retErr = multierror.Append(retErr, errors.New("GRPC disruptions can only be applied at the pod level"))
	}

	// Rule: count must be valid
	if err := ValidateCount(s.Count); err != nil {
		retErr = multierror.Append(retErr, err)
	}

	return retErr
}

// DisruptionKindPicker returns this DisruptionSpec's instance of a DisruptionKind based on given kind name
func (s *DisruptionSpec) DisruptionKindPicker(kind chaostypes.DisruptionKindName) chaosapi.DisruptionKind {
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
	}

	return disruptionKind
}

// GetKindNames returns the non-nil disruption kind names for the given disruption
func (s *DisruptionSpec) GetKindNames() []chaostypes.DisruptionKindName {
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
		return nil, fmt.Errorf("error finding absolute path: %v", err)
	}

	yaml, err := os.Open(filepath.Clean(fullPath))
	if err != nil {
		return nil, fmt.Errorf("could not open yaml file at %s: %v", fullPath, err)
	}

	yamlBytes, err := ioutil.ReadAll(yaml)
	if err != nil {
		return nil, fmt.Errorf("could not read yaml file: %v ", err)
	}

	parsedSpec := Disruption{}

	if err = goyaml.UnmarshalStrict(yamlBytes, &parsedSpec); err != nil {
		return nil, fmt.Errorf("could not unmarshal yaml file to Disruption: %v", err)
	}

	return &parsedSpec, nil
}

// GetDisruptionCount get the number of disruption types per disruption
func (s *DisruptionSpec) GetDisruptionCount() int {
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

	return count
}

// RemoveDeadTargets removes targets not found in matchingTargets from the targets list
func (status *DisruptionStatus) RemoveDeadTargets(matchingTargets []string) {
	var desiredTargets []string

	for index := 0; index < len(status.Targets); index++ {
		if utils.Contains(matchingTargets, status.Targets[index]) {
			desiredTargets = append(desiredTargets, status.Targets[index])
		}
	}

	status.Targets = desiredTargets
}

// AddTargets adds newTargetsCount random targets from the eligibleTargets list to the Target List
// - eligibleTargets should be previously filtered to not include current targets
func (status *DisruptionStatus) AddTargets(newTargetsCount int, eligibleTargets []string) {
	if len(eligibleTargets) == 0 || newTargetsCount <= 0 {
		return
	}

	for i := 0; i < newTargetsCount && len(eligibleTargets) > 0; i++ {
		index := rand.Intn(len(eligibleTargets)) //nolint:gosec
		status.Targets = append(status.Targets, eligibleTargets[index])
		eligibleTargets[len(eligibleTargets)-1], eligibleTargets[index] = eligibleTargets[index], eligibleTargets[len(eligibleTargets)-1]
		eligibleTargets = eligibleTargets[:len(eligibleTargets)-1]
	}
}

// RemoveTargets removes toRemoveTargetsCount random targets from the Target List
func (status *DisruptionStatus) RemoveTargets(toRemoveTargetsCount int) {
	for i := 0; i < toRemoveTargetsCount && len(status.Targets) > 0; i++ {
		index := rand.Intn(len(status.Targets)) //nolint:gosec
		status.Targets[len(status.Targets)-1], status.Targets[index] = status.Targets[index], status.Targets[len(status.Targets)-1]
		status.Targets = status.Targets[:len(status.Targets)-1]
	}
}

// HasTarget returns true when a target exists in the Target List or returns false.
func (status *DisruptionStatus) HasTarget(searchTarget string) bool {
	for _, target := range status.Targets {
		if target == searchTarget {
			return true
		}
	}

	return false
}

var NonReinjectableDisruptions = map[chaostypes.DisruptionKindName]struct{}{
	chaostypes.DisruptionKindGRPCDisruption: {},
}

func DisruptionIsReinjectable(kind chaostypes.DisruptionKindName) bool {
	_, found := NonReinjectableDisruptions[kind]

	return found
}

// NoSideEffectDisruptions is the list of all disruption kinds where the lifecycle of the failure matches the lifecycle of
// the chaos pod. So once the chaos pod is gone, there's nothing left for us to clean.
var NoSideEffectDisruptions = map[chaostypes.DisruptionKindName]struct{}{
	chaostypes.DisruptionKindNodeFailure:      {},
	chaostypes.DisruptionKindContainerFailure: {},
	chaostypes.DisruptionKindCPUPressure:      {},
}

func DisruptionHasNoSideEffects(kind string) bool {
	_, found := NoSideEffectDisruptions[chaostypes.DisruptionKindName(kind)]

	return found
}
