package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
	// DelayedStartTolerance specifies the allowed deadline to start the disruption
	// after detecting a change in the target resource. If the disruption
	// does not start within this duration, the execution is considered failed.
	// +nullable
	DelayedStartTolerance DisruptionDuration `json:"delayedStartTolerance,omitempty"`

	// +kubebuilder:validation:Required
	// +ddmark:validation:Required=true
	// TargetResource specifies the resource to run disruptions against.
	// It can only be a deployment or statefulset.
	TargetResource TargetResourceSpec `json:"targetResource"`

	// +kubebuilder:validation:Required
	// +ddmark:validation:Required=true
	// Specifies the Disruption that will be created when executing a disruptionrollout.
	DisruptionTemplate DisruptionSpec `json:"disruptionTemplate"`
}

// DisruptionRolloutStatus defines the observed state of DisruptionRollout
type DisruptionRolloutStatus struct {
	// TargetResourcePodSpecHash represents the MD5 hash of the pod spec
	// of the target resource.
	TargetResourcePodSpecHash string `json:"targetResourcePodSpecHash,omitempty"`

	// PodSpecChangeTimestamp captures the time when a change in the pod spec
	// was detected.
	PodSpecChangeTimestamp metav1.Time `json:"podSpecChangeTimestamp,omitempty"`
}
