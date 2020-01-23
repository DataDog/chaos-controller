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

// NetworkLatencyInjectionSpec defines the desired state of NetworkLatencyInjection
type NetworkLatencyInjectionSpec struct {
	Selector labels.Set `json:"selector"`
	Count    *int       `json:"count"`
	Delay    uint       `json:"delay"`
	Hosts    []string   `json:"hosts"`
}

// NetworkLatencyInjectionStatus defines the observed state of NetworkLatencyInjection
type NetworkLatencyInjectionStatus struct {
	Finalizing bool     `json:"finalizing"`
	Injected   bool     `json:"injected"`
	Pods       []string `json:"pods,omitempty"`
}

// +kubebuilder:object:root=true

// NetworkLatencyInjection is the Schema for the networklatencyinjections API
// +kubebuilder:resource:shortName=nli
type NetworkLatencyInjection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkLatencyInjectionSpec   `json:"spec,omitempty"`
	Status NetworkLatencyInjectionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NetworkLatencyInjectionList contains a list of NetworkLatencyInjection
type NetworkLatencyInjectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetworkLatencyInjection `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NetworkLatencyInjection{}, &NetworkLatencyInjectionList{})
}
