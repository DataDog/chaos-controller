// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package v1beta1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

func init() {
	SchemeBuilder.Register(&DisruptionCron{}, &DisruptionCronList{})
}

//+kubebuilder:object:root=true

// DisruptionCron is the Schema for the disruptioncron API
// +kubebuilder:resource:shortName=dicron
// +kubebuilder:subresource:status
type DisruptionCron struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              DisruptionCronSpec   `json:"spec,omitempty"`
	Status            DisruptionCronStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DisruptionCronList contains a list of DisruptionCron
type DisruptionCronList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DisruptionCron `json:"items"`
}

// DisruptionCronSpec defines the desired state of DisruptionCron
type DisruptionCronSpec struct {
	// +kubebuilder:validation:MinLength=0
	// +kubebuilder:validation:Required
	// +ddmark:validation:Required=true
	// The schedule in Cron format, see https://en.wikipedia.org/wiki/Cron.
	Schedule string `json:"schedule"`

	//+kubebuilder:validation:Minimum=0
	// Optional deadline in seconds for starting the disruption if it misses scheduled
	// time for any reason.  Missed disruption executions will be counted as failed ones.
	// +nullable
	StartingDeadlineSeconds *int64 `json:"startingDeadlineSeconds,omitempty"`

	// +kubebuilder:validation:Required
	// +ddmark:validation:Required=true
	// TargetResource specifies the resource to run disruptions against.
	// It can only be a deployment or statefulset.
	TargetResource TargetResourceSpec `json:"targetResource"`

	// +kubebuilder:validation:Required
	// +ddmark:validation:Required=true
	// Specifies the Disruption that will be created when executing a disruptioncron.
	DisruptionTemplate DisruptionSpec `json:"disruptionTemplate"`
}

// TargetResource specifies the long-lived resource to be targeted for disruptions.
// DisruptionCrons are intended to exist semi-permanently, and thus appropriate targets can only be other long-lived resources,
// such as statefulsets or deployment.
type TargetResourceSpec struct {
	// +kubebuilder:validation:Enum=deployment;statefulset
	// +kubebuilder:validation:Required
	// +ddmark:validation:Enum=deployment;statefulset
	// +ddmark:validation:Required=true
	// Kind specifies the type of the long-lived resource. Allowed values: "deployment", "statefulset".
	Kind string `json:"kind"`

	// +kubebuilder:validation:Required
	// +ddmark:validation:Required=true
	// Name specifies the name of the specific instance of the long-lived resource to be targeted.
	Name string `json:"name"`
}

// DisruptionCronStatus defines the observed state of DisruptionCron
type DisruptionCronStatus struct {
	// The last time when the disruption was last successfully scheduled.
	// +nullable
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty"`

	// Time when the target resource was previously missing.
	// +nullable
	TargetResourcePreviouslyMissing *metav1.Time `json:"TargetResourcePreviouslyMissing,omitempty"`
}
