// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package v1beta1

import (
	"errors"
	"strconv"
	"strings"
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

// Validate validates args for the given disruption
func (s *NetworkDisruptionSpec) Validate() error {
	if s.BandwidthLimit == 0 &&
		s.Drop == 0 &&
		s.Delay == 0 &&
		s.Corrupt == 0 &&
		s.Duplicate == 0 {
		return errors.New("the network disruption was selected, but no disruption type was specified. Please set at least one of: drop, delay, bandwidthLimit, corrupt, or duplicate. No injection will occur")
	}

	return nil
}

// GenerateArgs generates injection or cleanup pod arguments for the given spec
func (s *NetworkDisruptionSpec) GenerateArgs() []string {
	args := []string{
		"network-disruption",
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
		"--delay-jitter",
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

	return args
}
