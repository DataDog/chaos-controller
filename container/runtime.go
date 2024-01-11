// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package container

import (
	"fmt"
	"strings"
)

// Runtime is an interface abstracting a container runtime
// being able to return a container PID from its ID
type Runtime interface {
	PID(id string) (uint32, error)
	HostPath(id, path string) (string, error)
}

// ParseContainerID extract from given id the containerID and runtime
func ParseContainerID(id string) (containerID string, runtime string, err error) {
	// parse container id
	rawID := strings.Split(id, "://")
	if len(rawID) != 2 {
		return "", "", fmt.Errorf("unrecognized container ID format '%s', expecting 'containerd://<ID>' or 'docker://<ID>'", id)
	}

	return rawID[1], rawID[0], nil
}
