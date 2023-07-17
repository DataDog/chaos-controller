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
	Spec              DisruptionScheduleSpec   `json:"spec,omitempty"`
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
	//+kubebuilder:validation:MinLength=0
	// The schedule in Cron format, see https://en.wikipedia.org/wiki/Cron.
	Schedule string `json:"schedule"`

	// TargetResource specifies the resource to run disruptions against.
	// It can only be a Deployment or StatefulSet.
	TargetResource TargetResource `json:"TargetResource"`

	// Specifies the Disruption that will be created when executing a DisruptionShedule.
	DisruptionTemplate Disruption `json:"disruptionTemplate"`
}

// TargetResource specifies the long-lived resource to be targeted for disruptions.
// Disruptions are intended to exist semi-permanently, and thus appropriate targets can only be other long-lived resources,
// such as StatefulSets or Deployments.
type TargetResource struct {
	// Kind specifies the type of the long-lived resource. Allowed values: "Deployment", "StatefulSet".
	// +kubebuilder:validation:Enum=Deployment;StatefulSet
	Kind string `json:"kind"`

	// Name specifies the name of the specific instance of the long-lived resource to be targeted.
	Name string `json:"name"`
}

type DisruptionScheduleStatus struct {
	// The last time when the disruption was last successfully scheduled.
	// +optional
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty"`

	// Time when the target resource was previously missing.
	// +optional
	TargetResourcePreviouslyMissing *metav1.Time `json:"TargetResourcePreviouslyMissing,omitempty"`
}
