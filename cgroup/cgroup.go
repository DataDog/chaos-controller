// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package cgroup

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/opencontainers/runc/libcontainer/cgroups"

	"go.uber.org/zap"
)

type cgroup struct {
	manager *cgroups.Manager
	dryRun  bool
	paths   map[string]string
	mount   string
	log     *zap.SugaredLogger
}

// write appends the given data to the given cgroup file path
// NOTE: depending on the cgroup file, the append will result in an overwrite
func (m cgroup) write(path, data string) error {
	m.log.Infow("writing to cgroup file", "path", path, "data", data)
	// early exit if dry-run mode is enabled
	if m.dryRun {
		return nil
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error opening cgroup file %s: %w", path, err)
	}

	if _, err := file.WriteString(data); err != nil {
		return fmt.Errorf("error writing to cgroup file %s: %w", path, err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("error closing cgroup file %s: %w", path, err)
	}

	return nil
}

// generatePath generates a path within the cgroup like /<mount>/<kind>/<path (kubepods)>
func (m cgroup) generatePath(kind string) (string, error) {
	kindPath, found := m.paths[kind]
	if !found {
		return "", fmt.Errorf("cgroup path not found for kind %s", kind)
	}

	generatedPath := fmt.Sprintf("%s%s/%s", m.mount, kind, kindPath)

	return generatedPath, nil
}

// Read reads the given cgroup file data and returns the content as a string
func (m cgroup) Read(controller, file string) (string, error) {
	manager := *m.manager
	controllerDir := manager.Path(controller)
	content, err := cgroups.ReadFile(controllerDir, file)

	if err != nil {
		return "", err
	}

	return strings.TrimSuffix(content, "\n"), nil
}

// Write writes the given data to the given cgroup kind
func (m cgroup) Write(controller, file, data string) error {
	manager := *m.manager
	controllerDir := manager.Path(controller)

	return cgroups.WriteFile(controllerDir, file, data)
}

// Exists returns true if the given cgroup exists, false otherwise
func (m cgroup) Exists(controller string) bool {
	manager := *m.manager
	controllerDir := manager.Path(controller)

	return cgroups.PathExists(fmt.Sprintf("%s/cgroup.procs", controllerDir))
}

// Join adds the given PID to the given cgroup
// If inherit is set to true, all PID of the same group will be moved to the cgroup (writing to cgroup.procs file)
// Otherwise, only the given PID will be moved to the cgroup (writing to tasks file)
func (m cgroup) Join(controller string, pid int, inherit bool) error {
	file := "tasks"

	if inherit {
		file = "cgroup.procs"
	}

	manager := *m.manager
	controllerDir := manager.Path(controller)

	return cgroups.WriteFile(controllerDir, file, strconv.Itoa(pid))
}

// DiskThrottleRead adds a disk throttle on read operations to the given disk identifier
func (m cgroup) DiskThrottleRead(identifier, bps int) error {
	manager := *m.manager
	controllerDir := manager.Path("blkio")
	file := "blkio.throttle.read_bps_device"
	data := fmt.Sprintf("%d:0 %d", identifier, bps)

	return cgroups.WriteFile(controllerDir, file, data)
}

// DiskThrottleWrite adds a disk throttle on write operations to the given disk identifier
func (m cgroup) DiskThrottleWrite(identifier, bps int) error {
	manager := *m.manager
	controllerDir := manager.Path("blkio")
	file := "blkio.throttle.write_bps_device"
	data := fmt.Sprintf("%d:0 %d", identifier, bps)

	return cgroups.WriteFile(controllerDir, file, data)
}

func (m cgroup) IsCgroupV2() bool {
	return false
}
