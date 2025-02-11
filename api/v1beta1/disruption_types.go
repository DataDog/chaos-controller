// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

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
	"strings"
	"time"

	chaosapi "github.com/DataDog/chaos-controller/api"
	eventtypes "github.com/DataDog/chaos-controller/eventnotifier/types"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"github.com/DataDog/chaos-controller/utils"
	"github.com/hashicorp/go-multierror"
	authv1beta1 "k8s.io/api/authentication/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/intstr"
	goyaml "sigs.k8s.io/yaml"
)

// DisruptionSpec defines the desired state of Disruption
type DisruptionSpec struct {
	// +kubebuilder:validation:Required
	Count *intstr.IntOrString `json:"count" chaos_validate:"required"` // number of pods to target in either integer form or percent form appended with a %
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
	Triggers *DisruptionTriggers `json:"triggers,omitempty"` // alter the pre-injection lifecycle
	// +nullable
	Pulse    *DisruptionPulse   `json:"pulse,omitempty"`    // enable pulsing diruptions and specify the duration of the active state and the dormant state of the pulsing duration
	Duration DisruptionDuration `json:"duration,omitempty"` // time from disruption creation until chaos pods are deleted and no more are created
	// Level defines what the disruption will target, either a pod or a node
	// +kubebuilder:default=pod
	// +kubebuilder:validation:Enum=pod;node
	Level      chaostypes.DisruptionLevel `json:"level,omitempty" chaos_validate:"omitempty,oneofci=pod node"`
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

type TerminationStatus uint8

const (
	TSNotTerminated TerminationStatus = iota
	TSTemporarilyTerminated
	TSDefinitivelyTerminated
)

func (dt DisruptionTriggers) IsZero() bool {
	return dt.Inject.IsZero() && dt.CreatePods.IsZero()
}

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
	SlackChannel string `json:"slackChannel,omitempty" chaos_validate:"required"`
	// Purpose determines contextual informations about the disruption
	// a brief context to determines disruption goal
	// +kubebuilder:validation:MinLength=10
	// +kubebuilder:validation:Required
	Purpose string `json:"purpose,omitempty"`
	// MinNotificationType is the minimal notification type we want to receive informations for
	// In order of importance it's Info, Success, Warning, Error
	// Default level is considered Success, meaning all info will be ignored
	MinNotificationType eventtypes.NotificationType `json:"minNotificationType,omitempty"`
}

func (r *Reporting) Explain() string {
	return fmt.Sprintf("While the disruption is ongoing, it will send slack messages for every event of severity %s or higher, "+
		"to the slack channel with the ID (not name) %s, mentioning the purpose \"%s\"",
		r.MinNotificationType,
		r.SlackChannel,
		r.Purpose)
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
	InjectionStatus chaostypes.DisruptionTargetInjectionStatus `json:"injectionStatus,omitempty"`
	// since when this status is in place
	Since metav1.Time `json:"since,omitempty"`
}

type TargetInjectorMap map[chaostypes.DisruptionKindName]TargetInjection

// TargetInjections is a map of target names to injectors
type TargetInjections map[string]TargetInjectorMap

// GetInjectionWithDisruptionKind retrieves a TargetInjection associated with a specific DisruptionKindName from the map.
//
// Parameters:
//   - kindName: The DisruptionKindName to search for in the map.
//
// Returns:
//   - *TargetInjection: A pointer to the TargetInjection with the matching DisruptionKindName, or nil if not found.
func (m TargetInjectorMap) GetInjectionWithDisruptionKind(kindName chaostypes.DisruptionKindName) *TargetInjection {
	if targetInjection, exists := m[kindName]; exists {
		return &targetInjection
	}

	return nil
}

// GetTargetNames return the name of targets
func (in TargetInjections) GetTargetNames() []string {
	names := make([]string, 0, len(in))

	for targetName := range in {
		names = append(names, targetName)
	}

	return names
}

// NotFullyInjected checks if any of the TargetInjections in the list are not fully injected.
//
// Returns:
//   - bool: true if any TargetInjection is not fully injected, false if all are fully injected or the list is empty.
func (in TargetInjections) NotFullyInjected() bool {
	if len(in) == 0 {
		return true
	}

	for _, injectors := range in {
		for _, i := range injectors {
			if i.InjectionStatus != chaostypes.DisruptionTargetInjectionStatusInjected {
				return true
			}
		}
	}

	return false
}

