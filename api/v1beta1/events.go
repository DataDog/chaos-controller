// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
)

const EventOnTargetTemplate string = "Failing probably caused by disruption %s: "
const SourceDisruptionComponent string = "disruption-controller"

type DisruptionEventCategory string

const (
	// Event only attached to a target
	TargetEvent DisruptionEventCategory = "TargetEvent"
	// Event only attached to the disruption
	DisruptEvent DisruptionEventCategory = "DisruptionEvent"
)

type DisruptionEvent struct {
	Type                           string                  // Warning or Normal
	OnTargetTemplateMessage        string                  // Template message to attach to the target resource (pod or node). Empty if the event should not be sent to a target (DisruptEvent only)
	OnDisruptionTemplateMessage    string                  // We want to separate the aggregated message from the single message to include more info in the single message
	OnDisruptionTemplateAggMessage string                  // Template message to attach to the disruption. Empty if the event should not be sent on a disruption
	Category                       DisruptionEventCategory // Either TargetEvent or DisruptionEvent
}

// Complete list of events sent out by the controller
type DisruptionEventReason string
type KubernetesEventReason string

// Custom events from disruption-controller
const (
	// Targeted pods related
	// Warning events
	EventPodWarningState                      DisruptionEventReason = "TargetPodInWarningState"
	EventContainerWarningState                DisruptionEventReason = "TargetPodContainersInWarningState"
	EventLivenessProbeChange                  DisruptionEventReason = "TargetPodLivenessProbe"
	EventTooManyRestarts                      DisruptionEventReason = "TargetPodTooManyRestarts"
	EventReadinessProbeChangeBeforeDisruption DisruptionEventReason = "EventReadinessProbeChangeBeforeDisruption"
	// Normal events
	EventPodRecoveredState                    DisruptionEventReason = "RecoveredWarningStateInTargetPod"
	EventReadinessProbeChangeDuringDisruption DisruptionEventReason = "EventReadinessProbeChangeDuringDisruption"

	// Targeted nodes related
	// Warning events
	EventNodeMemPressureState        DisruptionEventReason = "TargetNodeUnderMemoryPressure"
	EventNodeDiskPressureState       DisruptionEventReason = "TargetNodeUnderDiskPressure"
	EventNodeUnavailableNetworkState DisruptionEventReason = "TargetNodeUnavailableNetwork"
	EventNodeWarningState            DisruptionEventReason = "TargetNodeInWarningState"
	// Normal events
	EventNodeRecoveredState DisruptionEventReason = "RecoveredWarningStateInTargetNode"

	// Disruption related events
	// Warning events
	EventEmptyDisruption                DisruptionEventReason = "EmptyDisruption"
	EventDisruptionCreationFailed       DisruptionEventReason = "CreateFailed"
	EventDisruptionStuckOnRemoval       DisruptionEventReason = "StuckOnRemoval"
	EventInvalidDisruptionLabelSelector DisruptionEventReason = "InvalidLabelSelector"
	EventDisruptionNoMoreValidTargets   DisruptionEventReason = "NoMoreTargets"
	EventDisruptionNoTargetsFound       DisruptionEventReason = "NoTargetsFound"
	EventInvalidSpecDisruption          DisruptionEventReason = "InvalidSpec"
	// Normal events
	EventDisruptionChaosPodCreated DisruptionEventReason = "ChaosPodCreated"
	EventDisruptionFinished        DisruptionEventReason = "Finished"
	EventDisruptionCreated         DisruptionEventReason = "Created"
	EventDisruptionDurationOver    DisruptionEventReason = "DurationOver"
	EventDisruptionGCOver          DisruptionEventReason = "GCOver"
	EventDisrupted                 DisruptionEventReason = "Disrupted"
)

