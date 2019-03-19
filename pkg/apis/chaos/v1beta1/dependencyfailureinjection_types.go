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
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DependencyFailureInjectionSpec defines the desired state of DependencyFailureInjection
type DependencyFailureInjectionSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Failure       DependencyFailureInjectionSpecFailure `json:"failure"`
	LabelSelector string                                `json:"labelSelector"`
}

// DependencyFailureInjectionSpecFailure defines the failure spec
type DependencyFailureInjectionSpecFailure struct {
	Host        string `json:"host"`
	Port        string `json:"port"`
	Probability int    `json:"probability"`
}

// DependencyFailureInjectionStatus defines the observed state of DependencyFailureInjection
type DependencyFailureInjectionStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DependencyFailureInjection is the Schema for the dependencyfailureinjections API
// +k8s:openapi-gen=true
type DependencyFailureInjection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DependencyFailureInjectionSpec   `json:"spec,omitempty"`
	Status DependencyFailureInjectionStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DependencyFailureInjectionList contains a list of DependencyFailureInjection
type DependencyFailureInjectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DependencyFailureInjection `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DependencyFailureInjection{}, &DependencyFailureInjectionList{})
}
