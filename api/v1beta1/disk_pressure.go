// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package v1beta1

import (
	"fmt"
	"strconv"
)

// DiskPressureSpec represents a disk pressure disruption
type DiskPressureSpec struct {
	// +kubebuilder:validation:Required
	Path string `json:"path"  chaos_validate:"required"`
	// +kubebuilder:validation:Required
	Throttling DiskPressureThrottlingSpec `json:"throttling" chaos_validate:"required"`
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

func (s *DiskPressureSpec) Explain() []string {
	explanation := fmt.Sprintf("spec.diskPressure will throttle io on the device mounted to the path %s, limiting it to ", s.Path)

	if s.Throttling.ReadBytesPerSec != nil {
		explanation += fmt.Sprintf("%d read bytes per second ", *s.Throttling.ReadBytesPerSec)
	}

	if s.Throttling.WriteBytesPerSec != nil {
		explanation += fmt.Sprintf("%d write bytes per second.", *s.Throttling.WriteBytesPerSec)
	}

	return []string{"", explanation}
}
