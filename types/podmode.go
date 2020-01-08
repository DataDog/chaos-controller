package types

// PodMode represents an enum of possible chaos pod modes
type PodMode string

const (
	// PodModeLabel is the label used to identify the pod mode
	PodModeLabel = "chaos.datadoghq.com/pod-mode"
	// PodModeInject mode
	PodModeInject = "inject"
	// PodModeClean mode
	PodModeClean = "clean"
)
