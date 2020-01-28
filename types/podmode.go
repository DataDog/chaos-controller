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

	// DisruptionKindLabel is the label used to identify the disruption kind for a chaos pod
	DisruptionKindLabel = "chaos.datadoghq.com/disruption-kind"
	// DisruptionKindNetworkFailure is a network failure disruption
	DisruptionKindNetworkFailure = "network-failure"
	// DisruptionKindNetworkLatency is a network latency disruption
	DisruptionKindNetworkLatency = "network-latency"
	// DisruptionKindNodeFailure is a node failure disruption
	DisruptionKindNodeFailure = "node-failure"
)