// DisruptionStatus defines the observed state of Disruption
type DisruptionStatus struct {
	IsStuckOnRemoval bool `json:"isStuckOnRemoval,omitempty"`
	IsInjected       bool `json:"isInjected,omitempty"`
	// +kubebuilder:validation:Enum=NotInjected;PartiallyInjected;PausedPartiallyInjected;Injected;PausedInjected;PreviouslyNotInjected;PreviouslyPartiallyInjected;PreviouslyInjected
	// +kubebuilder:default=NotInjected
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

	// timestamp of when a disruption has been cleaned last.
	// +nullable
	CleanedAt *metav1.Time `json:"cleanedAt,omitempty"`
}

type DisruptionFilter struct {
	Annotations labels.Set `json:"annotations,omitempty"`
}

//+kubebuilder:object:root=true

// Disruption is the Schema for the disruptions API
// +kubebuilder:resource:shortName=dis
// +kubebuilder:subresource:status
// +genclient
// +genclient:noStatus
// +genclient:onlyVerbs=create,get,list,delete,watch,update
type Disruption struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DisruptionSpec   `json:"spec,omitempty"`
	Status DisruptionStatus `json:"status,omitempty"`
}

// TimeToInject calculates the time at which the disruption should be injected based on its own creationTimestamp.
// It considers the specified triggers for injection timing in the disruption's specification.
func (r *Disruption) TimeToInject() time.Time {
	triggers := r.Spec.Triggers

	if triggers == nil || triggers.IsZero() {
		return r.CreationTimestamp.Time
	}

	if triggers.Inject.IsZero() {
		return r.TimeToCreatePods()
	}

	var notInjectedBefore time.Time

	// validation should have already prevented a situation where both Offset and NotBefore are set
	if !triggers.Inject.NotBefore.IsZero() {
		notInjectedBefore = triggers.Inject.NotBefore.Time
	}

	if triggers.Inject.Offset.Duration() > 0 {
		// We measure the offset from the latter of two timestamps: creationTimestamp of the disruption, and spec.triggers.createPods
		notInjectedBefore = r.TimeToCreatePods().Add(triggers.Inject.Offset.Duration())
	}

	if r.CreationTimestamp.Time.After(notInjectedBefore) {
		return r.CreationTimestamp.Time
	}

	return notInjectedBefore
}

// TimeToCreatePods takes the DisruptionTriggers field from a Disruption spec, along with the time.Time at which that disruption was created
// It returns the earliest time.Time at which the chaos-controller should begin creating chaos pods, given the specified DisruptionTriggers
func (r *Disruption) TimeToCreatePods() time.Time {
	triggers := r.Spec.Triggers

	if triggers == nil || triggers.IsZero() {
		return r.CreationTimestamp.Time
	}

	if triggers.CreatePods.IsZero() {
		return r.CreationTimestamp.Time
	}

	var noPodsBefore time.Time

	// validation should have already prevented a situation where both Offset and NotBefore are set
	if !triggers.CreatePods.NotBefore.IsZero() {
		noPodsBefore = triggers.CreatePods.NotBefore.Time
	}

	if triggers.CreatePods.Offset.Duration() > 0 {
		noPodsBefore = r.CreationTimestamp.Add(triggers.CreatePods.Offset.Duration())
	}

	if r.CreationTimestamp.After(noPodsBefore) {
		return r.CreationTimestamp.Time
	}

	return noPodsBefore
}

// RemainingDuration return the remaining duration of the disruption.
func (r *Disruption) RemainingDuration() time.Duration {
	return r.calculateDeadline(
		r.Spec.Duration.Duration(),
		r.TimeToInject(),
	)
}

func (r *Disruption) calculateDeadline(duration time.Duration, creationTime time.Time) time.Duration {
	// first we must calculate the timeout from when the disruption was created, not from now
	timeout := creationTime.Add(duration)
	now := time.Now() // rather not take the risk that the time changes by a second during this function

	// return the number of seconds between now and the deadline
	return timeout.Sub(now)
}

