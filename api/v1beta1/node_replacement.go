// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package v1beta1

import "strconv"

// NodeReplacementSpec represents a node replacement disruption
type NodeReplacementSpec struct {
	// DeleteStorage determines if PVCs associated with pods on the target node should be deleted
	// +kubebuilder:default=true
	DeleteStorage bool `json:"deleteStorage,omitempty"`
	// ForceDelete forces deletion of stuck pods by setting grace period to 0
	ForceDelete bool `json:"forceDelete,omitempty"`
	// GracePeriodSeconds specifies the grace period for pod deletion in seconds
	// If not specified, uses the pod's default grace period
	GracePeriodSeconds *int64 `json:"gracePeriodSeconds,omitempty"`
}

// Validate validates args for the given disruption
func (s *NodeReplacementSpec) Validate() error {
	return nil
}

// GenerateArgs generates injection or cleanup pod arguments for the given spec
func (s *NodeReplacementSpec) GenerateArgs() []string {
	args := []string{
		"node-replacement",
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

func (s *NodeReplacementSpec) Explain() []string {
	explanation := "spec.nodeReplacement will cordon the target node to prevent new pods from being scheduled, " +
		"then delete pods running on that node to force them to reschedule on other nodes. "

	if s.DeleteStorage {
		explanation += "PersistentVolumeClaims associated with the pods will also be deleted to simulate complete storage loss. "
	}

	explanation += "This simulates a complete node replacement scenario where both compute and storage are lost. " +
		"This will affect ALL pods on the target node."

	return []string{"", explanation}
}