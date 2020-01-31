package injector

import "go.uber.org/zap"

// Injector represents a generic failure injector
type Injector struct {
	Log *zap.SugaredLogger
	UID string
}

// ContainerInjector represents an injector for containers
type ContainerInjector struct {
	Injector
	ContainerID string
}
