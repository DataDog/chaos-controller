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
	Port int `json:"port,omitempty"`
	// +kubebuilder:validation:Enum=tcp;udp;""
	Protocol string `json:"protocol,omitempty"`
	// +kubebuilder:validation:Enum=egress;ingress
	Flow string `json:"flow,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	Drop int `json:"drop,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	Duplicate int `json:"duplicate,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	Corrupt int `json:"corrupt,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=59999
	Delay uint `json:"delay,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	DelayJitter uint `json:"delayJitter,omitempty"`
	// +kubebuilder:validation:Minimum=0
	BandwidthLimit int `json:"bandwidthLimit,omitempty"`
}

// GenerateArgs generates injection or cleanup pod arguments for the given spec
func (s *NetworkDisruptionSpec) GenerateArgs(mode chaostypes.PodMode, uid types.UID, level chaostypes.DisruptionLevel, containerID, sink string) []string {
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
			"--level",
			string(level),
			"--container-id",
			containerID,
			"--port",
			strconv.Itoa(s.Port),
			"--corrupt",
			strconv.Itoa(s.Corrupt),
			"--drop",
			strconv.Itoa(s.Drop),
			"--duplicate",
			strconv.Itoa(s.Duplicate),
			"--delay",
			strconv.Itoa(int(s.Delay)),
			"--delayJitter",
			strconv.Itoa(int(s.DelayJitter)),
			"--bandwidth-limit",
			strconv.Itoa(s.BandwidthLimit),
		}

		// append protocol
		if s.Protocol != "" {
			args = append(args, "--protocol", s.Protocol)
		}

		// append hosts
		if len(s.Hosts) > 0 {
			args = append(args, "--hosts")
			args = append(args, strings.Split(strings.Join(s.Hosts, " --hosts "), " ")...)
		}

		// append flow
		if s.Flow != "" {
			args = append(args, "--flow", s.Flow)
		}
	case chaostypes.PodModeClean:
		args = []string{
			"network-disruption",
			"clean",
			"--uid",
			string(uid),
			"--metrics-sink",
			sink,
			"--level",
			string(level),
			"--container-id",
			containerID,
		}

		// append hosts
		if len(s.Hosts) > 0 {
			args = append(args, "--hosts")
			args = append(args, strings.Split(strings.Join(s.Hosts, " --hosts "), " ")...)
		}
	}

	return args
}
