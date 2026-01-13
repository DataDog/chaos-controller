// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package v1beta1

import (
	"strconv"

	chaostypes "github.com/DataDog/chaos-controller/types"
)

// PodReplacementSpec represents a pod replacement disruption
type PodReplacementSpec struct {
	// DeleteStorage determines if PVCs associated with the target pod should be deleted
	// +kubebuilder:default=true
	DeleteStorage bool `json:"deleteStorage,omitempty"`
	// ForceDelete forces deletion of stuck pods by setting grace period to 0
	ForceDelete bool `json:"forceDelete,omitempty"`
	// GracePeriodSeconds specifies the grace period for pod deletion in seconds
	// If not specified, uses the pod's default grace period
	GracePeriodSeconds *int64 `json:"gracePeriodSeconds,omitempty"`
}

// Validate validates args for the given disruption
func (s *PodReplacementSpec) Validate() error {
	return nil
}

// GenerateArgs generates injection or cleanup pod arguments for the given spec
func (s *PodReplacementSpec) GenerateArgs() []string {
	args := []string{
		chaostypes.DisruptionKindPodReplacement,
		"inject",
	}

	if s.DeleteStorage {
		args = append(args, "--delete-storage")
	}

	if s.ForceDelete {
		args = append(args, "--force-delete")
	}

	if s.GracePeriodSeconds != nil {
		args = append(args, "--grace-period-seconds", strconv.FormatInt(*s.GracePeriodSeconds, 10))
	}

	return args
}

func (s *PodReplacementSpec) Explain() []string {
	explanation := "spec.podReplacement will cordon the node hosting the target pod to prevent new pods from being scheduled, " +
		"then delete the target pod to force it to reschedule. "

	if s.DeleteStorage {
		explanation += "PersistentVolumeClaims associated with the pod will also be deleted to simulate complete storage loss. "
	}

	explanation += "This simulates a complete pod replacement scenario where both the pod and its storage are recreated. " +
		"Unlike node-level disruptions, this only affects the specifically targeted pod."

	return []string{"", explanation}
}
