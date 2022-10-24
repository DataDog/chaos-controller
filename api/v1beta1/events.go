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
	TargetEvent  DisruptionEventCategory = "TargetEvent"
	DisruptEvent DisruptionEventCategory = "DisruptionEvent"
)

type DisruptionEvent struct {
	Type                           string
	Reason                         string
	OnTargetTemplateMessage        string
	OnDisruptionTemplateMessage    string // We want to separate the aggregated message from the single message to include more info in the single message
	OnDisruptionTemplateAggMessage string
	Category                       DisruptionEventCategory
}

// Complete list of events sent out by the controller

const (
	// Targeted pods related
	// Warning events
	EventPodWarningState                      string = "TargetPodInWarningState"
	EventContainerWarningState                string = "TargetPodContainersInWarningState"
	EventLivenessProbeChange                  string = "TargetPodLivenessProbe"
	EventTooManyRestarts                      string = "TargetPodTooManyRestarts"
	EventReadinessProbeChangeBeforeDisruption string = "EventReadinessProbeChangeBeforeDisruption"
	// Normal events
	EventPodRecoveredState                    string = "RecoveredWarningStateInTargetPod"
	EventReadinessProbeChangeDuringDisruption string = "EventReadinessProbeChangeDuringDisruption"

	// Targeted nodes related
	// Warning events
	EventNodeMemPressureState        string = "TargetNodeUnderMemoryPressure"
	EventNodeDiskPressureState       string = "TargetNodeUnderDiskPressure"
	EventNodeUnavailableNetworkState string = "TargetNodeUnavailableNetwork"
	EventNodeWarningState            string = "TargetNodeInWarningState"
	// Normal events
	EventNodeRecoveredState string = "RecoveredWarningStateInTargetNode"

	// Disruption related events
	// Warning events
	EventEmptyDisruption                string = "EmptyDisruption"
	EventDisruptionCreationFailed       string = "CreateFailed"
	EventDisruptionStuckOnRemoval       string = "StuckOnRemoval"
	EventInvalidDisruptionLabelSelector string = "InvalidLabelSelector"
	EventDisruptionNoMoreValidTargets   string = "NoMoreTargets"
	EventDisruptionNoTargetsFound       string = "NoTargetsFound"
	EventInvalidSpecDisruption          string = "InvalidSpec"
	// Normal events
	EventDisruptionChaosPodCreated string = "ChaosPodCreated"
	EventDisruptionFinished        string = "Finished"
	EventDisruptionCreated         string = "Created"
	EventDisruptionDurationOver    string = "DurationOver"
	EventDisruptionGCOver          string = "GCOver"
	EventDisrupted                 string = "Disrupted"
)

