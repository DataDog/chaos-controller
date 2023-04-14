// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package container

import (
	"fmt"
)

// Container describes a container
type Container interface {
	ID() string
	Runtime() Runtime
	PID() uint32
	Name() string
}

// Config contains needed interfaces
type Config struct {
	Runtime Runtime
}

type container struct {
	config Config
	id     string
	pid    uint32
	name   string
}

// New creates a new container object with default config
func New(id string) (Container, error) {
	return NewWithConfig(id, Config{})
}

// NewWithConfig creates a new container object with the given config
// nil fields are defaulted
func NewWithConfig(id string, config Config) (Container, error) {
	containerID, runtime, err := ParseContainerID(id)
	if err != nil {
		return nil, fmt.Errorf("unable to parse container ID %s: %w", id, err)
	}

	// create runtime driver
	if config.Runtime == nil {
		var err error

		switch runtime {
		case "containerd":
			config.Runtime, err = newContainerdRuntime()
		case "docker":
			config.Runtime, err = newDockerRuntime()
		default:
			return nil, fmt.Errorf("unsupported container runtime, only docker and containerd are supported")
		}

		if err != nil {
			return nil, err
		}
	}

	// retrieve pid from container info
	pid, err := config.Runtime.PID(containerID)
	if err != nil {
		return nil, fmt.Errorf("error getting PID: %w", err)
	}

	name, err := config.Runtime.Name(containerID)
	if err != nil {
		return nil, fmt.Errorf("error getting container name: %w", err)
	}

	return container{
		config: config,
		id:     containerID,
		pid:    pid,
		name:   name,
	}, nil
}

func (c container) ID() string {
	return c.id
}

func (c container) Runtime() Runtime {
	return c.config.Runtime
}

func (c container) PID() uint32 {
	return c.pid
}

func (c container) Name() string {
	return c.name
}
