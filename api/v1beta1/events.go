// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
)

const (
	EventOnTargetTemplate     string = "Failing probably caused by disruption %s: "
	SourceDisruptionComponent string = "disruption-controller"
)

type DisruptionEventCategory string

const (
	// Event only attached to a target
	TargetEvent DisruptionEventCategory = "TargetEvent"
	// Event only attached to the disruption
	DisruptEvent DisruptionEventCategory = "DisruptionEvent"
	// Event only attached to a chaos pod
	ChaosPodEvent DisruptionEventCategory = "ChaosPodEvent"
)

// DisruptionEventReason is the string that uniquely identify a disruption event
type DisruptionEventReason string

// MatchEventReason check if provided Kubernetes event match actual reason
func (r DisruptionEventReason) MatchEventReason(e corev1.Event) bool {
	return string(r) == e.Reason
}

type DisruptionEvent struct {
	Type                           string                  // Warning or Normal
	Reason                         DisruptionEventReason   // Short description of the event
	OnTargetTemplateMessage        string                  // Template message to attach to the target resource (pod or node). Empty if the event should not be sent to a target (DisruptEvent only)
	OnDisruptionTemplateMessage    string                  // We want to separate the aggregated message from the single message to include more info in the single message
	OnDisruptionTemplateAggMessage string                  // Template message to attach to the disruption. Empty if the event should not be sent on a disruption
	Category                       DisruptionEventCategory // Either TargetEvent, DisruptionEvent or ChaosPodEvent
}

// Complete list of events sent out by the controller
const (
	// Targeted pods related
	// Warning events
	EventTargetPodWarningState                      DisruptionEventReason = "TargetPodInWarningState"
	EventTargetContainerWarningState                DisruptionEventReason = "TargetPodContainersInWarningState"
	EventTargetLivenessProbeChange                  DisruptionEventReason = "TargetPodLivenessProbe"
	EventTargetTooManyRestarts                      DisruptionEventReason = "TargetPodTooManyRestarts"
	EventTargetReadinessProbeChangeBeforeDisruption DisruptionEventReason = "TargetReadinessProbeChangeBeforeDisruption"
	// Normal events
	EventTargetPodRecoveredState                    DisruptionEventReason = "RecoveredWarningStateInTargetPod"
	EventTargetReadinessProbeChangeDuringDisruption DisruptionEventReason = "ReadinessProbeChangeDuringDisruption"

	// Targeted nodes related
	// Warning events
	EventTargetNodeMemPressureState        DisruptionEventReason = "TargetNodeUnderMemoryPressure"
	EventTargetNodeDiskPressureState       DisruptionEventReason = "TargetNodeUnderDiskPressure"
	EventTargetNodeUnavailableNetworkState DisruptionEventReason = "TargetNodeUnavailableNetwork"
	EventTargetNodeWarningState            DisruptionEventReason = "TargetNodeInWarningState"
	// Normal events
	EventTargetNodeRecoveredState DisruptionEventReason = "RecoveredWarningStateInTargetNode"

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

	// Injection related events
	// Warning events
	EventChaosPodFailedState DisruptionEventReason = "ChaosPodWarningState"
)

