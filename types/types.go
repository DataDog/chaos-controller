// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package types

// DisruptionKindName represents a disruption kind
type DisruptionKindName string

// DisruptionLevel represents which level the disruption should be injected at
type DisruptionLevel string

// DisruptionInjectionStatus represents the injection status of a disruption
type DisruptionInjectionStatus string

const (
	// TargetLabel is the label used to identify the pod targeted by a chaos pod
	TargetLabel = "chaos.datadoghq.com/target"
	// InjectHandlerLabel is the expected label when a chaos handler init container must be injected
	DisruptOnInitLabel = "chaos.datadoghq.com/disrupt-on-init"

	// DisruptionKindLabel is the label used to identify the disruption kind for a chaos pod
	DisruptionKindLabel = "chaos.datadoghq.com/disruption-kind"
	// DisruptionKindNetworkDisruption is a network failure disruption
	DisruptionKindNetworkDisruption = "network-disruption"
	// DisruptionKindNodeFailure is a node failure disruption
	DisruptionKindNodeFailure = "node-failure"
	// DisruptionKindContainerFailure is a container failure disruption
	DisruptionKindContainerFailure = "container-failure"
	// DisruptionKindCPUPressure is a CPU pressure disruption
	DisruptionKindCPUPressure = "cpu-pressure"
	// DisruptionKindDiskPressure is a disk pressure disruption
	DisruptionKindDiskPressure = "disk-pressure"
	// DisruptionKindDNSDisruption is a dns disruption
	DisruptionKindDNSDisruption = "dns-disruption"

	// DisruptionLevelUnspecified is the value used when the level of injection is not specified
	DisruptionLevelUnspecified = ""
	// DisruptionLevelPod is a disruption injected at the pod level
	DisruptionLevelPod = "pod"
	// DisruptionLevelNode is a disruption injected at the node level
	DisruptionLevelNode = "node"

	// DisruptionInjectionStatusNotInjected is the value of the injection status of a not yet injected disruption
	DisruptionInjectionStatusNotInjected DisruptionInjectionStatus = "NotInjected"
	// DisruptionInjectionStatusPartiallyInjected is the value of the injection status of a partially injected disruption
	DisruptionInjectionStatusPartiallyInjected DisruptionInjectionStatus = "PartiallyInjected"
	// DisruptionInjectionStatusInjected is the value of the injection status of a fully injected disruption
	DisruptionInjectionStatusInjected DisruptionInjectionStatus = "Injected"
	// DisruptionInjectionStatusPreviouslyInjected is the value of the injection status after the duration has expired
	DisruptionInjectionStatusPreviouslyInjected DisruptionInjectionStatus = "PreviouslyInjected"

	// DisruptionNameLabel is the label used to identify the disruption name for a chaos pod. This is used to determine pod ownership.
	DisruptionNameLabel = "chaos.datadoghq.com/disruption-name"
	// DisruptionNamespaceLabel is the label used to identify the disruption namespace for a chaos pod. This is used to determine pod ownership.
	DisruptionNamespaceLabel = "chaos.datadoghq.com/disruption-namespace"
)

var (
	// DisruptionKindNames contains all existing disruption kinds that can be injected
	DisruptionKindNames = []DisruptionKindName{
		DisruptionKindNetworkDisruption,
		DisruptionKindNodeFailure,
		DisruptionKindContainerFailure,
		DisruptionKindCPUPressure,
		DisruptionKindDiskPressure,
		DisruptionKindDNSDisruption,
	}
)
