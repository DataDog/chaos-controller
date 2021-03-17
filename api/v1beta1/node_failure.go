// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package v1beta1

import (
	chaosapi "github.com/DataDog/chaos-controller/api"
	chaostypes "github.com/DataDog/chaos-controller/types"
)

// NodeFailureSpec represents a node failure injection
type NodeFailureSpec struct {
	Shutdown bool `json:"shutdown,omitempty"`
}

// Validate validates args for the given disruption
func (s *NodeFailureSpec) Validate() error {
	return nil
}

// GenerateArgs generates injection or cleanup pod arguments for the given spec
func (s *NodeFailureSpec) GenerateArgs(level chaostypes.DisruptionLevel, containerIDs []string, sink string, dryRun bool) []string {
	args := []string{
		"node-failure",
		"inject",
	}

	if s.Shutdown {
		args = append(args, "--shutdown")
	}

	// append common args
	args = chaosapi.AppendCommonArgs(args, "node", containerIDs, sink, dryRun)

	return args
}
