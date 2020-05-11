// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package container

import (
	"context"
	"fmt"

	containerdlib "github.com/containerd/containerd"
)

type containerdRuntime struct {
	client *containerdlib.Client
}

func newContainerdRuntime() (Runtime, error) {
	c, err := containerdlib.New("/run/containerd/containerd.sock", containerdlib.WithDefaultNamespace("k8s.io"))
	if err != nil {
		return nil, fmt.Errorf("unable to connect to the containerd socket: %w", err)
	}

	return &containerdRuntime{client: c}, nil
}

func (c containerdRuntime) PID(id string) (uint32, error) {
	// load container structure
	container, err := c.client.LoadContainer(context.Background(), id)
	if err != nil {
		return 0, fmt.Errorf("error while loading the given container: %w", err)
	}

	// retrieve container task (process)
	task, err := container.Task(context.Background(), nil)
	if err != nil {
		return 0, fmt.Errorf("error while loading the given container task: %w", err)
	}

	return task.Pid(), nil
}

func (c containerdRuntime) CgroupPath(id string) (string, error) {
	// load container structure
	container, err := c.client.LoadContainer(context.Background(), id)
	if err != nil {
		return "", fmt.Errorf("error while loading the given container: %w", err)
	}

	// retrieve container spec
	spec, err := container.Spec(context.Background())
	if err != nil {
		return "", fmt.Errorf("error retrieving the given container spec: %w", err)
	}

	return spec.Linux.CgroupsPath, nil
}