var Events = map[DisruptionEventReason]DisruptionEvent{
	EventTargetPodWarningState: {
		Type:                           corev1.EventTypeWarning,
		Reason:                         EventTargetPodWarningState,
		OnDisruptionTemplateMessage:    "Targeted pod %s is failing",
		OnDisruptionTemplateAggMessage: "Targeted pod(s) are failing",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "pod is failing",
		Category:                       TargetEvent,
	},
	EventTargetContainerWarningState: {
		Type:                           corev1.EventTypeWarning,
		Reason:                         EventTargetContainerWarningState,
		OnDisruptionTemplateMessage:    "Container on targeted pod %s is failing",
		OnDisruptionTemplateAggMessage: "Containers on targeted pod(s) are failing",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "containers on pod are failing",
		Category:                       TargetEvent,
	},
	EventTargetLivenessProbeChange: {
		Type:                           corev1.EventTypeWarning,
		Reason:                         EventTargetLivenessProbeChange,
		OnDisruptionTemplateMessage:    "Liveness probe on targeted pod %s are failing",
		OnDisruptionTemplateAggMessage: "Liveness probe(s) on targeted pod(s) are failing",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "liveness probes on pod are failing",
		Category:                       TargetEvent,
	}, EventTargetReadinessProbeChangeDuringDisruption: {
		Type:                           corev1.EventTypeNormal,
		Reason:                         EventTargetReadinessProbeChangeDuringDisruption,
		OnDisruptionTemplateMessage:    "Readiness probe on targeted pod %s is failing",
		OnDisruptionTemplateAggMessage: "Readiness probes on targeted pod(s) are failing",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "readiness probes on pod are failing",
		Category:                       TargetEvent,
	}, EventTargetReadinessProbeChangeBeforeDisruption: {
		Type:                           corev1.EventTypeWarning,
		Reason:                         EventTargetReadinessProbeChangeBeforeDisruption,
		OnDisruptionTemplateMessage:    "Readiness probe on targeted pod %s is failing",
		OnDisruptionTemplateAggMessage: "Readiness probes on targeted pod(s) are failing",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "readiness probes on pod are failing",
		Category:                       TargetEvent,
	},
	EventTargetTooManyRestarts: {
		Type:                           corev1.EventTypeWarning,
		Reason:                         EventTargetTooManyRestarts,
		OnDisruptionTemplateMessage:    "Targeted pod %s has restarted too many times",
		OnDisruptionTemplateAggMessage: "Targeted pod(s) have restarted too many times",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "pod has restarted too many times",
		Category:                       TargetEvent,
	},
	EventTargetPodRecoveredState: {
		Type:                           corev1.EventTypeNormal,
		Reason:                         EventTargetPodRecoveredState,
		OnDisruptionTemplateMessage:    "Targeted pod %s seems to have recovered",
		OnDisruptionTemplateAggMessage: "Targeted pod(s) seem to have recovered",
		OnTargetTemplateMessage:        "pod seems to have recovered from the disruption %s failure",
		Category:                       TargetEvent,
	},
	EventTargetNodeMemPressureState: {
		Type:                           corev1.EventTypeWarning,
		Reason:                         EventTargetNodeMemPressureState,
		OnDisruptionTemplateMessage:    "Targeted node %s is under memory pressure",
		OnDisruptionTemplateAggMessage: "Targeted node(s) are under memory pressure",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "node is under memory pressure",
		Category:                       TargetEvent,
	},
	EventTargetNodeDiskPressureState: {
		Type:                           corev1.EventTypeWarning,
		Reason:                         EventTargetNodeDiskPressureState,
		OnDisruptionTemplateMessage:    "Targeted node %s is under disk pressure",
		OnDisruptionTemplateAggMessage: "Targeted node(s) are under disk pressure",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "node is under disk pressure",
		Category:                       TargetEvent,
	},
	EventTargetNodeUnavailableNetworkState: {
		Type:                           corev1.EventTypeWarning,
		Reason:                         EventTargetNodeUnavailableNetworkState,
		OnDisruptionTemplateMessage:    "Targeted node %s network is unavailable",
		OnDisruptionTemplateAggMessage: "Targeted node(s) network are unavailable",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "node network is unavaialble",
		Category:                       TargetEvent,
	},
	EventTargetNodeWarningState: {
		Type:                           corev1.EventTypeWarning,
		Reason:                         EventTargetNodeWarningState,
		OnDisruptionTemplateMessage:    "Targeted node %s is not ready",
		OnDisruptionTemplateAggMessage: "Targeted node(s) are not ready",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "node is not ready",
		Category:                       TargetEvent,
	},
	EventTargetNodeRecoveredState: {
		Type:                           corev1.EventTypeNormal,
		Reason:                         EventTargetNodeRecoveredState,
		OnDisruptionTemplateMessage:    "Targeted node %s seems to have recovered",
		OnDisruptionTemplateAggMessage: "Targeted node(s) seem to have recovered",
		OnTargetTemplateMessage:        "Node seems to have recovered from the disruption %s failure",
		Category:                       TargetEvent,
	},
	EventDisruptionDurationOver: {
		Type:                        corev1.EventTypeNormal,
		Reason:                      EventDisruptionDurationOver,
		OnDisruptionTemplateMessage: "The disruption has lived longer than its specified duration, and will be deleted in %s.",
		Category:                    DisruptEvent,
	},
	EventDisruptionGCOver: {
		Type:                        corev1.EventTypeNormal,
		Reason:                      EventDisruptionGCOver,
		OnDisruptionTemplateMessage: "The disruption has lived %s longer than its specified duration, and will now be deleted.",
		Category:                    DisruptEvent,
	},
	EventEmptyDisruption: {
		Type:                        corev1.EventTypeWarning,
		Reason:                      EventEmptyDisruption,
		OnDisruptionTemplateMessage: "No disruption recognized for \"%s\" therefore no disruption applied.",
		Category:                    DisruptEvent,
	},
	EventDisruptionCreationFailed: {
		Type:                        corev1.EventTypeWarning,
		Reason:                      EventDisruptionCreationFailed,
		OnDisruptionTemplateMessage: "Injection pod for disruption \"%s\" failed to be created",
		Category:                    DisruptEvent,
	},
	EventDisruptionStuckOnRemoval: {
		Type:                        corev1.EventTypeWarning,
		Reason:                      EventDisruptionStuckOnRemoval,
		OnDisruptionTemplateMessage: "Instance is stuck on removal because of chaos pods not being able to terminate correctly, please check pods logs before manually removing their finalizer. https://github.com/DataDog/chaos-controller/blob/main/docs/faq.md",
		Category:                    DisruptEvent,
	},
	EventInvalidDisruptionLabelSelector: {
		Type:                        corev1.EventTypeWarning,
		Reason:                      EventInvalidDisruptionLabelSelector,
		OnDisruptionTemplateMessage: "%s. No targets will be selected.",
		Category:                    DisruptEvent,
	},
	EventDisruptionNoMoreValidTargets: {
		Type:                        corev1.EventTypeNormal,
		Reason:                      EventDisruptionNoMoreValidTargets,
		OnDisruptionTemplateMessage: "No more targets found for injection for this disruption (either ignored or already targeted by another disruption)",
		Category:                    DisruptEvent,
	},
	EventDisruptionNoTargetsFound: {
		Type:                        corev1.EventTypeWarning,
		Reason:                      EventDisruptionNoTargetsFound,
		OnDisruptionTemplateMessage: "The given label selector did not return any targets. Please ensure that both the selector and the count are correct (should be either a percentage or an integer greater than 0).",
		Category:                    DisruptEvent,
	},
	EventInvalidSpecDisruption: {
		Type:                        corev1.EventTypeWarning,
		Reason:                      EventInvalidSpecDisruption,
		OnDisruptionTemplateMessage: "%s",
		Category:                    DisruptEvent,
	},
	EventDisruptionChaosPodCreated: {
		Type:                        corev1.EventTypeNormal,
		Reason:                      EventDisruptionChaosPodCreated,
		OnDisruptionTemplateMessage: "Created disruption injection pod for \"%s\"",
		Category:                    DisruptEvent,
	},
	EventDisruptionCreated: {
		Type:                        corev1.EventTypeNormal,
		Reason:                      EventDisruptionCreated,
		OnDisruptionTemplateMessage: "Disruption created",
		Category:                    DisruptEvent,
	},
	EventDisruptionFinished: {
		Type:                        corev1.EventTypeNormal,
		Reason:                      EventDisruptionFinished,
		OnDisruptionTemplateMessage: "Disruption finished",
		Category:                    DisruptEvent,
	},
	EventDisrupted: {
		Type:                    corev1.EventTypeNormal,
		Reason:                  EventDisrupted,
		OnTargetTemplateMessage: "Pod %s from disruption %s targeted this resource for injection",
		Category:                DisruptEvent,
	},
	EventChaosPodFailedState: {
		Type:                           corev1.EventTypeWarning,
		Reason:                         EventChaosPodFailedState,
		OnDisruptionTemplateMessage:    "Chaos pod %s is not ready. Phase: '%s'. Reason: '%s'",
		OnDisruptionTemplateAggMessage: "Chaos pod(s) are not ready",
		Category:                       ChaosPodEvent,
	},
}

// IsNotifiableEvent this event can be broadcasted to our notifiers
func IsNotifiableEvent(event corev1.Event) bool {
	return event.Source.Component == SourceDisruptionComponent
}

func IsRecoveryEvent(event corev1.Event) bool {
	return EventTargetNodeRecoveredState.MatchEventReason(event) || EventTargetPodRecoveredState.MatchEventReason(event)
}

func IsTargetEvent(event corev1.Event) bool {
	targetEvent, ok := Events[DisruptionEventReason(event.Reason)]
	if !ok {
		return false
	}

	return event.Source.Component == SourceDisruptionComponent &&
		targetEvent.Category == TargetEvent
}
