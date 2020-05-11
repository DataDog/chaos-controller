// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

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

func (d dockerRuntime) PID(id string) (uint32, error) {
	ci, err := d.client.ContainerInspect(context.Background(), id)
	if err != nil {
		return 0, fmt.Errorf("error while loading given container: %w", err)
	}

	return uint32(ci.State.Pid), nil
}

func (d dockerRuntime) CgroupPath(id string) (string, error) {
	ci, err := d.client.ContainerInspect(context.Background(), id)
	if err != nil {
		return "", fmt.Errorf("error while loading given container: %w", err)
	}

	return ci.HostConfig.CgroupParent, nil
}
