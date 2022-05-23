// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
)

const TargetEventOnTargetTemplate string = "Failing probably caused by disruption %s: "
const SourceDisruptionComponent string = "disruption-controller"

type DisruptionEvent struct {
	Reason                         string
	OnTargetTemplateMessage        string
	OnDisruptionTemplateMessage    string
	OnDisruptionTemplateAggMessage string
}

const (
	// Pods related warning event reasons
	DisPodWarningState       string = "PodInWarningStateChaos"
	DisContainerWarningState string = "ContainersInWarningStateChaos"
	DIsContainerChangeState  string = "ContainersStateChangeChaos"
	DisLivenessProbeChange   string = "LivenessProbeChaos"
	DisReadinessProbeChange  string = "ReadinessProbeChaos"
	DisTooManyRestarts       string = "TooManyRestartsChaos"

	DisNodeMemPressureState        string = "NodeUnderMemoryPressureChaos"
	DisNodeDiskPressureState       string = "NodeUnderDiskPressureChaos"
	DisNodeUnavailableNetworkState string = "NodeUnavailableNetworkChaos"
	DisNodeWarningState            string = "NodeInWarningStateChaos"

	DisPodRecoveredState  string = "RecoveredWarningPodStateChaos"
	DisNodeRecoveredState string = "RecoveredWarningNodeStateChaos"
)

var AllDisruptionEvents = map[string]DisruptionEvent{
	DisPodWarningState: {
		Reason:                         DisPodWarningState,
		OnDisruptionTemplateMessage:    "targeted pod %s is failing",
		OnDisruptionTemplateAggMessage: "targeted pod(s) are failing",
		OnTargetTemplateMessage:        TargetEventOnTargetTemplate + "pod is failing",
	},
	DisContainerWarningState: {
		Reason:                         DisContainerWarningState,
		OnDisruptionTemplateMessage:    "container on targeted pod %s is failing",
		OnDisruptionTemplateAggMessage: "containers on targeted pod(s) are failing",
		OnTargetTemplateMessage:        TargetEventOnTargetTemplate + "containers on pod are failing",
	},
	DisLivenessProbeChange: {
		Reason:                         DisLivenessProbeChange,
		OnDisruptionTemplateMessage:    "liveness probe on targeted pod %s are failing",
		OnDisruptionTemplateAggMessage: "liveness probe(s) on targeted pod(s) are failing",
		OnTargetTemplateMessage:        TargetEventOnTargetTemplate + "liveness probes on pod are failing",
	},
	DisReadinessProbeChange: {
		Reason:                         DisReadinessProbeChange,
		OnDisruptionTemplateMessage:    "readiness probe on targeted pod %s are failing",
		OnDisruptionTemplateAggMessage: "readiness probes on targeted pod(s) are failing",
		OnTargetTemplateMessage:        TargetEventOnTargetTemplate + "readiness probes on pod are failing",
	},
	DisTooManyRestarts: {
		Reason:                         DisTooManyRestarts,
		OnDisruptionTemplateMessage:    "targeted pod %s has restarted too many times",
		OnDisruptionTemplateAggMessage: "targeted pod(s) have restarted too many times",
		OnTargetTemplateMessage:        TargetEventOnTargetTemplate + "pod has restarted too many times",
	},
	DisNodeDiskPressureState: {
		Reason:                         DisNodeDiskPressureState,
		OnDisruptionTemplateMessage:    "targeted node %s is under disk pressure",
		OnDisruptionTemplateAggMessage: "targeted node(s) are under disk pressure",
		OnTargetTemplateMessage:        TargetEventOnTargetTemplate + "node is under disk pressure",
	},
	DisNodeMemPressureState: {
		Reason:                         DisNodeMemPressureState,
		OnDisruptionTemplateMessage:    "targeted node %s is under memory pressure",
		OnDisruptionTemplateAggMessage: "targeted node(s) are under memory pressure",
		OnTargetTemplateMessage:        TargetEventOnTargetTemplate + "node is under memory pressure",
	},
	DisNodeUnavailableNetworkState: {
		Reason:                         DisNodeUnavailableNetworkState,
		OnDisruptionTemplateMessage:    "targeted node %s network is unavailable",
		OnDisruptionTemplateAggMessage: "targeted node(s) network are unavailable",
		OnTargetTemplateMessage:        TargetEventOnTargetTemplate + "node network is unavaialble",
	},
	DisNodeWarningState: {
		Reason:                         DisNodeWarningState,
		OnDisruptionTemplateMessage:    "targeted node %s is not ready",
		OnDisruptionTemplateAggMessage: "targeted node(s) are not ready",
		OnTargetTemplateMessage:        TargetEventOnTargetTemplate + "node is not ready",
	},
	DisPodRecoveredState: {
		Reason:                         DisPodRecoveredState,
		OnDisruptionTemplateMessage:    "targeted pod %s seems to have recovered",
		OnDisruptionTemplateAggMessage: "targeted pod(s) seem to have recovered",
		OnTargetTemplateMessage:        "pod seems to have recovered from the disruption %s failure",
	},
	DisNodeRecoveredState: {
		Reason:                         DisNodeRecoveredState,
		OnDisruptionTemplateMessage:    "targeted node %s seems to have recovered",
		OnDisruptionTemplateAggMessage: "targeted node(s) seem to have recovered",
		OnTargetTemplateMessage:        "node seems to have recovered from the disruption %s failure",
	},
}

func IsDisruptionEvent(event corev1.Event, eventType string) bool {
	for _, disruptionEvent := range AllDisruptionEvents {
		if disruptionEvent.Reason == event.Reason && eventType == event.Type {
			return true
		}
	}

	return false
}
