// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package container

import (
	"context"
	"fmt"

	dockerlib "github.com/docker/docker/client"
)

type dockerRuntime struct {
	client *dockerlib.Client
}

func newDockerRuntime() (Runtime, error) {
	c, err := dockerlib.NewClientWithOpts(dockerlib.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to docker: %w", err)
	}

	c.NegotiateAPIVersion(context.Background())

	return &dockerRuntime{client: c}, nil
}

func (d dockerRuntime) Name(id string) (string, error) {
	ci, err := d.client.ContainerInspect(context.Background(), id)
	if err != nil {
		return "", fmt.Errorf("error while loading given container: %w", err)
	}

	return ci.ContainerJSONBase.Name, nil
}

func (d dockerRuntime) PID(id string) (uint32, error) {
	ci, err := d.client.ContainerInspect(context.Background(), id)
	if err != nil {
		return 0, fmt.Errorf("error while loading given container: %w", err)
	}

	return uint32(ci.State.Pid), nil
}

func (d dockerRuntime) HostPath(id, path string) (string, error) {
	var hostPath string

	// inspect container
	ci, err := d.client.ContainerInspect(context.Background(), id)
	if err != nil {
		return "", fmt.Errorf("error while loading given container: %w", err)
	}

	// search for the mount in mounts
	for _, mount := range ci.Mounts {
		if mount.Destination != path {
			continue
		}

		hostPath = mount.Source
	}

	// ignore if no matching path found
	if hostPath == "" {
		return "", nil
	}

	return hostPath, nil
}
