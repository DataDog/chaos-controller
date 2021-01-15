// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package v1beta1

import (
	chaostypes "github.com/DataDog/chaos-controller/types"
)

// NodeFailureSpec represents a node failure injection
type NodeFailureSpec struct {
	Shutdown bool `json:"shutdown,omitempty"`
}

// GenerateArgs generates injection or cleanup pod arguments for the given spec
func (s *NodeFailureSpec) GenerateArgs(level chaostypes.DisruptionLevel, containerID, sink string, dryRun bool) []string {
	args := []string{
		"node-failure",
		"inject",
		"--metrics-sink",
		sink,
		"--level",
		"node",
	}

	// enable dry-run mode
	if dryRun {
		args = append(args, "--dry-run")
	}

	if s.Shutdown {
		args = append(args, "--shutdown")
	}

	return args
}
