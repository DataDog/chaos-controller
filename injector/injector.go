package injector

// Injector represents a generic failure injector
type Injector struct {
	UID string
}

type ContainerInjector struct {
	Injector
	ContainerID string
}