// TerminationStatus determines the termination status of a disruption based on various factors.
func (r *Disruption) TerminationStatus(chaosPods []corev1.Pod) TerminationStatus {
	// a not yet created disruption is neither temporarily nor definitively ended
	if r.CreationTimestamp.IsZero() {
		return TSNotTerminated
	}

	// a definitive state (expired duration or deletion) imply a definitively deleted injection
	// and should be returned prior to a temporarily terminated state
	if r.RemainingDuration() <= 0 || !r.DeletionTimestamp.IsZero() {
		return TSDefinitivelyTerminated
	}

	if len(chaosPods) == 0 {
		// we were never injected, we are hence not terminated if we reach here
		if r.Status.InjectionStatus.NeverInjected() {
			return TSNotTerminated
		}

		// we were injected before hence temporarily not terminated
		return TSTemporarilyTerminated
	}

	// if all pods exited successfully, we can consider the disruption is ended already
	// it can be caused by either an appromixative date sync (in a distributed infra it's hard)
	// or by deletion of targets leading to deletion of injectors
	// injection terminated with an error are considered NOT terminated
	for _, chaosPod := range chaosPods {
		for _, containerStatuses := range chaosPod.Status.ContainerStatuses {
			if containerStatuses.State.Terminated == nil || containerStatuses.State.Terminated.ExitCode != 0 {
				return TSNotTerminated
			}
		}
	}

	// this MIGHT be a temporary status, that could become definitive once disruption is expired or deleted
	return TSTemporarilyTerminated
}

// GetTargetsCountAsInt This function returns a scaled value from the spec.Count IntOrString type. If the count
// // is a percentage string value it's treated as a percentage and scaled appropriately
// // in accordance to the total, if it's an int value it's treated as a simple value and
// // if it is a string value which is either non-numeric or numeric but lacking a trailing '%' it returns an error.
func (r *Disruption) GetTargetsCountAsInt(targetTotal int, roundUp bool) (int, error) {
	if r.Spec.Count == nil {
		return 0, apierrors.NewBadRequest("nil value for IntOrString")
	}

	return intstr.GetScaledValueFromIntOrPercent(r.Spec.Count, targetTotal, roundUp)
}

// IsDeletionExpired checks if a Disruption resource has exceeded a specified deletion timeout duration
// for deletion. It returns true if the resource should be considered deleted based on
// the DeletionTimestamp and the deletion timeout duration.
func (r *Disruption) IsDeletionExpired(deletionTimeout time.Duration) bool {
	if r.DeletionTimestamp.IsZero() {
		return false
	}

	return time.Now().After(r.DeletionTimestamp.Add(deletionTimeout))
}

// IsReadyToRemoveFinalizer checks if a disruption has been cleaned and has waited for finalizerDelay duration before removing finalizer
func (r *Disruption) IsReadyToRemoveFinalizer(finalizerDelay time.Duration) bool {
	return r.Status.CleanedAt != nil && time.Now().After(r.Status.CleanedAt.Add(finalizerDelay))
}

// CopyOwnerAnnotations copies the annotations from the owner object to the disruption.
// This ensures that any important metadata from the owner, such as custom annotations,
// is preserved in the newly created disruption.
func (r *Disruption) CopyOwnerAnnotations(owner metav1.Object) {
	if r.Annotations == nil {
		r.Annotations = make(map[string]string)
	}

	ownerAnnotations := owner.GetAnnotations()
	for k, v := range ownerAnnotations {
		r.Annotations[k] = v
	}
}

// SetScheduledAtAnnotation sets the scheduled time of the disruption in the annotations.
func (r *Disruption) SetScheduledAtAnnotation(scheduledTime time.Time) {
	if r.Annotations == nil {
		r.Annotations = make(map[string]string)
	}

	scheduledAt := scheduledTime.Format(time.RFC3339)
	r.Annotations[chaostypes.ScheduledAtAnnotation] = scheduledAt
}

// GetScheduledAtAnnotation retrieves the scheduled time from the disruption's annotations.
// Returns an error if the annotation is not found or cannot be parsed.
func (r *Disruption) GetScheduledAtAnnotation() (time.Time, error) {
	scheduledAt, exists := r.Annotations[chaostypes.ScheduledAtAnnotation]
	if !exists {
		return time.Time{}, errors.New("scheduledAt annotation not found")
	}

	scheduledTime, err := time.Parse(time.RFC3339, scheduledAt)
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to parse scheduledAt annotation: %w", err)
	}

	return scheduledTime, nil
}

