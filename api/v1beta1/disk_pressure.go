// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package v1beta1

import (
	"errors"
	"strconv"
	"strings"

	chaostypes "github.com/DataDog/chaos-controller/types"
)

// DiskPressureSpec represents a disk pressure disruption
type DiskPressureSpec struct {
	Path       string                     `json:"path"`
	Throttling DiskPressureThrottlingSpec `json:"throttling"`
}

// DiskPressureThrottlingSpec represents a throttle on read and write disk operations
type DiskPressureThrottlingSpec struct {
	ReadBytesPerSec  *int `json:"readBytesPerSec,omitempty"`
	WriteBytesPerSec *int `json:"writeBytesPerSec,omitempty"`
}

// Validate validates args for the given disruption
func (s *DiskPressureSpec) Validate() error {
	if s.Throttling.ReadBytesPerSec == nil && s.Throttling.WriteBytesPerSec == nil {
		return errors.New("the disk pressure disruption was selected, but no throttling values were set. Please set at least one of: readBytesPerSec, or writeBytesPerSec. No injection will occur")
	}

	return nil
}

// GenerateArgs generates injection or cleanup pod arguments for the given spec
func (s *DiskPressureSpec) GenerateArgs(level chaostypes.DisruptionLevel, containerIDs []string, sink string, dryRun bool) []string {
	args := []string{
		"disk-pressure",
		"--metrics-sink",
		sink,
		"--level",
		string(level),
		"--containers-id",
		strings.Join(containerIDs, ","),
		"--path",
		s.Path,
	}

	// enable dry-run mode
	if dryRun {
		args = append(args, "--dry-run")
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
