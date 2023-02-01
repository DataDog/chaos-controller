// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package v1beta1

// DiskFailureSpec represents a disk failure disruption
type DiskFailureSpec struct {
	Path string `json:"path"`
}

// Validate validates args for the given disruption
func (s *DiskFailureSpec) Validate() error {
	return nil
}

// GenerateArgs generates injection or cleanup pod arguments for the given spec
func (s *DiskFailureSpec) GenerateArgs() (args []string) {
	args = append(args, "disk-failure")
	if s.Path != "" {
		args = append(args, "--path", s.Path)
	}

	return args
}
