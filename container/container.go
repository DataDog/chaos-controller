// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package container

import (
	"fmt"
	"runtime"
	"strings"
)

// Container describes a container
type Container interface {
	ID() string
	Runtime() Runtime
	Netns() Netns
	EnterNetworkNamespace() error
	ExitNetworkNamespace() error
	JoinCPUCgroup() error
}

// Config contains needed interfaces
type Config struct {
	Runtime Runtime
	Netns   Netns
	Cgroup  Cgroup
}

type container struct {
	config Config
	id     string
	rootns int
}

// New creates a new container object with default config
func New(id string) (Container, error) {
	return NewWithConfig(id, Config{})
}

// NewWithConfig creates a new container object with the given config
// nil fields are defaulted
func NewWithConfig(id string, config Config) (Container, error) {
	// parse container id
	rawID := strings.Split(id, "://")
	if len(rawID) != 2 {
		return nil, fmt.Errorf("unrecognized container ID format '%s', expecting 'containerd://<ID>' or 'docker://<ID>'", id)
	}

	// parse config
	if config.Netns == nil {
		config.Netns = &netnsDriver{}
	}

	if config.Runtime == nil {
		var err error

		switch {
		case strings.HasPrefix(id, "containerd://"):
			config.Runtime, err = newContainerdRuntime()
		case strings.HasPrefix(id, "docker://"):
			config.Runtime, err = newDockerRuntime()
		default:
			return nil, fmt.Errorf("unsupported container runtime, only docker and containerd are supported")
		}

		if err != nil {
			return nil, err
		}
	}

	if config.Cgroup == nil {
		config.Cgroup = newCgroup()
	}

	// retrieve root ns
	rootns, err := config.Netns.GetCurrent()
	if err != nil {
		return nil, fmt.Errorf("can't get root network namespace: %w", err)
	}

	return container{
		id:     rawID[1],
		rootns: rootns,
		config: config,
	}, nil
}

func (c container) ID() string {
	return c.id
}

func (c container) Runtime() Runtime {
	return c.config.Runtime
}

func (c container) Netns() Netns {
	return c.config.Netns
}

// exitNetworkNamespace returns into the root network namespace
func (c container) ExitNetworkNamespace() error {
	// re-enter into the root network namespace
	err := c.config.Netns.Set(c.rootns)
	if err != nil {
		return fmt.Errorf("error while re-entering the root network namespace: %w", err)
	}

	// unlock the goroutine so it can be moved to another thread
	// if needed
	runtime.UnlockOSThread()

	return nil
}

// enterNetworkNamespace saves the actual namespace and enters the given container network namespace
func (c container) EnterNetworkNamespace() error {
	// lock actual goroutine on the thread it is running on
	// to avoid it to be moved to another thread which would cause
	// the network namespace to change (and leak)
	runtime.LockOSThread()

	// get container pid
	pid, err := c.config.Runtime.PID(c.id)
	if err != nil {
		return fmt.Errorf("can't get container pid: %w", err)
	}

	// get container network namespace
	ns, err := c.config.Netns.GetFromPID(pid)
	if err != nil {
		return fmt.Errorf("error while retrieving the given container network namespace: %w", err)
	}

	// ensure root ns and container ns are not the same
	// if it's the case, it could leak network injections on the host running the container
	if ns == c.rootns {
		return fmt.Errorf("root network namespace seems to be the same as the container network namespace")
	}

	// enter the container network namespace
	err = c.config.Netns.Set(ns)
	if err != nil {
		return fmt.Errorf("error while entering the container network namespace: %w", err)
	}

	return nil
}

// JoinCPUCgroup attaches the current process to the container CPU cgroup
func (c container) JoinCPUCgroup() error {
	// retrieve cgroup path from container data
	path, err := c.config.Runtime.CgroupPath(c.id)
	if err != nil {
		return fmt.Errorf("error joining CPU cgroup: %w", err)
	}

	// join cpu cgroup path
	if err := c.config.Cgroup.JoinCPU(path); err != nil {
		return fmt.Errorf("error joining CPU cgroup: %w", err)
	}

	return nil
}