// Events from https://github.com/kubernetes/kubernetes/blob/v1.25.3/pkg/kubelet/events/event.go. No import possible
const (
	// Container event reason list
	CreatedContainer        KubernetesEventReason = "Created"
	StartedContainer        KubernetesEventReason = "Started"
	FailedToCreateContainer KubernetesEventReason = "Failed"
	FailedToStartContainer  KubernetesEventReason = "Failed"
	KillingContainer        KubernetesEventReason = "Killing"
	PreemptContainer        KubernetesEventReason = "Preempting"
	BackOffStartContainer   KubernetesEventReason = "BackOff"
	ExceededGracePeriod     KubernetesEventReason = "ExceededGracePeriod"
	// Pod event reason list
	FailedToKillPod                KubernetesEventReason = "FailedKillPod"
	FailedToCreatePodContainer     KubernetesEventReason = "FailedCreatePodContainer"
	FailedToMakePodDataDirectories KubernetesEventReason = "Failed"
	NetworkNotReady                KubernetesEventReason = "NetworkNotReady"
	// Image event reason list
	PullingImage            KubernetesEventReason = "Pulling"
	PulledImage             KubernetesEventReason = "Pulled"
	FailedToPullImage       KubernetesEventReason = "Failed"
	FailedToInspectImage    KubernetesEventReason = "InspectFailed"
	ErrImageNeverPullPolicy KubernetesEventReason = "ErrImageNeverPull"
	BackOffPullImage        KubernetesEventReason = "BackOff"
	// kubelet event reason list
	NodeReady                            KubernetesEventReason = "NodeReady"
	NodeNotReady                         KubernetesEventReason = "NodeNotReady"
	NodeSchedulable                      KubernetesEventReason = "NodeSchedulable"
	NodeNotSchedulable                   KubernetesEventReason = "NodeNotSchedulable"
	StartingKubelet                      KubernetesEventReason = "Starting"
	KubeletSetupFailed                   KubernetesEventReason = "KubeletSetupFailed"
	FailedAttachVolume                   KubernetesEventReason = "FailedAttachVolume"
	FailedMountVolume                    KubernetesEventReason = "FailedMount"
	VolumeResizeFailed                   KubernetesEventReason = "VolumeResizeFailed"
	VolumeResizeSuccess                  KubernetesEventReason = "VolumeResizeSuccessful"
	FileSystemResizeFailed               KubernetesEventReason = "FileSystemResizeFailed"
	FileSystemResizeSuccess              KubernetesEventReason = "FileSystemResizeSuccessful"
	FailedMapVolume                      KubernetesEventReason = "FailedMapVolume"
	WarnAlreadyMountedVolume             KubernetesEventReason = "AlreadyMountedVolume"
	SuccessfulAttachVolume               KubernetesEventReason = "SuccessfulAttachVolume"
	SuccessfulMountVolume                KubernetesEventReason = "SuccessfulMountVolume"
	NodeRebooted                         KubernetesEventReason = "Rebooted"
	NodeShutdown                         KubernetesEventReason = "Shutdown"
	ContainerGCFailed                    KubernetesEventReason = "ContainerGCFailed"
	ImageGCFailed                        KubernetesEventReason = "ImageGCFailed"
	FailedNodeAllocatableEnforcement     KubernetesEventReason = "FailedNodeAllocatableEnforcement"
	SuccessfulNodeAllocatableEnforcement KubernetesEventReason = "NodeAllocatableEnforced"
	SandboxChanged                       KubernetesEventReason = "SandboxChanged"
	FailedCreatePodSandBox               KubernetesEventReason = "FailedCreatePodSandBox"
	FailedStatusPodSandBox               KubernetesEventReason = "FailedPodSandBoxStatus"
	FailedMountOnFilesystemMismatch      KubernetesEventReason = "FailedMountOnFilesystemMismatch"
	// Image manager event reason list
	InvalidDiskCapacity KubernetesEventReason = "InvalidDiskCapacity"
	FreeDiskSpaceFailed KubernetesEventReason = "FreeDiskSpaceFailed"
	// Probe event reason list
	ContainerUnhealthy    KubernetesEventReason = "Unhealthy"
	ContainerProbeWarning KubernetesEventReason = "ProbeWarning"
	// Pod worker event reason list
	FailedSync KubernetesEventReason = "FailedSync"
	// Config event reason list
	FailedValidation KubernetesEventReason = "FailedValidation"
	// Lifecycle hooks
	FailedPostStartHook KubernetesEventReason = "FailedPostStartHook"
	FailedPreStopHook   KubernetesEventReason = "FailedPreStopHook"
)

