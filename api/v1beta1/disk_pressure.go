// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package v1beta1

import (
	"strconv"

	chaostypes "github.com/DataDog/chaos-controller/types"
	"k8s.io/apimachinery/pkg/types"
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

// GenerateArgs generates injection or cleanup pod arguments for the given spec
func (s *DiskPressureSpec) GenerateArgs(mode chaostypes.PodMode, uid types.UID, level chaostypes.DisruptionLevel, containerID, sink string) []string {
	var args []string

	if mode == chaostypes.PodModeInject {
		args = []string{
			"disk-pressure",
			"inject",
			"--uid",
			string(uid),
			"--metrics-sink",
			sink,
			"--container-id",
			containerID,
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
	} else {
		args = []string{
			"disk-pressure",
			"clean",
			"--uid",
			string(uid),
			"--metrics-sink",
			sink,
			"--container-id",
			containerID,
			"--path",
			s.Path,
		}
	}

	return args
}
