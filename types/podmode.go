package types

// PodMode represents an enum of possible chaos pod modes
type PodMode string

// DisruptionKind represents a disruption kind
type DisruptionKind string

const (
	// PodModeLabel is the label used to identify the pod mode
	PodModeLabel = "chaos.datadoghq.com/pod-mode"
	// PodModeInject mode
	PodModeInject = "inject"
	// PodModeClean mode
	PodModeClean = "clean"

	// TargetPodLabel is the label used to identify the pod targeted by a chaos pod
	TargetPodLabel = "chaos.datadoghq.com/target-pod"

	DisruptionKindLabel          = "chaos.datadoghq.com/disruption-kind"
	DisruptionKindNetworkFailure = "network-failure"
	DisruptionKindNetworkLatency = "network-latency"
	DisruptionKindNodeFailure    = "node-failure"
)
