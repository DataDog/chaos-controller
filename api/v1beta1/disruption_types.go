// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package v1beta1

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	chaosapi "github.com/DataDog/chaos-controller/api"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"github.com/hashicorp/go-multierror"
	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// DisruptionSpec defines the desired state of Disruption
// +ddmark:validation:ExclusiveFields={Network,DNS}
// +ddmark:validation:ExclusiveFields={ContainerFailure,CPUPressure,DiskPressure,NodeFailure,Network,DNS}
// +ddmark:validation:ExclusiveFields={NodeFailure,CPUPressure,DiskPressure,ContainerFailure,Network,DNS}
// +ddmark:validation:AtLeastOneOf={DNS,CPUPressure,Network,NodeFailure,ContainerFailure,DiskPressure,GRPC}
type DisruptionSpec struct {
	// +kubebuilder:validation:Required
	// +ddmark:validation:Required=true
	Count *intstr.IntOrString `json:"count"` // number of pods to target in either integer form or percent form appended with a %
	// +nullable
	// +kubebuilder:validation:Required
	// +ddmark:validation:Required=true
	Selector labels.Set `json:"selector,omitempty"` // label selector
	// +nullable
	AdvancedSelector []metav1.LabelSelectorRequirement `json:"advancedSelector,omitempty"` // advanced label selector
	DryRun           bool                              `json:"dryRun,omitempty"`           // enable dry-run mode
	OnInit           bool                              `json:"onInit,omitempty"`           // enable disruption on init
	Duration         DisruptionDuration                `json:"duration,omitempty"`         // time from disruption creation until chaos pods are deleted and no more are created
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
	// +kubebuilder:validation:Enum=NotInjected;PartiallyInjected;Injected
	// +ddmark:validation:Enum=NotInjected;PartiallyInjected;Injected
	InjectionStatus chaostypes.DisruptionInjectionStatus `json:"injectionStatus,omitempty"`
	// +nullable
	Targets []string `json:"targets,omitempty"`
	// +nullable
	IgnoredTargets []string `json:"ignoredTargets,omitempty"`
	// +nullable
	UserInfo *authv1.UserInfo `json:"userInfo,omitempty"`
}

//+kubebuilder:object:root=true

// Disruption is the Schema for the disruptions API
// +kubebuilder:resource:shortName=dis
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

		if s.Level != chaostypes.DisruptionLevelPod && s.Level != chaostypes.DisruptionLevelUnspecified {
			retErr = multierror.Append(retErr, errors.New("OnInit is only compatible with pod level disruptions"))
		}

		if len(s.Containers) > 0 {
			retErr = multierror.Append(retErr, errors.New("OnInit is not compatible with containers scoping"))
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
