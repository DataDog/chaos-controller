// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package types

import (
	"time"
)

// DisruptionKindName represents a disruption kind
type DisruptionKindName string

// DisruptionLevel represents which level the disruption should be injected at
type DisruptionLevel string

// DisruptionInjectionStatus represents the injection status of a disruption
type DisruptionInjectionStatus string

func (i DisruptionInjectionStatus) Previously() bool {
	switch i {
	case DisruptionInjectionStatusPreviouslyInjected,
		DisruptionInjectionStatusPreviouslyNotInjected,
		DisruptionInjectionStatusPreviouslyPartiallyInjected:
		return true
	}

	return false
}

// NeverInjected return true if the disruption has never been injected at all
func (i DisruptionInjectionStatus) NeverInjected() bool {
	return i == DisruptionInjectionStatusInitial || i == DisruptionInjectionStatusNotInjected
}

// NotFullyInjected return true if the status enables more pods to be injected, false otherwise
func (i DisruptionInjectionStatus) NotFullyInjected() bool {
	switch i {
	case DisruptionInjectionStatusInitial,
		DisruptionInjectionStatusNotInjected,
		DisruptionInjectionStatusPartiallyInjected,
		DisruptionInjectionStatusPausedInjected,
		DisruptionInjectionStatusPausedPartiallyInjected:
		return true
	}

	return false
}

// DisruptionTargetInjectionStatus represents the injection status of the target of a disruption
type DisruptionTargetInjectionStatus string

