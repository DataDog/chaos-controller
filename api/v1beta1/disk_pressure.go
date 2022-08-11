// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package v1beta1

import (
	"strconv"
)

// DiskPressureSpec represents a disk pressure disruption
type DiskPressureSpec struct {
	Path string `json:"path"`
	// +ddmark:validation:Required=true
	Throttling DiskPressureThrottlingSpec `json:"throttling"`
}

// DiskPressureThrottlingSpec represents a throttle on read and write disk operations
type DiskPressureThrottlingSpec struct {
	ReadBytesPerSec  *int `json:"readBytesPerSec,omitempty"`
	WriteBytesPerSec *int `json:"writeBytesPerSec,omitempty"`
}

// Validate validates args for the given disruption
func (s *DiskPressureSpec) Validate() error {
	return nil
}

// GenerateArgs generates injection or cleanup pod arguments for the given spec
func (s *DiskPressureSpec) GenerateArgs() []string {
	args := []string{
		"disk-pressure",
		"--path",
		s.Path,
	}

	// add read throttling flag if specified
	if s.Throttling.ReadBytesPerSec != nil {
		args = append(args, []string{"--read-bytes-per-sec", strconv.Itoa(*s.Throttling.ReadBytesPerSec)}...)
	}

	// add write throttling flag if specified
	if s.Throttling.WriteBytesPerSec != nil {
		args = append(args, []string{"--write-bytes-per-sec", strconv.Itoa(*s.Throttling.WriteBytesPerSec)}...)
	}

	return args
}