var Events = map[DisruptionEventReason]DisruptionEvent{
	EventPodWarningState: {
		Type:                           corev1.EventTypeWarning,
		OnDisruptionTemplateMessage:    "Targeted pod %s is failing",
		OnDisruptionTemplateAggMessage: "Targeted pod(s) are failing",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "pod is failing",
		Category:                       TargetEvent,
	},
	EventContainerWarningState: {
		Type:                           corev1.EventTypeWarning,
		OnDisruptionTemplateMessage:    "Container on targeted pod %s is failing",
		OnDisruptionTemplateAggMessage: "Containers on targeted pod(s) are failing",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "containers on pod are failing",
		Category:                       TargetEvent,
	},
	EventLivenessProbeChange: {
		Type:                           corev1.EventTypeWarning,
		OnDisruptionTemplateMessage:    "Liveness probe on targeted pod %s are failing",
		OnDisruptionTemplateAggMessage: "Liveness probe(s) on targeted pod(s) are failing",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "liveness probes on pod are failing",
		Category:                       TargetEvent,
	}, EventReadinessProbeChangeDuringDisruption: {
		Type:                           corev1.EventTypeNormal,
		OnDisruptionTemplateMessage:    "Readiness probe on targeted pod %s is failing",
		OnDisruptionTemplateAggMessage: "Readiness probes on targeted pod(s) are failing",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "readiness probes on pod are failing",
		Category:                       TargetEvent,
	}, EventReadinessProbeChangeBeforeDisruption: {
		Type:                           corev1.EventTypeWarning,
		OnDisruptionTemplateMessage:    "Readiness probe on targeted pod %s is failing",
		OnDisruptionTemplateAggMessage: "Readiness probes on targeted pod(s) are failing",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "readiness probes on pod are failing",
		Category:                       TargetEvent,
	},
	EventTooManyRestarts: {
		Type:                           corev1.EventTypeWarning,
		OnDisruptionTemplateMessage:    "Targeted pod %s has restarted too many times",
		OnDisruptionTemplateAggMessage: "Targeted pod(s) have restarted too many times",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "pod has restarted too many times",
		Category:                       TargetEvent,
	},
	EventPodRecoveredState: {
		Type:                           corev1.EventTypeNormal,
		OnDisruptionTemplateMessage:    "Targeted pod %s seems to have recovered",
		OnDisruptionTemplateAggMessage: "Targeted pod(s) seem to have recovered",
		OnTargetTemplateMessage:        "pod seems to have recovered from the disruption %s failure",
		Category:                       TargetEvent,
	},
	EventNodeMemPressureState: {
		Type:                           corev1.EventTypeWarning,
		OnDisruptionTemplateMessage:    "Targeted node %s is under memory pressure",
		OnDisruptionTemplateAggMessage: "Targeted node(s) are under memory pressure",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "node is under memory pressure",
		Category:                       TargetEvent,
	},
	EventNodeDiskPressureState: {
		Type:                           corev1.EventTypeWarning,
		OnDisruptionTemplateMessage:    "Targeted node %s is under disk pressure",
		OnDisruptionTemplateAggMessage: "Targeted node(s) are under disk pressure",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "node is under disk pressure",
		Category:                       TargetEvent,
	},
	EventNodeUnavailableNetworkState: {
		Type:                           corev1.EventTypeWarning,
		OnDisruptionTemplateMessage:    "Targeted node %s network is unavailable",
		OnDisruptionTemplateAggMessage: "Targeted node(s) network are unavailable",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "node network is unavaialble",
		Category:                       TargetEvent,
	},
	EventNodeWarningState: {
		Type:                           corev1.EventTypeWarning,
		OnDisruptionTemplateMessage:    "Targeted node %s is not ready",
		OnDisruptionTemplateAggMessage: "Targeted node(s) are not ready",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "node is not ready",
		Category:                       TargetEvent,
	},
	EventNodeRecoveredState: {
		Type:                           corev1.EventTypeNormal,
		OnDisruptionTemplateMessage:    "Targeted node %s seems to have recovered",
		OnDisruptionTemplateAggMessage: "Targeted node(s) seem to have recovered",
		OnTargetTemplateMessage:        "Node seems to have recovered from the disruption %s failure",
		Category:                       TargetEvent,
	},
	EventDisruptionDurationOver: {
		Type:                        corev1.EventTypeNormal,
		OnDisruptionTemplateMessage: "The disruption has lived longer than its specified duration, and will be deleted in %s.",
		Category:                    DisruptEvent,
	},
	EventDisruptionGCOver: {
		Type:                        corev1.EventTypeNormal,
		OnDisruptionTemplateMessage: "The disruption has lived %s longer than its specified duration, and will now be deleted.",
		Category:                    DisruptEvent,
	},
	EventEmptyDisruption: {
		Type:                        corev1.EventTypeWarning,
		OnDisruptionTemplateMessage: "No disruption recognized for \"%s\" therefore no disruption applied.",
		Category:                    DisruptEvent,
	},
	EventDisruptionCreationFailed: {
		Type:                        corev1.EventTypeWarning,
		OnDisruptionTemplateMessage: "Injection pod for disruption \"%s\" failed to be created",
		Category:                    DisruptEvent,
	},
	EventDisruptionStuckOnRemoval: {
		Type:                        corev1.EventTypeWarning,
		OnDisruptionTemplateMessage: "Instance is stuck on removal because of chaos pods not being able to terminate correctly, please check pods logs before manually removing their finalizer. https://github.com/DataDog/chaos-controller/blob/main/docs/faq.md",
		Category:                    DisruptEvent,
	},
	EventInvalidDisruptionLabelSelector: {
		Type:                        corev1.EventTypeWarning,
		OnDisruptionTemplateMessage: "%s. No targets will be selected.",
		Category:                    DisruptEvent,
	},
	EventDisruptionNoMoreValidTargets: {
		Type:                        corev1.EventTypeNormal,
		OnDisruptionTemplateMessage: "No more targets found for injection for this disruption (either ignored or already targeted by another disruption)",
		Category:                    DisruptEvent,
	},
	EventDisruptionNoTargetsFound: {
		Type:                        corev1.EventTypeWarning,
		OnDisruptionTemplateMessage: "The given label selector did not return any targets. Please ensure that both the selector and the count are correct (should be either a percentage or an integer greater than 0).",
		Category:                    DisruptEvent,
	},
	EventInvalidSpecDisruption: {
		Type:                        corev1.EventTypeWarning,
		OnDisruptionTemplateMessage: "%s",
		Category:                    DisruptEvent,
	},
	EventDisruptionChaosPodCreated: {
		Type:                        corev1.EventTypeNormal,
		OnDisruptionTemplateMessage: "Created disruption injection pod for \"%s\"",
		Category:                    DisruptEvent,
	},
	EventDisruptionCreated: {
		Type:                        corev1.EventTypeNormal,
		OnDisruptionTemplateMessage: "Disruption created",
		Category:                    DisruptEvent,
	},
	EventDisruptionFinished: {
		Type:                        corev1.EventTypeNormal,
		OnDisruptionTemplateMessage: "Disruption finished",
		Category:                    DisruptEvent,
	},
	EventDisrupted: {
		Type:                    corev1.EventTypeNormal,
		OnTargetTemplateMessage: "Pod %s from disruption %s targeted this resource for injection",
		Category:                DisruptEvent,
	},
}

// IsNotifiableEvent this event can be broadcasted to our notifiers
func IsNotifiableEvent(event corev1.Event) bool {
	return event.Source.Component == SourceDisruptionComponent
}

func CompareCustom(reason string, toCompareReason DisruptionEventReason) bool {
	return string(toCompareReason) == reason
}

func CompareK8S(reason string, toCompareReason KubernetesEventReason) bool {
	return string(toCompareReason) == reason
}

func IsRecoveryEvent(event corev1.Event) bool {
	return CompareCustom(event.Reason, EventNodeRecoveredState) || CompareCustom(event.Reason, EventPodRecoveredState)
}

func IsTargetEvent(event corev1.Event) bool {
	reason := DisruptionEventReason(event.Reason)

	if reason == "" {
		return false
	}

	return event.Source.Component == SourceDisruptionComponent &&
		Events[reason].Category == TargetEvent
}
