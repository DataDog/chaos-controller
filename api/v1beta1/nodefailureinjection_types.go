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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NodeFailureInjectionSpec defines the desired state of NodeFailureInjection
type NodeFailureInjectionSpec struct {
	Selector labels.Set `json:"selector"`
	// Number of pods to target, defaults to 1 if not specified
	Quantity *int `json:"quantity,omitempty"`
	Shutdown bool `json:"shutdown,omitempty"`
}

// NodeFailureInjectionStatus defines the observed state of NodeFailureInjection
type NodeFailureInjectionStatus struct {
	Injected []NodeFailureInjectionStatusInjectedEntry `json:"injected,omitempty"`
}

// NodeFailureInjectionStatusInjectedEntry defines an injected node and pod
type NodeFailureInjectionStatusInjectedEntry struct {
	Node string `json:"node"`
	Pod  string `json:"pod"`
}

// +kubebuilder:object:root=true

// NodeFailureInjection is the Schema for the nodefailureinjections API
// +kubebuilder:resource:shortName=nofi
type NodeFailureInjection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeFailureInjectionSpec   `json:"spec,omitempty"`
	Status NodeFailureInjectionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NodeFailureInjectionList contains a list of NodeFailureInjection
type NodeFailureInjectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeFailureInjection `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NodeFailureInjection{}, &NodeFailureInjectionList{})
}
