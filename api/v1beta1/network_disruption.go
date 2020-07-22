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

// NetworkDisruptionSpec represents a network disruption injection
type NetworkDisruptionSpec struct {
	// +nullable
	Hosts []string `json:"hosts,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65535
	Port     int    `json:"port,omitempty"`
	Protocol string `json:"protocol"`
	Drop     int    `json:"drop"`
	Corrupt  int    `json:"corrupt"`
	// +kubebuilder:validation:Maximum=59999
	Delay          uint `json:"delay"`
	BandwidthLimit int  `json:"bandwidthLimit"`
}

// GenerateArgs generates injection or cleanup pod arguments for the given spec
func (s *NetworkDisruptionSpec) GenerateArgs(mode chaostypes.PodMode, uid types.UID, containerID, sink string) []string {
	args := []string{}

	switch mode {
	case chaostypes.PodModeInject:
		args = []string{
			"network-disruption",
			"inject",
			"--uid",
			string(uid),
			"--metrics-sink",
			sink,
			"--container-id",
			containerID,
			"--port",
			strconv.Itoa(s.Port),
			"--corrupt",
			strconv.Itoa(s.Corrupt),
			"--drop",
			strconv.Itoa(s.Drop),
			"--delay",
			strconv.Itoa(int(s.Delay)),
			"--bandwidth-limit",
			strconv.Itoa(s.BandwidthLimit),
			"--protocol",
			s.Protocol,
			"--hosts",
		}
		args = append(args, strings.Split(strings.Join(s.Hosts, " --hosts "), " ")...)

	case chaostypes.PodModeClean:
		args = []string{
			"network-disruption",
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