var Events = map[string]DisruptionEvent{
	EventPodWarningState: {
		Type:                           corev1.EventTypeWarning,
		Reason:                         EventPodWarningState,
		OnDisruptionTemplateMessage:    "targeted pod %s is failing",
		OnDisruptionTemplateAggMessage: "targeted pod(s) are failing",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "pod is failing",
		Category:                       TargetEvent,
	},
	EventContainerWarningState: {
		Type:                           corev1.EventTypeWarning,
		Reason:                         EventContainerWarningState,
		OnDisruptionTemplateMessage:    "container on targeted pod %s is failing",
		OnDisruptionTemplateAggMessage: "containers on targeted pod(s) are failing",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "containers on pod are failing",
		Category:                       TargetEvent,
	},
	EventLivenessProbeChange: {
		Type:                           corev1.EventTypeWarning,
		Reason:                         EventLivenessProbeChange,
		OnDisruptionTemplateMessage:    "liveness probe on targeted pod %s are failing",
		OnDisruptionTemplateAggMessage: "liveness probe(s) on targeted pod(s) are failing",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "liveness probes on pod are failing",
		Category:                       TargetEvent,
	}, EventReadinessProbeChangeDuringDisruption: {
		Type:                           corev1.EventTypeNormal,
		Reason:                         EventReadinessProbeChangeDuringDisruption,
		OnDisruptionTemplateMessage:    "readiness probe on targeted pod %s is failing",
		OnDisruptionTemplateAggMessage: "readiness probes on targeted pod(s) are failing",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "readiness probes on pod are failing",
		Category:                       TargetEvent,
	}, EventReadinessProbeChangeBeforeDisruption: {
		Type:                           corev1.EventTypeWarning,
		Reason:                         EventReadinessProbeChangeBeforeDisruption,
		OnDisruptionTemplateMessage:    "readiness probe on targeted pod %s is failing",
		OnDisruptionTemplateAggMessage: "readiness probes on targeted pod(s) are failing",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "readiness probes on pod are failing",
		Category:                       TargetEvent,
	},
	EventTooManyRestarts: {
		Type:                           corev1.EventTypeWarning,
		Reason:                         EventTooManyRestarts,
		OnDisruptionTemplateMessage:    "targeted pod %s has restarted too many times",
		OnDisruptionTemplateAggMessage: "targeted pod(s) have restarted too many times",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "pod has restarted too many times",
		Category:                       TargetEvent,
	},
	EventPodRecoveredState: {
		Type:                           corev1.EventTypeNormal,
		Reason:                         EventPodRecoveredState,
		OnDisruptionTemplateMessage:    "targeted pod %s seems to have recovered",
		OnDisruptionTemplateAggMessage: "targeted pod(s) seem to have recovered",
		OnTargetTemplateMessage:        "pod seems to have recovered from the disruption %s failure",
		Category:                       TargetEvent,
	},
	EventNodeMemPressureState: {
		Type:                           corev1.EventTypeWarning,
		Reason:                         EventNodeMemPressureState,
		OnDisruptionTemplateMessage:    "targeted node %s is under memory pressure",
		OnDisruptionTemplateAggMessage: "targeted node(s) are under memory pressure",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "node is under memory pressure",
		Category:                       TargetEvent,
	},
	EventNodeDiskPressureState: {
		Type:                           corev1.EventTypeWarning,
		Reason:                         EventNodeDiskPressureState,
		OnDisruptionTemplateMessage:    "targeted node %s is under disk pressure",
		OnDisruptionTemplateAggMessage: "targeted node(s) are under disk pressure",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "node is under disk pressure",
		Category:                       TargetEvent,
	},
	EventNodeUnavailableNetworkState: {
		Type:                           corev1.EventTypeWarning,
		Reason:                         EventNodeUnavailableNetworkState,
		OnDisruptionTemplateMessage:    "targeted node %s network is unavailable",
		OnDisruptionTemplateAggMessage: "targeted node(s) network are unavailable",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "node network is unavaialble",
		Category:                       TargetEvent,
	},
	EventNodeWarningState: {
		Type:                           corev1.EventTypeWarning,
		Reason:                         EventNodeWarningState,
		OnDisruptionTemplateMessage:    "targeted node %s is not ready",
		OnDisruptionTemplateAggMessage: "targeted node(s) are not ready",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "node is not ready",
		Category:                       TargetEvent,
	},
	EventNodeRecoveredState: {
		Type:                           corev1.EventTypeNormal,
		Reason:                         EventNodeRecoveredState,
		OnDisruptionTemplateMessage:    "targeted node %s seems to have recovered",
		OnDisruptionTemplateAggMessage: "targeted node(s) seem to have recovered",
		OnTargetTemplateMessage:        "node seems to have recovered from the disruption %s failure",
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
}

// IsNotifiableEvent this event can be broadcasted to our notifiers
func IsNotifiableEvent(event corev1.Event) bool {
	return event.Source.Component == SourceDisruptionComponent
}

func IsRecoveryEvent(event corev1.Event) bool {
	return event.Reason == EventNodeRecoveredState || event.Reason == EventPodRecoveredState
}

func IsTargetEvent(event corev1.Event) bool {
	if _, ok := Events[event.Reason]; !ok {
		return false
	}

	return event.Source.Component == SourceDisruptionComponent &&
		Events[event.Reason].Category == TargetEvent
}
