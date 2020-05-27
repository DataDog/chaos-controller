// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package v1beta1

import (
	"strconv"
	"strings"

	chaostypes "github.com/DataDog/chaos-controller/types"
	"k8s.io/apimachinery/pkg/types"
)

// NetworkLatencySpec represents a network latency injection
type NetworkLatencySpec struct {
	// +kubebuilder:validation:Maximum=59999
	Delay uint `json:"delay"`
	// +nullable
	Hosts []string `json:"hosts,omitempty"`
	// +nullable
	Port int `json:"port,omitempty"`
}

// GenerateArgs generates injection or cleanup pod arguments for the given spec
func (s *NetworkLatencySpec) GenerateArgs(mode chaostypes.PodMode, uid types.UID, containerID, sink string) []string {
	args := []string{}

	switch mode {
	case chaostypes.PodModeInject:
		args = []string{
			"network-latency",
			"inject",
			"--uid",
			string(uid),
			"--metrics-sink",
			sink,
			"--container-id",
			containerID,
			"--delay",
			strconv.Itoa(int(s.Delay)),
			"--hosts",
		}

		args = append(args, strings.Split(strings.Join(s.Hosts, " --hosts "), " ")...)

		if s.Port != 0 {
			args = append(args, "--port", strconv.Itoa(s.Port))
		}
	case chaostypes.PodModeClean:
		args = []string{
			"network-latency",
			"clean",
			"--uid",
			string(uid),
			"--metrics-sink",
			sink,
			"--container-id",
			containerID,
			"--hosts",
		}
		args = append(args, strings.Split(strings.Join(s.Hosts, " --hosts "), " ")...)
	}

	return args
}
