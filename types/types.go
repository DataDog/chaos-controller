// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package types

// DisruptionKind represents a disruption kind
type DisruptionKind string

// DisruptionLevel represents which level the disruption should be injected at
type DisruptionLevel string

const (
	// TargetLabel is the label used to identify the pod targeted by a chaos pod
	TargetLabel = "chaos.datadoghq.com/target"
	// TargetPodHostIPEnv is the target pod host IP environment variable name
	TargetPodHostIPEnv = "TARGET_POD_HOST_IP"

	// DisruptionKindLabel is the label used to identify the disruption kind for a chaos pod
	DisruptionKindLabel = "chaos.datadoghq.com/disruption-kind"
	// DisruptionKindNetworkDisruption is a network failure disruption
	DisruptionKindNetworkDisruption = "network-disruption"
	// DisruptionKindNodeFailure is a node failure disruption
	DisruptionKindNodeFailure = "node-failure"
	// DisruptionKindCPUPressure is a CPU pressure disruption
	DisruptionKindCPUPressure = "cpu-pressure"
	// DisruptionKindDiskPressure is a disk pressure disruption
	DisruptionKindDiskPressure = "disk-pressure"

	// DisruptionLevelUnspecified is the value used when the level of injection is not specified
	DisruptionLevelUnspecified = ""
	// DisruptionLevelPod is a disruption injected at the pod level
	DisruptionLevelPod = "pod"
	// DisruptionLevelNode is a disruption injected at the node level
	DisruptionLevelNode = "node"
)

var (
	// DisruptionKinds contains all existing disruption kinds that can be injected
	DisruptionKinds = []DisruptionKind{
		DisruptionKindNetworkDisruption,
		DisruptionKindNodeFailure,
		DisruptionKindCPUPressure,
		DisruptionKindDiskPressure,
	}
)
