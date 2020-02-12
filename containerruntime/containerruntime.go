// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package containerruntime

import (
	"context"
	"fmt"
	"strings"

	"github.com/containerd/containerd"
	"github.com/docker/docker/client"
)

// ContainerRuntime handles
type ContainerRuntime interface {
	GetPID() (uint32, error)
	GetContainerID() string
}

type containerRuntime struct {
	containerID string
}

type containerdContainerRuntime struct {
	containerRuntime
}
type dockerContainerRuntime struct {
	containerRuntime
}

// New creates new runtime specific struct to retreive PID
func New(containerID string) (ContainerRuntime, error) {
	cID := strings.Split(containerID, "://")
	if len(cID) != 2 {
		return nil, fmt.Errorf("unrecognized container ID format '%s', expecting 'containerd://<ID>' or 'docker://<ID>'", containerID)
	}

	switch {
	case strings.HasPrefix(containerID, "containerd://"):
		return containerdContainerRuntime{containerRuntime{containerID: cID[1]}}, nil
	case strings.HasPrefix(containerID, "docker://"):
		return dockerContainerRuntime{containerRuntime{containerID: cID[1]}}, nil
	default:
		return nil, fmt.Errorf("unsupported container runtime. docker or containerd are supported")
	}
}

func (c containerdContainerRuntime) GetPID() (uint32, error) {
	// get containerd instance
	containerdClient, err := containerd.New("/run/containerd/containerd.sock", containerd.WithDefaultNamespace("k8s.io"))
	if err != nil {
		return 0, fmt.Errorf("unable to connect to the containerd socket: %w", err)
	}

	// load container structure
	container, err := containerdClient.LoadContainer(context.Background(), c.containerID)
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

func (c containerdContainerRuntime) GetContainerID() string {
	return c.containerID
}

func (d dockerContainerRuntime) GetPID() (uint32, error) {
	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return 0, fmt.Errorf("unable to connect to docker: %w", err)
	}

	cli.NegotiateAPIVersion(ctx)

	ci, err := cli.ContainerInspect(ctx, d.containerID)
	if err != nil {
		return 0, fmt.Errorf("error while loading given container: %w", err)
	}

	return uint32(ci.State.Pid), nil
}

func (d dockerContainerRuntime) GetContainerID() string {
	return d.containerID
}
