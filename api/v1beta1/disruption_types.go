// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	chaosapi "github.com/DataDog/chaos-controller/api"
	chaostypes "github.com/DataDog/chaos-controller/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DisruptionSpec defines the desired state of Disruption
type DisruptionSpec struct {
	// +kubebuilder:validation:Required
	Count *intstr.IntOrString `json:"count"` // number of pods to target in either integer form or percent form appended with a %
	// +kubebuilder:validation:Required
	Selector labels.Set `json:"selector"`         // label selector
	DryRun   bool       `json:"dryRun,omitempty"` // enable dry-run mode
	OnInit   bool       `json:"onInit,omitempty"` // enable disruption on init
	// +kubebuilder:validation:Enum=pod;node;""
	Level      chaostypes.DisruptionLevel `json:"level,omitempty"`
	Containers []string                   `json:"containers,omitempty"`
	// +nullable
	Network *NetworkDisruptionSpec `json:"network,omitempty"`
	// +nullable
	NodeFailure *NodeFailureSpec `json:"nodeFailure,omitempty"`
	// +nullable
	CPUPressure *CPUPressureSpec `json:"cpuPressure,omitempty"`
	// +nullable
	DiskPressure *DiskPressureSpec `json:"diskPressure,omitempty"`
	// +nullable
	DNS DNSDisruptionSpec `json:"dns,omitempty"`
}

// DisruptionStatus defines the observed state of Disruption
type DisruptionStatus struct {
	IsStuckOnRemoval bool `json:"isStuckOnRemoval,omitempty"`
	IsInjected       bool `json:"isInjected,omitempty"`
	// +kubebuilder:validation:Enum=NotInjected;PartiallyInjected;Injected
	InjectionStatus chaostypes.DisruptionInjectionStatus `json:"injectionStatus,omitempty"`
	// +nullable
	Targets []string `json:"targets,omitempty"`
	// +nullable
	IgnoredTargets []string `json:"ignoredTargets,omitempty"`
}

// +kubebuilder:object:root=true

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
func (s *DisruptionSpec) Validate() error {
	err := s.validateGlobalDisruptionScope()
	if err != nil {
		return err
	}

	for _, kind := range chaostypes.DisruptionKindNames {
		disruptionKind := s.DisruptionKindPicker(kind)
		if reflect.ValueOf(disruptionKind).IsNil() {
			continue
		}

		err := disruptionKind.Validate()
		if err != nil {
			return err
		}
	}

	return nil
}

// Validate applies rules for disruption global scope
func (s *DisruptionSpec) validateGlobalDisruptionScope() error {
	// Rule: no targeted container if disruption is node-level
	if len(s.Containers) > 0 && s.Level == chaostypes.DisruptionLevelNode {
		return errors.New("cannot target specific containers because the level configuration is set to node")
	}

	// Rule: at least one disruption field
	if s.DNS == nil &&
		s.CPUPressure == nil &&
		s.Network == nil &&
		s.NodeFailure == nil &&
		s.DiskPressure == nil {
		return errors.New("cannot apply an empty disruption - at least one of Network, DNS, DiskPressure, NodeFailure, CPUPressure fields is needed")
	}

	// Rule: on init compatibility
	if s.OnInit {
		if s.CPUPressure != nil ||
			s.NodeFailure != nil ||
			s.DiskPressure != nil {
			return errors.New("OnInit is only compatible with network and dns disruptions")
		}

		if s.Level != chaostypes.DisruptionLevelPod && s.Level != chaostypes.DisruptionLevelUnspecified {
			return errors.New("OnInit is only compatible with pod level disruptions")
		}

		if len(s.Containers) > 0 {
			return errors.New("OnInit is not compatible with containers scoping")
		}
	}

	// Rule: count must be valid
	if err := ValidateCount(s.Count); err != nil {
		return err
	}

	return nil
}

// DisruptionKindPicker returns this DisruptionSpec's instance of a DisruptionKind based on given kind name
func (s *DisruptionSpec) DisruptionKindPicker(kind chaostypes.DisruptionKindName) chaosapi.DisruptionKind {
	var disruptionKind chaosapi.DisruptionKind

	switch kind {
	case chaostypes.DisruptionKindNodeFailure:
		disruptionKind = s.NodeFailure
	case chaostypes.DisruptionKindNetworkDisruption:
		disruptionKind = s.Network
	case chaostypes.DisruptionKindDNSDisruption:
		disruptionKind = s.DNS
	case chaostypes.DisruptionKindCPUPressure:
		disruptionKind = s.CPUPressure
	case chaostypes.DisruptionKindDiskPressure:
		disruptionKind = s.DiskPressure
	}

	return disruptionKind
}