const (
	GroupName = "chaos.datadoghq.com"
	// TargetLabel is the label used to identify the pod targeted by a chaos pod
	TargetLabel = GroupName + "/target"
	// InjectHandlerLabel is the expected label when a chaos handler init container must be injected
	DisruptOnInitLabel = GroupName + "/disrupt-on-init"

	// MultiDistruptionAllowed is the expected annotation to put on a pod to enable multi disruption
	MultiDistruptionAllowed = GroupName + "/multi-disruption-allowed"

	// DisruptionKindLabel is the label used to identify the disruption kind for a chaos pod
	DisruptionKindLabel = GroupName + "/disruption-kind"
	// DisruptionKindNetworkDisruption is a network failure disruption
	DisruptionKindNetworkDisruption = "network-disruption"
	// DisruptionKindNodeFailure is a node failure disruption
	DisruptionKindNodeFailure = "node-failure"
	// DisruptionKindContainerFailure is a container failure disruption
	DisruptionKindContainerFailure = "container-failure"
	// DisruptionKindCPUPressure is a CPU pressure disruption
	DisruptionKindCPUPressure = "cpu-pressure"
	// DisruptionKindCPUStress is a CPU pressure sub-disruption that stress a single container
	DisruptionKindCPUStress = "cpu-pressure-stress"
	// DisruptionKindDiskFailure is a disk failure disruption
	DisruptionKindDiskFailure = "disk-failure"
	// DisruptionKindDiskPressure is a disk pressure disruption
	DisruptionKindDiskPressure = "disk-pressure"
	// DisruptionKindDNSDisruption is a dns disruption
	DisruptionKindDNSDisruption = "dns-disruption"
	// DisruptionKindGRPCDisruption is a grpc disruption
	DisruptionKindGRPCDisruption = "grpc-disruption"

	// DisruptionLevelPod is a disruption injected at the pod level
	DisruptionLevelPod DisruptionLevel = "pod"
	// DisruptionLevelNode is a disruption injected at the node level
	DisruptionLevelNode DisruptionLevel = "node"

	// DisruptionInjectionStatusInitial is the initial injection status before anything is happening
	DisruptionInjectionStatusInitial DisruptionInjectionStatus = ""
	// DisruptionInjectionStatusNotInjected is the value of the injection status of a not yet injected disruption
	DisruptionInjectionStatusNotInjected DisruptionInjectionStatus = "NotInjected"
	// DisruptionInjectionStatusPartiallyInjected is the value of the injection status of a partially injected disruption
	DisruptionInjectionStatusPartiallyInjected DisruptionInjectionStatus = "PartiallyInjected"
	// DisruptionInjectionStatusInjected is the value of the injection status of a fully injected disruption
	DisruptionInjectionStatusInjected DisruptionInjectionStatus = "Injected"
	// DisruptionInjectionStatusPausedPartiallyInjected is the value of the injection when the disruption was partially injected and later, all targets disappeared AND disruption has not yet expired or being deleted.
	// It imply the disruption could be re-injected at some point if new targets matching provided selector appears
	DisruptionInjectionStatusPausedPartiallyInjected DisruptionInjectionStatus = "PausedPartiallyInjected"
	// DisruptionInjectionStatusPausedInjected is the value of the injection status when the disruption was injected but is no longer due to target disappearance (and duration has not expired and disruption is not deleted)
	// It imply the disruption could be re-injected at some point if new targets matching provided selector appears
	DisruptionInjectionStatusPausedInjected DisruptionInjectionStatus = "PausedInjected"
	// DisruptionInjectionStatusPreviouslyNotInjected is the value of the injection status after the duration has expired and the disruption was not injected
	DisruptionInjectionStatusPreviouslyNotInjected DisruptionInjectionStatus = "PreviouslyNotInjected"
	// DisruptionInjectionStatusPreviouslyPartiallyInjected is the value of the injection status after the duration has expired and the disruption was partially injected
	DisruptionInjectionStatusPreviouslyPartiallyInjected DisruptionInjectionStatus = "PreviouslyPartiallyInjected"
	// DisruptionInjectionStatusPreviouslyInjected is the value of the injection status after the duration has expired
	DisruptionInjectionStatusPreviouslyInjected DisruptionInjectionStatus = "PreviouslyInjected"

	// DisruptionTargetInjectionStatusNotInjected is the value of the injection status of a not yet injected disruption into the target
	DisruptionTargetInjectionStatusNotInjected DisruptionTargetInjectionStatus = "NotInjected"
	// DisruptionInjectionStatusInjected is the value of the injection status when the injection has been injected into the target
	DisruptionTargetInjectionStatusInjected DisruptionTargetInjectionStatus = "Injected"
	// DisruptionInjectionStatusIsStuckOnRemoval is the value of the injection status when the injection could not be removed on the target
	DisruptionTargetInjectionStatusStatusIsStuckOnRemoval DisruptionTargetInjectionStatus = "IsStuckOnRemoval"

	// DisruptionNameLabel is the label used to identify the disruption name for a chaos pod. This is used to determine pod ownership.
	DisruptionNameLabel = GroupName + "/disruption-name"
	// DisruptionNamespaceLabel is the label used to identify the disruption namespace for a chaos pod. This is used to determine pod ownership.
	DisruptionNamespaceLabel = GroupName + "/disruption-namespace"

	// ScheduledAtAnnotation is the annotation key for the scheduled time of the disruption when it is created from DisruptionCron or DisruptionRollout.
	ScheduledAtAnnotation = GroupName + "/scheduled-at"

	// UserAnnotation is the annotation key that stores the username of the user who created the parent resource (DisruptionCron or DisruptionRollout).
	UserAnnotation = GroupName + "/user"
	// UserGroupsAnnotation is the annotation key that stores the user groups of the individual who created the parent resource (DisruptionCron or DisruptionRollout).
	UserGroupsAnnotation = GroupName + "/user-groups"

	finalizerPrefix         = "finalizer." + GroupName
	DisruptionFinalizer     = finalizerPrefix
	DisruptionCronFinalizer = finalizerPrefix
	ChaosPodFinalizer       = finalizerPrefix + "/chaos-pod"

	PulsingDisruptionMinimumDuration = 500 * time.Millisecond
	// InjectorPadDuration is the length of time we extend the injector's duration on top of the disruption's duration,
	// in order to ensure the manager stops the disruption prior to the injectors dying
	InjectorPadDuration = 2 * time.Second

	// InjectorCgroupClassID is linked to the TC tree in the injector network disruption.
	// Also used in the DNS Disruption to allow combined Network + DNS Disruption
	// This value should NEVER be changed without changing the Network Disruption TC tree.
	InjectorCgroupClassID = "0x00020002"
)

func (d DisruptionKindName) String() string {
	return string(d)
}

// DisruptionKindNames contains all existing disruption kinds that can be injected
var DisruptionKindNames = []DisruptionKindName{
	DisruptionKindNetworkDisruption,
	DisruptionKindNodeFailure,
	DisruptionKindContainerFailure,
	DisruptionKindCPUPressure,
	DisruptionKindDiskPressure,
	DisruptionKindDiskFailure,
	DisruptionKindDNSDisruption,
	DisruptionKindGRPCDisruption,
}
