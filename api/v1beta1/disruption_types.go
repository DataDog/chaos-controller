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
func (ds *DisruptionSpec) Hash() (string, error) {
	// serialize instance spec to JSON
	specBytes, err := json.Marshal(ds)
	if err != nil {
		return "", fmt.Errorf("error serializing instance spec: %w", err)
	}

	// compute bytes hash
	return fmt.Sprintf("%x", md5.Sum(specBytes)), nil
}

func (s *DisruptionSpec) Validate() error {
	// Rule: no targeted container if disruption is node-level
	if len(s.Containers) > 0 && s.Level != chaostypes.DisruptionLevelPod {
		return errors.New("cannot target specific container of a node-level disruption")
	}

	// Rule: at least one disruption field
	if s.DNS == nil &&
		s.CPUPressure == nil &&
		s.Network == nil &&
		s.NodeFailure == nil &&
		s.DiskPressure == nil {
		return errors.New("cannot apply a disruption with no target process")
	}

	//Rule: DNS and Network disruptions are incompatible
	if s.DNS != nil && s.Network != nil {
		return errors.New("cannot apply DNS and Network disruptions concurrently")
	}

	return nil
}

// validates rules for disruption global scope and all subsequent disruption specifications
func (s *DisruptionSpec) ValidateDisruptionSpec() error {
	err := s.Validate()
	if err != nil {
		return err
	}

	for _, kind := range chaostypes.DisruptionKinds {
		var validator chaosapi.DisruptionValidator

		disruptionExists, validator, _ := s.DisruptionKindInterfaceGenerator(kind)
		if !disruptionExists {
			continue
		}

		err := validator.Validate()
		if err != nil {
			return err
		}
	}
	return nil
}

// validates existence and generates instances of DisruptionKind Interfaces: DisruptionValidator, DisruptionArgsGenerator
func (s *DisruptionSpec) DisruptionKindInterfaceGenerator(kind chaostypes.DisruptionKind) (bool, chaosapi.DisruptionValidator, chaosapi.DisruptionArgsGenerator) {
	var validator chaosapi.DisruptionValidator
	var generator chaosapi.DisruptionArgsGenerator

	switch kind {
	case chaostypes.DisruptionKindNodeFailure:
		validator, generator = s.NodeFailure, s.NodeFailure
	case chaostypes.DisruptionKindNetworkDisruption:
		validator, generator = s.Network, s.Network
	case chaostypes.DisruptionKindDNSDisruption:
		validator, generator = s.DNS, s.DNS
	case chaostypes.DisruptionKindCPUPressure:
		validator, generator = s.CPUPressure, s.CPUPressure
	case chaostypes.DisruptionKindDiskPressure:
		validator, generator = s.DiskPressure, s.DiskPressure
	}

	// ensure that the underlying disruption spec is not nil
	disruptionExists := !reflect.ValueOf(validator).IsNil() && !reflect.ValueOf(generator).IsNil()
	return disruptionExists, validator, generator
}
