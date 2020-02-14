// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package container

import (
	"context"
	"fmt"

	"github.com/docker/docker/client"
)

type dockerRuntime struct{}

func (d dockerRuntime) PID(id string) (uint32, error) {
	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return 0, fmt.Errorf("unable to connect to docker: %w", err)
	}

	cli.NegotiateAPIVersion(ctx)

	ci, err := cli.ContainerInspect(ctx, id)
	if err != nil {
		return 0, fmt.Errorf("error while loading given container: %w", err)
	}

	return uint32(ci.State.Pid), nil
}
