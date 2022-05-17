package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
)

const TARGET_POD_EVENT_TEMPLATE string = "Failing probably caused by disruption %s: "
const DISRUPTION_EVENT_TEMPLATE string = "%s"

type DisruptionEvent struct {
	Reason                         string
	OnTargetTemplateMessage        string
	OnDisruptionTemplateMessage    string
	OnDisruptionTemplateAggMessage string
}

const (
	DIS_POD_WARNING_STATE       string = "PodInWarningStateChaos"
	DIS_CONTAINER_WARNING_STATE string = "ContainersInWarningStateChaos"
	DIS_LIVENESS_PROBE_CHANGE   string = "LivenessProbeChaos"
	DIS_READINESS_PROBE_CHANGE  string = "ReadinessProbeChaos"
	DIS_PROBE_CHANGE            string = "ProbeChaos"
	DIS_TOO_MANY_RESTARTS       string = "TooManyRestartsChaos"
	DIS_RECOVERED_STATE         string = "RecoveredWarningStateChaos"
)

var ALL_DISRUPTION_EVENTS map[string]DisruptionEvent = map[string]DisruptionEvent{
	DIS_POD_WARNING_STATE: {
		Reason:                         DIS_POD_WARNING_STATE,
		OnDisruptionTemplateMessage:    "targetted pod %s is failing",
		OnDisruptionTemplateAggMessage: "targetted pod %s is failing %d times in the last minutes",
		OnTargetTemplateMessage:        TARGET_POD_EVENT_TEMPLATE + "pod is failing",
	},
	DIS_CONTAINER_WARNING_STATE: {
		Reason:                         DIS_CONTAINER_WARNING_STATE,
		OnDisruptionTemplateMessage:    "containers on targetted pod %s are failing",
		OnDisruptionTemplateAggMessage: "containers on targetted pod %s are failing %d times in the last minutes",
		OnTargetTemplateMessage:        TARGET_POD_EVENT_TEMPLATE + "containers on pod are failing",
	},
	DIS_LIVENESS_PROBE_CHANGE: {
		Reason:                         DIS_LIVENESS_PROBE_CHANGE,
		OnDisruptionTemplateMessage:    "liveness probes on targetted pod %s are failing",
		OnDisruptionTemplateAggMessage: "liveness probes on targetted pod %s are failing %d times in the last minutes",
		OnTargetTemplateMessage:        TARGET_POD_EVENT_TEMPLATE + "liveness probes on pod are failing",
	},
	DIS_READINESS_PROBE_CHANGE: {
		Reason:                         DIS_READINESS_PROBE_CHANGE,
		OnDisruptionTemplateMessage:    "readiness probes on targetted pod %s are failing",
		OnDisruptionTemplateAggMessage: "readiness probes on targetted pod %s are failing %d times in the last minutes",
		OnTargetTemplateMessage:        TARGET_POD_EVENT_TEMPLATE + "readiness probes on pod are failing",
	},
	DIS_PROBE_CHANGE: {
		Reason:                         DIS_PROBE_CHANGE,
		OnDisruptionTemplateMessage:    "probes are failing on targetted pod %s",
		OnDisruptionTemplateAggMessage: "probes are failing on targetted pod %s for %d times in the last minutes",
		OnTargetTemplateMessage:        TARGET_POD_EVENT_TEMPLATE + "probes are failing on pod",
	},
	DIS_TOO_MANY_RESTARTS: {
		Reason:                         DIS_TOO_MANY_RESTARTS,
		OnDisruptionTemplateMessage:    "targeted pod %s has restarted too many times",
		OnDisruptionTemplateAggMessage: "targeted pod %s has restarted too many times",
		OnTargetTemplateMessage:        TARGET_POD_EVENT_TEMPLATE + "pod has restarted too many times",
	},
	DIS_RECOVERED_STATE: {
		Reason:                      DIS_RECOVERED_STATE,
		OnDisruptionTemplateMessage: "targeted pod %s seems to have recovered",
		OnTargetTemplateMessage:     TARGET_POD_EVENT_TEMPLATE + "pod seems to have recovered",
	},
}

//
func IsDisruptionEvent(event corev1.Event, eventType string) bool {
	for _, disruptionEvent := range ALL_DISRUPTION_EVENTS {
		if disruptionEvent.Reason == event.Reason && eventType == event.Type {
			return true
		}
	}

	return false
}
