// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package v1beta1

import (
	"fmt"
	"strings"
)

// DiskFailureSpec represents a disk failure disruption
type DiskFailureSpec struct {
	Path string `json:"path"`
}

const MaxDiskPathCharacters = 62

// Validate validates args for the given disruption
func (s *DiskFailureSpec) Validate() error {
	path := strings.TrimSpace(s.Path)

	if path == "" {
		return fmt.Errorf("the path of the disk failure disruption must not be empty")
	}

	if len(path) > MaxDiskPathCharacters {
		return fmt.Errorf("the path of the disk failure disruption must not exceed %d characters", MaxDiskPathCharacters)
	}

	return nil
}

// GenerateArgs generates injection or cleanup pod arguments for the given spec
func (s *DiskFailureSpec) GenerateArgs() (args []string) {
	args = append(args, "disk-failure")
	path := strings.TrimSpace(s.Path)
	args = append(args, "--path", path)

	return args
}
