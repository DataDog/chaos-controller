package v1beta1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

func init() {
	SchemeBuilder.Register(&DisruptionRollout{}, &DisruptionRolloutList{})
}

//+kubebuilder:object:root=true

// DisruptionRollout is the Schema for the disruptionrollout API
// +kubebuilder:resource:shortName=diroll
// +kubebuilder:subresource:status
type DisruptionRollout struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              DisruptionRolloutSpec   `json:"spec,omitempty"`
	Status            DisruptionRolloutStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DisruptionRolloutList contains a list of DisruptionRollout
type DisruptionRolloutList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DisruptionRollout `json:"items"`
}

// DisruptionRolloutSpec defines the desired state of DisruptionRollout
type DisruptionRolloutSpec struct {
}

// DisruptionRolloutStatus defines the observed state of DisruptionRollout
type DisruptionRolloutStatus struct {
}
