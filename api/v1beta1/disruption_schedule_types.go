// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package v1beta1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

func init() {
	SchemeBuilder.Register(&DisruptionSchedule{}, &DisruptionScheduleList{})
}

//+kubebuilder:object:root=true

// DisruptionSchedule is the Schema for the disruptions API
// +kubebuilder:resource:shortName=disch
// +kubebuilder:subresource:status
type DisruptionSchedule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              DisruptionScheduleSpec   `json:",inline"`
	Status            DisruptionScheduleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DisruptionScheduleList contains a list of DisruptionSchedule
type DisruptionScheduleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DisruptionSchedule `json:"items"`
}

type DisruptionScheduleSpec struct {
}
type DisruptionScheduleStatus struct {
}
