// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package v1beta1

import (
	"fmt"
	"time"

	"github.com/DataDog/chaos-controller/log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	SchemeBuilder.Register(&DisruptionCron{}, &DisruptionCronList{})
}

//+kubebuilder:object:root=true

// DisruptionCron is the Schema for the disruptioncron API
// +kubebuilder:resource:shortName=dicron
// +kubebuilder:subresource:status
// +genclient
// +genclient:noStatus
// +genclient:onlyVerbs=create,get,list,delete,watch,update
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
	// The schedule in Cron format, see https://en.wikipedia.org/wiki/Cron.
	Schedule string `json:"schedule"`

	// If set to true, no disruptions will be created from this DisruptionCron
	// useful if there's a reason to temporarily stop injecting, but without
	// deleting this DisruptionCron
	Paused bool `json:"paused,omitempty"`

	// Optional deadline for starting the disruption if it misses scheduled time
	// for any reason.  Missed disruption executions will be counted as failed ones.
	// +nullable
	DelayedStartTolerance DisruptionDuration `json:"delayedStartTolerance,omitempty"`

	// +kubebuilder:validation:Required
	// TargetResource specifies the resource to run disruptions against.
	// It can only be a deployment or statefulset.
	TargetResource TargetResourceSpec `json:"targetResource"`

	// +kubebuilder:validation:Required
	// Specifies the Disruption that will be created when executing a disruptioncron.
	DisruptionTemplate DisruptionSpec `json:"disruptionTemplate"`

	// +nullable
	Reporting *Reporting `json:"reporting,omitempty"`
}

// TargetResourceSpec specifies the long-lived resource to be targeted for disruptions.
// DisruptionCrons are intended to exist semi-permanently, and thus appropriate targets can only be other long-lived resources,
// such as statefulsets or deployment.
type TargetResourceSpec struct {
	// +kubebuilder:validation:Enum=deployment;statefulset
	// +kubebuilder:validation:Required
	// Kind specifies the type of the long-lived resource. Allowed values: "deployment", "statefulset".
	Kind string `json:"kind"`

	// +kubebuilder:validation:Required
	// Name specifies the name of the specific instance of the long-lived resource to be targeted.
	Name string `json:"name"`
}

func (trs TargetResourceSpec) String() string {
	return fmt.Sprintf("%s/%s", trs.Kind, trs.Name)
}

// DisruptionCronStatus defines the observed state of DisruptionCron
type DisruptionCronStatus struct {
	// The last time when the disruption was last successfully scheduled.
	// +nullable
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty"`

	// Time when the target resource was previously missing.
	// +nullable
	TargetResourcePreviouslyMissing *metav1.Time `json:"targetResourcePreviouslyMissing,omitempty"`

	History []DisruptionCronTrigger `json:"history,omitempty"`
}

const MaxHistoryLen = 5

type DisruptionCronTrigger struct {
	Name      string      `json:"name,omitempty"`
	Kind      string      `json:"kind,omitempty"`
	CreatedAt metav1.Time `json:"createdAt,omitempty"`
}

// IsReadyToRemoveFinalizer checks if adisruptioncron has been deleting for > finalizerDelay
func (d *DisruptionCron) IsReadyToRemoveFinalizer(finalizerDelay time.Duration) bool {
	return d.DeletionTimestamp != nil && time.Now().After(d.DeletionTimestamp.Add(finalizerDelay))
}

// getMetricsTags parses the disruptioncron to generate metrics tags
func (d *DisruptionCron) getMetricsTags() []string {
	tags := []string{
		fmt.Sprintf("%s:%s", log.DisruptionCronNameKey, d.Name),
		fmt.Sprintf("%s:%s", log.DisruptionCronNamespaceKey, d.Namespace),
	}

	return tags
}
