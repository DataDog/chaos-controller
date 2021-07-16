// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package v1beta1

// PodFailureSpec represents a pod failure injection
type PodFailureSpec struct {
	Kill bool `json:"kill,omitempty"`
}

// Validate validates args for the given disruption
func (s *PodFailureSpec) Validate() error {
	return nil
}

// GenerateArgs generates injection or cleanup pod arguments for the given spec
func (s *PodFailureSpec) GenerateArgs() []string {
	args := []string{
		"pod-failure",
		"inject",
	}

	if s.Kill {
		args = append(args, "--kill")
	}

	return args
}
