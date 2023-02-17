// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package cgroup

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/opencontainers/runc/libcontainer/cgroups"
	"github.com/opencontainers/runc/libcontainer/cgroups/fs"
	"github.com/opencontainers/runc/libcontainer/cgroups/fs2"
	"github.com/opencontainers/runc/libcontainer/configs"
	"go.uber.org/zap"
)

type cgroup struct {
	manager   *cgroups.Manager
	mountPath string
	isV2      bool
	dryRun    bool
	log       *zap.SugaredLogger
}

// NewManager creates a new cgroup manager from the given cgroup root path
func NewManager(dryRun bool, pid uint32, cgroupMount string, log *zap.SugaredLogger) (Manager, error) {
	manager, err := newCgroupManager(fmt.Sprintf("/proc/%d/cgroup", pid), cgroupMount)
	if err != nil {
		return nil, err
	}

	return cgroup{
		manager:   &manager,
		mountPath: cgroupMount,
		isV2:      cgroups.PathExists("/sys/fs/cgroup/cgroup.controllers"),
		dryRun:    dryRun,
		log:       log,
	}, nil
}

// Read reads the given cgroup file data and returns the content as a string
func (cg cgroup) Read(controller, file string) (string, error) {
	controllerDir := (*cg.manager).Path(controller)
	content, err := cgroups.ReadFile(controllerDir, file)

	if err != nil {
		return "", err
	}

	return strings.TrimSuffix(content, "\n"), nil
}

// Write writes the given data to the given cgroup kind
func (cg cgroup) Write(controller, file, data string) error {
	controllerDir := (*cg.manager).Path(controller)

	cg.log.Infow("writing to cgroup file", "path", filepath.Join(controllerDir, file), "data", data)

	if cg.dryRun {
		return nil
	}

	return cgroups.WriteFile(controllerDir, file, data)
}

// Join adds the given PID to all available controllers of the cgroup
func (cg cgroup) Join(pid int) error {
	cg.log.Infow("moving the pid to cgroup", "pid", pid)

	if cg.dryRun {
		return nil
	}

	return cgroups.EnterPid((*cg.manager).GetPaths(), pid)
}

func (cg cgroup) IsCgroupV2() bool {
	return cg.isV2
}

// RelativePath returns the cgroup relative path (without the mount path)
func (cg cgroup) RelativePath(controller string) string {
	return strings.TrimPrefix((*cg.manager).Path(controller), cg.mountPath)
}

func newCgroupManager(cgroupFile string, cgroupMount string) (cgroups.Manager, error) {
	cg := &configs.Cgroup{
		Resources: &configs.Resources{},
	}

	// parse the proc cgroup file
	cgroupPaths, err := cgroups.ParseCgroupFile(cgroupFile)
	if err != nil {
		return nil, err
	}

	// prefix the cgroup path with the mount point path
	for subsystem, path := range cgroupPaths {
		cgroupPaths[subsystem] = filepath.Join(cgroupMount, subsystem, path)
	}

	// for cgroup v2 unified hierarchy, there are no per-controller
	// cgroup paths, so the resulting map will have a single element where the key
	// is empty string ("") and the value is the cgroup path the <pid> is in.
	if cgroups.IsCgroup2UnifiedMode() {
		return fs2.NewManager(cg, cgroupPaths[""])
	}

	// cgroup v1 manager
	return fs.NewManager(cg, cgroupPaths)
}
