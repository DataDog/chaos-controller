package v1beta1

const (
	DIS_CONTAINER_STATE_WAIT_CHANGE      string = "ContainerStateWaitingChaosBroadcaster"
	DIS_CONTAINER_STATE_TERMINATE_CHANGE string = "ContainerStateTerminateChaosBroadcaster"
	DIS_READINESS_PROBE_CHANGE           string = "ReadinessProbeChaosBroadcaster"
	DIS_LIVENESS_PROBE_CHANGE            string = "LivenessProbeChaosBroadcaster"
	DIS_TOO_MANY_RESTARTS                string = "TooManyRestartsChaosBroadcaster"
)

var ALL_EVENT_TYPES []string = []string{
	DIS_CONTAINER_STATE_WAIT_CHANGE,
	DIS_CONTAINER_STATE_TERMINATE_CHANGE,
	DIS_READINESS_PROBE_CHANGE,
	DIS_LIVENESS_PROBE_CHANGE,
	DIS_TOO_MANY_RESTARTS,
}
