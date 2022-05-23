package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
)

const TARGET_EVENT_ON_TARGET_TEMPLATE string = "Failing probably caused by disruption %s: "

type DisruptionEvent struct {
	Reason                         string
	OnTargetTemplateMessage        string
	OnDisruptionTemplateMessage    string
	OnDisruptionTemplateAggMessage string
}

const (
	// Pods related warning event reasons
	DIS_POD_WARNING_STATE       string = "PodInWarningStateChaos"
	DIS_CONTAINER_WARNING_STATE string = "ContainersInWarningStateChaos"
	DIS_CONTAINER_CHANGE_STATE  string = "ContainersStateChangeChaos"
	DIS_LIVENESS_PROBE_CHANGE   string = "LivenessProbeChaos"
	DIS_READINESS_PROBE_CHANGE  string = "ReadinessProbeChaos"
	DIS_PROBE_CHANGE            string = "ProbeChaos"
	DIS_TOO_MANY_RESTARTS       string = "TooManyRestartsChaos"

	DIS_NODE_MEM_PRESSURE_STATE        string = "NodeUnderMemoryPressureChaos"
	DIS_NODE_DISK_PRESSURE_STATE       string = "NodeUnderDiskPressureChaos"
	DIS_NODE_UNAVAILABLE_NETWORK_STATE string = "NodeUnavailableNetworkChaos"
	DIS_NODE_WARNING_STATE             string = "NodeInWarningStateChaos"

	DIS_POD_RECOVERED_STATE  string = "RecoveredWarningPodStateChaos"
	DIS_NODE_RECOVERED_STATE string = "RecoveredWarningNodeStateChaos"
)

var ALL_DISRUPTION_EVENTS map[string]DisruptionEvent = map[string]DisruptionEvent{
	DIS_POD_WARNING_STATE: {
		Reason:                         DIS_POD_WARNING_STATE,
		OnDisruptionTemplateMessage:    "targeted pod %s is failing",
		OnDisruptionTemplateAggMessage: "targeted pod(s) are failing",
		OnTargetTemplateMessage:        TARGET_EVENT_ON_TARGET_TEMPLATE + "pod is failing",
	},
	DIS_CONTAINER_WARNING_STATE: {
		Reason:                         DIS_CONTAINER_WARNING_STATE,
		OnDisruptionTemplateMessage:    "container on targeted pod %s is failing",
		OnDisruptionTemplateAggMessage: "containers on targeted pod(s) are failing",
		OnTargetTemplateMessage:        TARGET_EVENT_ON_TARGET_TEMPLATE + "containers on pod are failing",
	},
	DIS_LIVENESS_PROBE_CHANGE: {
		Reason:                         DIS_LIVENESS_PROBE_CHANGE,
		OnDisruptionTemplateMessage:    "liveness probe on targetted pod %s are failing",
		OnDisruptionTemplateAggMessage: "liveness probe(s) on targeted pod(s) are failing",
		OnTargetTemplateMessage:        TARGET_EVENT_ON_TARGET_TEMPLATE + "liveness probes on pod are failing",
	},
	DIS_READINESS_PROBE_CHANGE: {
		Reason:                         DIS_READINESS_PROBE_CHANGE,
		OnDisruptionTemplateMessage:    "readiness probe on targeted pod %s are failing",
		OnDisruptionTemplateAggMessage: "readiness probes on targeted pod(s) are failing",
		OnTargetTemplateMessage:        TARGET_EVENT_ON_TARGET_TEMPLATE + "readiness probes on pod are failing",
	},
	DIS_PROBE_CHANGE: {
		Reason:                         DIS_PROBE_CHANGE,
		OnDisruptionTemplateMessage:    "probes are failing on targetted pod %s",
		OnDisruptionTemplateAggMessage: "probes are failing on targetted pod %s for %d times in the last minutes",
		OnTargetTemplateMessage:        TARGET_EVENT_ON_TARGET_TEMPLATE + "probes are failing on pod",
	},
	DIS_TOO_MANY_RESTARTS: {
		Reason:                         DIS_TOO_MANY_RESTARTS,
		OnDisruptionTemplateMessage:    "targeted pod %s has restarted too many times",
		OnDisruptionTemplateAggMessage: "targeted pod(s) have restarted too many times",
		OnTargetTemplateMessage:        TARGET_EVENT_ON_TARGET_TEMPLATE + "pod has restarted too many times",
	},
	DIS_NODE_DISK_PRESSURE_STATE: {
		Reason:                         DIS_NODE_DISK_PRESSURE_STATE,
		OnDisruptionTemplateMessage:    "targeted node %s is under disk pressure",
		OnDisruptionTemplateAggMessage: "targeted node(s) are under disk pressure",
		OnTargetTemplateMessage:        TARGET_EVENT_ON_TARGET_TEMPLATE + "node is under disk pressure",
	},
	DIS_NODE_MEM_PRESSURE_STATE: {
		Reason:                         DIS_NODE_MEM_PRESSURE_STATE,
		OnDisruptionTemplateMessage:    "targeted node %s is under memory pressure",
		OnDisruptionTemplateAggMessage: "targeted node(s) are under memory pressure",
		OnTargetTemplateMessage:        TARGET_EVENT_ON_TARGET_TEMPLATE + "node is under memory pressure",
	},
	DIS_NODE_UNAVAILABLE_NETWORK_STATE: {
		Reason:                         DIS_NODE_UNAVAILABLE_NETWORK_STATE,
		OnDisruptionTemplateMessage:    "targeted node %s network is unavailable",
		OnDisruptionTemplateAggMessage: "targeted node(s) network are unavailable",
		OnTargetTemplateMessage:        TARGET_EVENT_ON_TARGET_TEMPLATE + "node network is unavaialble",
	},
	DIS_NODE_WARNING_STATE: {
		Reason:                         DIS_NODE_WARNING_STATE,
		OnDisruptionTemplateMessage:    "targeted node %s is not ready",
		OnDisruptionTemplateAggMessage: "targeted node(s) are not ready",
		OnTargetTemplateMessage:        TARGET_EVENT_ON_TARGET_TEMPLATE + "node is not ready",
	},
	DIS_POD_RECOVERED_STATE: {
		Reason:                         DIS_POD_RECOVERED_STATE,
		OnDisruptionTemplateMessage:    "targeted pod %s seems to have recovered",
		OnDisruptionTemplateAggMessage: "targeted pod(s) seem to have recovered",
		OnTargetTemplateMessage:        "pod seems to have recovered from the disruption %s failure",
	},
	DIS_NODE_RECOVERED_STATE: {
		Reason:                         DIS_NODE_RECOVERED_STATE,
		OnDisruptionTemplateMessage:    "targeted node %s seems to have recovered",
		OnDisruptionTemplateAggMessage: "targeted node(s) seem to have recovered",
		OnTargetTemplateMessage:        "node seems to have recovered from the disruption %s failure",
	},
}

func IsDisruptionEvent(event corev1.Event, eventType string) bool {
	for _, disruptionEvent := range ALL_DISRUPTION_EVENTS {
		if disruptionEvent.Reason == event.Reason && eventType == event.Type {
			return true
		}
	}

	return false
}
