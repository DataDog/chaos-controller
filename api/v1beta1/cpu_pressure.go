// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package v1beta1

import (
	chaostypes "github.com/DataDog/chaos-controller/types"
	"k8s.io/apimachinery/pkg/types"
)

// CPUPressureSpec represents a node failure injection
type CPUPressureSpec struct {
}

// GenerateArgs generates injection or cleanup pod arguments for the given spec
func (s *CPUPressureSpec) GenerateArgs(mode chaostypes.PodMode, uid types.UID, containerID, sink string) []string {
	args := []string{}

	if mode == chaostypes.PodModeInject {
		args = []string{
			"cpu-pressure",
			"inject",
			"--uid",
			string(uid),
			"--metrics-sink",
			sink,
			"--container-id",
			containerID,
		}
	}

	return args
}
