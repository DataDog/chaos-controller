// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package container

import (
	"context"
	"fmt"
	"strings"

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

func (d dockerRuntime) CgroupPath(id string) (string, error) {
	// inspect container
	ci, err := d.client.ContainerInspect(context.Background(), id)
	if err != nil {
		return "", fmt.Errorf("error while loading given container: %w", err)
	}

	// Assumes default cgroupfs cgroup driver, looks like: /kubepods/burstable/pod89987a43-773a-48a6-8fbd-6464c8d84d13
	path := ci.HostConfig.CgroupParent

	// systemd cgroup driver, looks like: kubepods-besteffort-pod6c35b3b9_e1cf_491e_a79e_c1e756bc34c7.slice
	if !strings.Contains(path, "/") {
		// parse cgroup parent
		parts := strings.Split(ci.HostConfig.CgroupParent, "-")
		if len(parts) != 3 {
			return "", fmt.Errorf("unexpected cgroup format: %s", path)
		}

		// path is like: kubepods.slice/kubepods-besteffort.slice/kubepods-besteffort-pod6c35b3b9_e1cf_491e_a79e_c1e756bc34c7.slice
		path = fmt.Sprintf("kubepods.slice/%s-%s.slice/%s", parts[0], parts[1], path)
	}

	return path, nil
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

	// error if no matching mount has been found
	if hostPath == "" {
		return "", fmt.Errorf("no matching mount found for path %s, the given path must be a container mount", path)
	}

	return hostPath, nil
}