// CopyUserInfoToAnnotations copies the user-related annotations from the owner object to the disruption.
// Any UserInfo annotations will be overwritten when the Disruption is created, so this function ensures
// that the parent resource's user information is preserved by storing it in separate annotations.
func (r *Disruption) CopyUserInfoToAnnotations(owner metav1.Object) error {
	if r.Annotations == nil {
		r.Annotations = make(map[string]string)
	}

	ownerAnnotations := owner.GetAnnotations()
	if userInfoJSON, exists := ownerAnnotations["UserInfo"]; exists {
		var userInfo authv1beta1.UserInfo
		if err := json.Unmarshal([]byte(userInfoJSON), &userInfo); err != nil {
			return fmt.Errorf("unable to parse UserInfo annotation: %w", err)
		}

		// Set user-related annotations using the parsed UserInfo struct
		r.Annotations[chaostypes.UserAnnotation] = userInfo.Username
		r.Annotations[chaostypes.UserGroupsAnnotation] = strings.Join(userInfo.Groups, ",")
	}

	return nil
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

// SetDefaults finds any unset spec options, and sets the default values. Only operates on values that lack
// kubebuilder struct default tags, which typically only includes complex types, such as spec.duration
func (s *DisruptionSpec) SetDefaults() {
	if s.Duration.Duration() == 0 {
		s.Duration = DisruptionDuration(defaultDuration.String())
	}
}

// Validate applies rules for disruption global scope and all subsequent disruption specifications, requiring selectors
// intended to be called when DisruptionSpec belongs directly to a Disruption
// also exists for backwards compatibility
func (s DisruptionSpec) Validate() (retErr error) {
	return s.ValidateSelectorsOptional(true)
}

// ValidateSelectorsOptional applies rules for disruption global scope and all subsequent disruption specifications
func (s DisruptionSpec) ValidateSelectorsOptional(requireSelectors bool) (retErr error) {
	if err := s.validateGlobalDisruptionScope(requireSelectors); err != nil {
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

	if err := validateStructTags(s); err != nil {
		retErr = multierror.Append(retErr, err)
	}

	return multierror.Prefix(retErr, "Spec:")
}

// AdvancedSelectorsToRequirements converts a slice of LabelSelectorRequirements into a slice of Requirements
// and returns an error if any of those LabelSelectorRequirements are invalid. It's used for translating
// user specified advanced selectors into real requirements for selecting targets
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

// validateGlobalDisruptionScope applies rules for disruption global scope, leaving selectors optional
func (s DisruptionSpec) validateGlobalDisruptionScope(requireSelectors bool) (retErr error) {
	if requireSelectors {
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
	}

	// Rule: no targeted container if disruption is node-level
	if len(s.Containers) > 0 && s.Level == chaostypes.DisruptionLevelNode {
		retErr = multierror.Append(retErr, errors.New("cannot target specific containers because the level configuration is set to node"))
	}

	// Rule: container failure not possible if disruption is node-level
	if s.ContainerFailure != nil && s.Level == chaostypes.DisruptionLevelNode {
		retErr = multierror.Append(retErr, errors.New("cannot execute a container failure because the level configuration is set to node"))
	}

	// Rule: At least one disruption kind must be applied
	if s.CPUPressure == nil && s.DiskPressure == nil && s.DiskFailure == nil && s.Network == nil && s.GRPC == nil && s.ContainerFailure == nil && s.NodeFailure == nil && len(s.DNS) == 0 {
		retErr = multierror.Append(retErr, errors.New("at least one disruption kind must be specified, please read the docs to see your options"))
	}

	// Rule: ContainerFailure and NodeFailure disruptions are not compatible with other failure types
	if s.ContainerFailure != nil {
		if s.CPUPressure != nil || s.DiskPressure != nil || s.DiskFailure != nil || s.Network != nil || s.GRPC != nil || s.NodeFailure != nil || len(s.DNS) > 0 {
			retErr = multierror.Append(retErr, errors.New("container failure disruptions are not compatible with other disruption kinds. The container failure will remove the impact of the other disruption types"))
		}
	}

	if s.NodeFailure != nil {
		if s.CPUPressure != nil || s.DiskPressure != nil || s.DiskFailure != nil || s.Network != nil || s.GRPC != nil || s.ContainerFailure != nil || len(s.DNS) > 0 {
			retErr = multierror.Append(retErr, errors.New("node failure disruptions are not compatible with other disruption kinds. The node failure will remove the impact of the other disruption types"))
		}
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
	if s.Triggers != nil && !s.Triggers.IsZero() {
		if !s.Triggers.Inject.IsZero() && !s.Triggers.CreatePods.IsZero() {
			if !s.Triggers.Inject.NotBefore.IsZero() && !s.Triggers.CreatePods.NotBefore.IsZero() && s.Triggers.Inject.NotBefore.Before(&s.Triggers.CreatePods.NotBefore) {
				retErr = multierror.Append(retErr, fmt.Errorf("spec.triggers.inject.notBefore is %s, which is before your spec.triggers.createPods.notBefore of %s. inject.notBefore must come after createPods.notBefore if both are specified", s.Triggers.Inject.NotBefore, s.Triggers.CreatePods.NotBefore))
			}
		}

		if !s.Triggers.Inject.IsZero() {
			if !s.Triggers.Inject.NotBefore.IsZero() && s.Triggers.Inject.Offset.Duration() != 0 {
				retErr = multierror.Append(retErr, errors.New("its not possible to set spec.triggers.inject.notBefore and spec.triggers.inject.offset"))
			}
		}

		if !s.Triggers.CreatePods.IsZero() {
			if !s.Triggers.CreatePods.NotBefore.IsZero() && s.Triggers.CreatePods.Offset.Duration() != 0 {
				retErr = multierror.Append(retErr, errors.New("its not possible to set spec.triggers.createPods.notBefore and spec.triggers.createPods.offset"))
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

		if (s.Pulse.ActiveDuration.Duration() > 0 && s.Pulse.DormantDuration.Duration() == 0) || (s.Pulse.ActiveDuration.Duration() == 0 && s.Pulse.DormantDuration.Duration() > 0) {
			retErr = multierror.Append(retErr, errors.New("if spec.pulse.activeDuration or spec.pulse.dormantDuration are specified, then both options must be set"))
		}

		if s.Pulse.ActiveDuration.Duration() != 0 && s.Pulse.ActiveDuration.Duration() < chaostypes.PulsingDisruptionMinimumDuration {
			retErr = multierror.Append(retErr, fmt.Errorf("pulse activeDuration of %s should be greater than %s", s.Pulse.ActiveDuration.Duration(), chaostypes.PulsingDisruptionMinimumDuration))
		}

		if s.Pulse.DormantDuration.Duration() != 0 && s.Pulse.DormantDuration.Duration() < chaostypes.PulsingDisruptionMinimumDuration {
			retErr = multierror.Append(retErr, fmt.Errorf("pulse dormantDuration of %s should be greater than %s", s.Pulse.DormantDuration.Duration(), chaostypes.PulsingDisruptionMinimumDuration))
		}
	}

	if s.GRPC != nil && s.Level == chaostypes.DisruptionLevelNode {
		retErr = multierror.Append(retErr, errors.New("GRPC disruptions can only be applied at the pod level"))
	}

	// Rule: count must be valid
	if err := ValidateCount(s.Count); err != nil {
		retErr = multierror.Append(retErr, err)
	}

	if s.Unsafemode != nil {
		if err := s.Unsafemode.Validate(); err != nil {
			retErr = multierror.Append(retErr, err)
		}
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

// Explain returns a string explanation of this disruption spec
func (s DisruptionSpec) Explain() []string {
	var explanation []string
	explanation = append(explanation, "Here's our best explanation of what this spec will do when run:")

	explanation = append(explanation, fmt.Sprintf("spec.duration is %s. After that amount of time, the disruption "+
		"will stop and clean itself up. If it fails to clean up, an alert will be sent. If you want the disruption to stop early, "+
		"just try to delete the disruption. All chaos-injector pods will immediately try to stop the failure.",
		s.Duration.Duration().String()))

	if s.DryRun {
		explanation = append(explanation, "spec.dryRun is set to true, meaning we will simulate a real disruption "+
			"as best as possible, by creating the resource, picking targets, and creating chaos-injector pods, "+
			"but we will not inject any actual failure.")
	}

	// s.Level can be "", which defaults to Pod
	if s.Level != chaostypes.DisruptionLevelNode {
		explanation = append(explanation, "spec.level is pod. We will pick pods as targets based on your selector, and inject the failure into the pods' containers.")
	} else {
		explanation = append(explanation, "spec.level is node. We will pick nodes as targets based on your selector, and inject the failure into the nodes, affecting all pods on those nodes.")
	}

	if s.Selector != nil {
		explanation = append(explanation, fmt.Sprintf("This spec has the following selectors which will be used to target %ss with these labels:\n\t%s", s.Level, s.Selector.String()))
	}

	if s.AdvancedSelector != nil {
		advancedSelectorExplanation := fmt.Sprintf("This spec has the following advanced selectors which will be used to target %ss based on their labels:\n", s.Level)

		for _, selector := range s.AdvancedSelector {
			advancedSelectorExplanation += fmt.Sprintf("\t\t%s\n", selector.String())
		}

		explanation = append(explanation, advancedSelectorExplanation)
	}

	if s.Filter != nil && s.Filter.Annotations != nil {
		explanation = append(explanation, fmt.Sprintf("This spec has the following annotation filters which will be used to target %ss with these annotations.\n\t\t  %s\n", s.Level, s.Filter.Annotations.String()))
	}

	if s.Containers != nil {
		explanation = append(explanation, fmt.Sprintf("spec.containers is set, so this disruption will only inject the failure the following containers on the target pods\n\t\t  %s\n", strings.Join(s.Containers, ",")))
	}

	if s.Pulse != nil {
		explanation = append(explanation,
			fmt.Sprintf("spec.pulse is set, so rather than a constant failure injection,after an initial delay of %s"+
				" the disruption will alternate between an active injected state with a duration of %s,"+
				" and an inactive dormant state with a duration of %s.\n",
				s.Pulse.InitialDelay.Duration().String(),
				s.Pulse.ActiveDuration.Duration().String(),
				s.Pulse.DormantDuration.Duration().String()))
	}

	if s.OnInit {
		explanation = append(explanation, fmt.Sprintf("spec.onInit is true. "+
			"The disruptions will be launched during the initialization of the targeted pods."+
			"This requires some extra setup on your end, please [read the full documentation](https://github.com/DataDog/chaos-controller/blob/main/docs/features.md#applying-a-disruption-on-pod-initialization)"))
	}

	countSuffix := ""
	if s.Count.Type == intstr.Int {
		countSuffix = fmt.Sprintf("exactly %d %ss. If it can't find that many targets, it will inject into as many as it discovers. "+
			"If there are more than %d eligible targets, a random %d will be chosen.",
			s.Count.IntValue(),
			s.Level,
			s.Count.IntValue(),
			s.Count.IntValue(),
		)
		countSuffix += " If it's more convenient, you can set spec.count to a % instead (just append the '%' character)."
	} else {
		countSuffix = fmt.Sprintf("%s percent of all eligible %ss found. "+
			"If it's more convenient, you can set spec.count to an int intead of a percentage.",
			s.Count.String(),
			s.Level,
		)
	}

	explanation = append(explanation, fmt.Sprintf("spec.count is %s, so the disruption will try to target %s",
		s.Count.String(),
		countSuffix,
	))

	if s.StaticTargeting {
		explanation = append(explanation, fmt.Sprintf("spec.staticTargeting is true, so after we pick an initial set of targets and inject, "+
			"we will not attempt to inject into any new targets that appear while the disruption is ongoing."))
	} else {
		explanation = append(explanation, "By default we will continually compare the injected target count "+
			"to your defined spec.count, and add/remove targets as needed, e.g., with a count of \"100%\", if new targets "+
			"are scheduled, we will inject into them as well. "+
			"If you want a different behavior, trying setting spec.staticTargeting to true.")
	}

	if s.Reporting != nil {
		explanation = append(explanation, s.Reporting.Explain())
	}

	if s.NodeFailure != nil {
		explanation = append(explanation, s.NodeFailure.Explain()...)
	}

	if s.ContainerFailure != nil {
		explanation = append(explanation, s.ContainerFailure.Explain()...)
	}

	if s.Network != nil {
		explanation = append(explanation, s.Network.Explain()...)
	}

	if s.CPUPressure != nil {
		explanation = append(explanation, s.CPUPressure.Explain()...)
	}

	if s.DiskPressure != nil {
		explanation = append(explanation, s.DiskPressure.Explain()...)
	}

	if s.DiskFailure != nil {
		explanation = append(explanation, s.DiskFailure.Explain()...)
	}

	if s.DNS != nil {
		explanation = append(explanation, s.DNS.Explain()...)
	}

	if s.GRPC != nil {
		explanation = append(explanation, s.GRPC.Explain()...)
	}

	return explanation
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

		if len(allContainers) < 1 {
			return nil, fmt.Errorf("couldn't find any running containers for pod '%s'", pod.Name)
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
			return nil, fmt.Errorf("could not find specified container in pod (pod: %s, targetContainer: %s)", pod.ObjectMeta.Name, containerName)
		}
	}

	return targetedContainers, nil
}
