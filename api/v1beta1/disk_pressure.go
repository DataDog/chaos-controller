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
	// +kubebuilder:validation:Minimum=0
	ReadBytesPerSec *int `json:"readBytesPerSec,omitempty"`
	// +kubebuilder:validation:Minimum=0
	WriteBytesPerSec *int `json:"writeBytesPerSec,omitempty"`
	// +kubebuilder:validation:Minimum=0
	ReadIOPSPerSec *int `json:"readIOPSPerSec,omitempty"`
	// +kubebuilder:validation:Minimum=0
	WriteIOPSPerSec *int `json:"writeIOPSPerSec,omitempty"`
}

// Validate validates args for the given disruption
func (s *DiskPressureSpec) Validate() error {
	// cgroup requires BPS values to be either 0 (remove limit) or >= 2; value 1 is rejected by the kernel
	bpsThrottles := map[string]*int{
		"readBytesPerSec":  s.Throttling.ReadBytesPerSec,
		"writeBytesPerSec": s.Throttling.WriteBytesPerSec,
	}

	for name, value := range bpsThrottles {
		if value != nil && *value != 0 && *value < 2 {
			return fmt.Errorf("disk pressure throttling %s must be 0 (remove limit) or >= 2, got %d", name, *value)
		}
	}

	// cgroup requires IOPS values to be either 0 (remove limit) or >= 2; value 1 is rejected by the kernel
	iopsThrottles := map[string]*int{
		"readIOPSPerSec":  s.Throttling.ReadIOPSPerSec,
		"writeIOPSPerSec": s.Throttling.WriteIOPSPerSec,
	}

	for name, value := range iopsThrottles {
		if value != nil && *value != 0 && *value < 2 {
			return fmt.Errorf("disk pressure throttling %s must be 0 (remove limit) or >= 2, got %d", name, *value)
		}
	}

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

	// add read iops throttling flag if specified
	if s.Throttling.ReadIOPSPerSec != nil {
		args = append(args, []string{"--read-iops-per-sec", strconv.Itoa(*s.Throttling.ReadIOPSPerSec)}...)
	}

	// add write iops throttling flag if specified
	if s.Throttling.WriteIOPSPerSec != nil {
		args = append(args, []string{"--write-iops-per-sec", strconv.Itoa(*s.Throttling.WriteIOPSPerSec)}...)
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

	if s.Throttling.ReadIOPSPerSec != nil {
		explanation += fmt.Sprintf("%d read io per second ", *s.Throttling.ReadIOPSPerSec)
	}

	if s.Throttling.WriteIOPSPerSec != nil {
		explanation += fmt.Sprintf("%d write io per second.", *s.Throttling.WriteIOPSPerSec)
	}

	return []string{"", explanation}
}
