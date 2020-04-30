// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

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
	"strconv"
	"strings"

	chaostypes "github.com/DataDog/chaos-controller/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DisruptionSpec defines the desired state of Disruption
type DisruptionSpec struct {
	// +kubebuilder:validation:Required
	Count int `json:"count"` // number of pods to target
	// +kubebuilder:validation:Required
	Selector labels.Set `json:"selector"` // label selector
	// +nullable
	NetworkFailure *NetworkFailureSpec `json:"networkFailure,omitempty"`
	// +nullable
	NetworkLatency *NetworkLatencySpec `json:"networkLatency,omitempty"`
	// +nullable
	NodeFailure *NodeFailureSpec `json:"nodeFailure,omitempty"`
}

// NetworkFailureSpec represents a network failure injection
type NetworkFailureSpec struct {
	// +nullable
	Hosts              []string `json:"hosts,omitempty"`
	Port               int      `json:"port"`
	Probability        int      `json:"probability"`
	Protocol           string   `json:"protocol"`
	AllowEstablishment bool     `json:"allowEstablishment,omitempty"`
}

// GenerateArgs generates injection or cleanup pod arguments for the given spec
func (s *NetworkFailureSpec) GenerateArgs(mode chaostypes.PodMode, uid types.UID, containerID, sink string) []string {
	args := []string{}

	switch mode {
	case chaostypes.PodModeInject:
		args = []string{
			"network-failure",
			"inject",
			"--uid",
			string(uid),
			"--metrics-sink",
			sink,
			"--container-id",
			containerID,
			"--port",
			strconv.Itoa(s.Port),
			"--protocol",
			s.Protocol,
			"--probability",
			strconv.Itoa(s.Probability),
			"--hosts",
		}
		args = append(args, strings.Split(strings.Join(s.Hosts, " --hosts "), " ")...)

		// allow establishment
		if s.AllowEstablishment {
			args = append(args, "--allow-establishment")
		}
	case chaostypes.PodModeClean:
		args = []string{
			"network-failure",
			"clean",
			"--uid",
			string(uid),
			"--metrics-sink",
			sink,
			"--container-id",
			containerID,
		}
	}

	return args
}

// NetworkLatencySpec represents a network latency injection
type NetworkLatencySpec struct {
	// +kubebuilder:validation:Maximum=59999
	Delay uint `json:"delay"`
	// +nullable
	Hosts []string `json:"hosts,omitempty"`
}

// GenerateArgs generates injection or cleanup pod arguments for the given spec
func (s *NetworkLatencySpec) GenerateArgs(mode chaostypes.PodMode, uid types.UID, containerID, sink string) []string {
	args := []string{}

	switch mode {
	case chaostypes.PodModeInject:
		args = []string{
			"network-latency",
			"inject",
			"--uid",
			string(uid),
			"--metrics-sink",
			sink,
			"--container-id",
			containerID,
			"--delay",
			strconv.Itoa(int(s.Delay)),
			"--hosts",
		}
		args = append(args, strings.Split(strings.Join(s.Hosts, " --hosts "), " ")...)
	case chaostypes.PodModeClean:
		args = []string{
			"network-latency",
			"clean",
			"--uid",
			string(uid),
			"--metrics-sink",
			sink,
			"--container-id",
			containerID,
			"--hosts",
		}
		args = append(args, strings.Split(strings.Join(s.Hosts, " --hosts "), " ")...)
	}

	return args
}

// NodeFailureSpec represents a node failure injection
type NodeFailureSpec struct {
	Shutdown bool `json:"shutdown,omitempty"`
}

// GenerateArgs generates injection or cleanup pod arguments for the given spec
func (s *NodeFailureSpec) GenerateArgs(mode chaostypes.PodMode, uid types.UID, containerID, sink string) []string {
	args := []string{}

	if mode == chaostypes.PodModeInject {
		args = []string{
			"node-failure",
			"inject",
			"--uid",
			string(uid),
			"--metrics-sink",
			sink,
		}
		if s.Shutdown {
			args = append(args, "--shutdown")
		}
	}

	return args
}

// DisruptionStatus defines the observed state of Disruption
type DisruptionStatus struct {
	IsFinalizing bool `json:"isFinalizing,omitempty"`
	IsInjected   bool `json:"isInjected,omitempty"`
	// +nullable
	TargetPods []string `json:"targetPods,omitempty"`
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
