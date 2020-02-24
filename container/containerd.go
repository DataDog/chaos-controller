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

type containerdRuntime struct{}

func (c containerdRuntime) PID(id string) (uint32, error) {
	// get containerd instance
	containerdClient, err := containerdlib.New("/run/containerd/containerd.sock", containerdlib.WithDefaultNamespace("k8s.io"))
	if err != nil {
		return 0, fmt.Errorf("unable to connect to the containerd socket: %w", err)
	}

	// load container structure
	container, err := containerdClient.LoadContainer(context.Background(), id)
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
