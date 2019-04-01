/*
Copyright 2019 Datadog.

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

// NetworkFailureInjectionSpec defines the desired state of NetworkFailureInjection
type NetworkFailureInjectionSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Failure  NetworkFailureInjectionSpecFailure `json:"failure"`
	Selector labels.Set                         `json:"selector"`
}

// NetworkFailureInjectionSpecFailure defines the failure spec
type NetworkFailureInjectionSpecFailure struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Probability int    `json:"probability"`
	Protocol    string `json:"protocol"`
}

// NetworkFailureInjectionStatus defines the observed state of NetworkFailureInjection
type NetworkFailureInjectionStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Finalizing bool `json:"finalizing"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NetworkFailureInjection is the Schema for the networkfailureinjections API
// +k8s:openapi-gen=true
type NetworkFailureInjection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkFailureInjectionSpec   `json:"spec,omitempty"`
	Status NetworkFailureInjectionStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NetworkFailureInjectionList contains a list of NetworkFailureInjection
type NetworkFailureInjectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetworkFailureInjection `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NetworkFailureInjection{}, &NetworkFailureInjectionList{})
}
