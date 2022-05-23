// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
)

const EventOnTargetTemplate string = "Failing probably caused by disruption %s: "
const SourceDisruptionComponent string = "disruption-controller"

type DisruptionEvent struct {
	Type                           string
	Reason                         string
	OnTargetTemplateMessage        string
	OnDisruptionTemplateMessage    string // We want to separate the aggregated message from the single message to include more info in the single message
	OnDisruptionTemplateAggMessage string
}

const (
	// Targeted pods related
	// Warning events
	EventPodWarningState       string = "TargetPodInWarningState"
	EventContainerWarningState string = "TargetPodContainersInWarningState"
	EventLivenessProbeChange   string = "TargetPodLivenessProbe"
	EventReadinessProbeChange  string = "TargetPodReadinessProbe"
	EventTooManyRestarts       string = "TargetPodTooManyRestarts"
	// Normal events
	EventPodRecoveredState string = "RecoveredWarningStateInTargetPod"

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
	EventDisruptionNoTarget             string = "NoTarget"
	EventInvalidSpecDisruption          string = "InvalidSpec"
	// Normal events
	EventDisruptionCreated      string = "Created"
	EventDisruptionDurationOver string = "DurationOver"
	EventDisrupted              string = "Disrupted"
)

var Events = map[string]DisruptionEvent{
	EventPodWarningState: {
		Type:                           corev1.EventTypeWarning,
		Reason:                         EventPodWarningState,
		OnDisruptionTemplateMessage:    "targeted pod %s is failing",
		OnDisruptionTemplateAggMessage: "targeted pod(s) are failing",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "pod is failing",
	},
	EventContainerWarningState: {
		Type:                           corev1.EventTypeWarning,
		Reason:                         EventContainerWarningState,
		OnDisruptionTemplateMessage:    "container on targeted pod %s is failing",
		OnDisruptionTemplateAggMessage: "containers on targeted pod(s) are failing",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "containers on pod are failing",
	},
	EventLivenessProbeChange: {
		Type:                           corev1.EventTypeWarning,
		Reason:                         EventLivenessProbeChange,
		OnDisruptionTemplateMessage:    "liveness probe on targeted pod %s are failing",
		OnDisruptionTemplateAggMessage: "liveness probe(s) on targeted pod(s) are failing",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "liveness probes on pod are failing",
	},
	EventReadinessProbeChange: {
		Type:                           corev1.EventTypeWarning,
		Reason:                         EventReadinessProbeChange,
		OnDisruptionTemplateMessage:    "readiness probe on targeted pod %s are failing",
		OnDisruptionTemplateAggMessage: "readiness probes on targeted pod(s) are failing",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "readiness probes on pod are failing",
	},
	EventTooManyRestarts: {
		Type:                           corev1.EventTypeWarning,
		Reason:                         EventTooManyRestarts,
		OnDisruptionTemplateMessage:    "targeted pod %s has restarted too many times",
		OnDisruptionTemplateAggMessage: "targeted pod(s) have restarted too many times",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "pod has restarted too many times",
	},
	EventPodRecoveredState: {
		Type:                           corev1.EventTypeNormal,
		Reason:                         EventPodRecoveredState,
		OnDisruptionTemplateMessage:    "targeted pod %s seems to have recovered",
		OnDisruptionTemplateAggMessage: "targeted pod(s) seem to have recovered",
		OnTargetTemplateMessage:        "pod seems to have recovered from the disruption %s failure",
	},
	EventNodeMemPressureState: {
		Type:                           corev1.EventTypeWarning,
		Reason:                         EventNodeMemPressureState,
		OnDisruptionTemplateMessage:    "targeted node %s is under memory pressure",
		OnDisruptionTemplateAggMessage: "targeted node(s) are under memory pressure",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "node is under memory pressure",
	},
	EventNodeDiskPressureState: {
		Type:                           corev1.EventTypeWarning,
		Reason:                         EventNodeDiskPressureState,
		OnDisruptionTemplateMessage:    "targeted node %s is under disk pressure",
		OnDisruptionTemplateAggMessage: "targeted node(s) are under disk pressure",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "node is under disk pressure",
	},
	EventNodeUnavailableNetworkState: {
		Type:                           corev1.EventTypeWarning,
		Reason:                         EventNodeUnavailableNetworkState,
		OnDisruptionTemplateMessage:    "targeted node %s network is unavailable",
		OnDisruptionTemplateAggMessage: "targeted node(s) network are unavailable",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "node network is unavaialble",
	},
	EventNodeWarningState: {
		Type:                           corev1.EventTypeWarning,
		Reason:                         EventNodeWarningState,
		OnDisruptionTemplateMessage:    "targeted node %s is not ready",
		OnDisruptionTemplateAggMessage: "targeted node(s) are not ready",
		OnTargetTemplateMessage:        EventOnTargetTemplate + "node is not ready",
	},
	EventNodeRecoveredState: {
		Type:                           corev1.EventTypeNormal,
		Reason:                         EventNodeRecoveredState,
		OnDisruptionTemplateMessage:    "targeted node %s seems to have recovered",
		OnDisruptionTemplateAggMessage: "targeted node(s) seem to have recovered",
		OnTargetTemplateMessage:        "node seems to have recovered from the disruption %s failure",
	},
	EventDisruptionDurationOver: {
		Type:                        corev1.EventTypeWarning,
		Reason:                      EventDisruptionDurationOver,
		OnDisruptionTemplateMessage: "The disruption has lived %s longer than its specified duration, and will now be deleted.",
	},
	EventEmptyDisruption: {
		Type:                        corev1.EventTypeWarning,
		Reason:                      EventEmptyDisruption,
		OnDisruptionTemplateMessage: "No disruption recognized for \"%s\" therefore no disruption applied.",
	},
	EventDisruptionCreationFailed: {
		Type:                        corev1.EventTypeWarning,
		Reason:                      EventDisruptionCreationFailed,
		OnDisruptionTemplateMessage: "Injection pod for disruption \"%s\" failed to be created",
	},
	EventDisruptionStuckOnRemoval: {
		Type:                        corev1.EventTypeWarning,
		Reason:                      EventDisruptionStuckOnRemoval,
		OnDisruptionTemplateMessage: "Instance is stuck on removal because of chaos pods not being able to terminate correctly, please check pods logs before manually removing their finalizer. https://github.com/DataDog/chaos-controller/blob/main/docs/faq.md",
	},
	EventInvalidDisruptionLabelSelector: {
		Type:                        corev1.EventTypeWarning,
		Reason:                      EventInvalidDisruptionLabelSelector,
		OnDisruptionTemplateMessage: "%s. No targets will be selected.",
	},
	EventDisruptionNoTarget: {
		Type:                        corev1.EventTypeWarning,
		Reason:                      EventDisruptionNoTarget,
		OnDisruptionTemplateMessage: "No more targets found for injection for this disruption (either ignored or already targeted by another disruption)",
	},
	EventInvalidSpecDisruption: {
		Type:                        corev1.EventTypeWarning,
		Reason:                      EventInvalidSpecDisruption,
		OnDisruptionTemplateMessage: "%s",
	},
	EventDisruptionCreated: {
		Type:                        corev1.EventTypeNormal,
		Reason:                      EventDisruptionCreated,
		OnDisruptionTemplateMessage: "Created disruption injection pod for \"%s\"",
	},
	EventDisrupted: {
		Type:                    corev1.EventTypeWarning,
		Reason:                  EventDisrupted,
		OnTargetTemplateMessage: "Pod %s from disruption %s targeted this resource for injection",
	},
}

func IsDisruptionEvent(event corev1.Event, eventType string) bool {
	for _, disruptionEvent := range Events {
		if disruptionEvent.Reason == event.Reason && eventType == event.Type {
			return true
		}
	}

	return false
}
