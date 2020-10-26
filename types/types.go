// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package types

// PodMode represents an enum of possible chaos pod modes
type PodMode string

// DisruptionKind represents a disruption kind
type DisruptionKind string

const (
	// PodModeLabel is the label used to identify the pod mode
	PodModeLabel = "chaos.datadoghq.com/pod-mode"
	// PodModeInject mode
	PodModeInject = "inject"
	// PodModeClean mode
	PodModeClean = "clean"

	// TargetPodLabel is the label used to identify the pod targeted by a chaos pod
	TargetPodLabel = "chaos.datadoghq.com/target-pod"
	// TargetPodHostIPEnv is the target pod host IP environment variable name
	TargetPodHostIPEnv = "TARGET_POD_HOST_IP"
	// TargetPodIPEnv is the target pod IP environment variable name
	TargetPodIPEnv = "TARGET_POD_IP"

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
)
