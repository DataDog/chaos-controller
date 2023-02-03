// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package cgroup

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/opencontainers/runc/libcontainer/cgroups"
	"github.com/opencontainers/runc/libcontainer/cgroups/fs"
	"github.com/opencontainers/runc/libcontainer/cgroups/fs2"
	"github.com/opencontainers/runc/libcontainer/configs"
	"go.uber.org/zap"
)

func cgroupManager(cgroupFile string, cgroupMount string) (cgroups.Manager, error) {
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

	fmt.Println(cgroupPaths)

	// for cgroup v2 unified hierarchy, there are no per-controller
	// cgroup paths, so the resulting map will have a single element where the key
	// is empty string ("") and the value is the cgroup path the <pid> is in.
	if cgroups.IsCgroup2UnifiedMode() {
		return fs2.NewManager(cg, cgroupPaths[""])
	}

	// cgroup v1 manager
	return fs.NewManager(cg, cgroupPaths)
}

type cgroup struct {
	manager *cgroups.Manager
	dryRun  bool
	log     *zap.SugaredLogger
}

// Read reads the given cgroup file data and returns the content as a string
func (cg cgroup) Read(controller, file string) (string, error) {
	manager := *cg.manager
	controllerDir := manager.Path(controller)
	content, err := cgroups.ReadFile(controllerDir, file)

	if err != nil {
		return "", err
	}

	return strings.TrimSuffix(content, "\n"), nil
}

// Write writes the given data to the given cgroup kind
func (cg cgroup) Write(controller, file, data string) error {
	manager := *cg.manager
	controllerDir := manager.Path(controller)

	return cgroups.WriteFile(controllerDir, file, data)
}

// Exists returns true if the given cgroup exists, false otherwise
func (cg cgroup) Exists(controller string) bool {
	manager := *cg.manager
	controllerDir := manager.Path(controller)

	return cgroups.PathExists(fmt.Sprintf("%s/cgroup.procs", controllerDir))
}

// Join adds the given PID to the given cgroup
// If inherit is set to true, all PID of the same group will be moved to the cgroup (writing to cgroup.procs file)
// Otherwise, only the given PID will be moved to the cgroup (writing to tasks file)
func (cg cgroup) Join(controller string, pid int, inherit bool) error {
	file := "tasks"

	if inherit {
		file = "cgroup.procs"
	}

	manager := *cg.manager
	controllerDir := manager.Path(controller)

	return cgroups.WriteFile(controllerDir, file, strconv.Itoa(pid))
}

// DiskThrottleRead adds a disk throttle on read operations to the given disk identifier
func (cg cgroup) DiskThrottleRead(identifier, bps int) error {
	manager := *cg.manager
	controllerDir := manager.Path("blkio")
	file := "blkio.throttle.read_bps_device"
	data := fmt.Sprintf("%d:0 %d", identifier, bps)

	return cgroups.WriteFile(controllerDir, file, data)
}

// DiskThrottleWrite adds a disk throttle on write operations to the given disk identifier
func (cg cgroup) DiskThrottleWrite(identifier, bps int) error {
	manager := *cg.manager
	controllerDir := manager.Path("blkio")
	file := "blkio.throttle.write_bps_device"
	data := fmt.Sprintf("%d:0 %d", identifier, bps)

	return cgroups.WriteFile(controllerDir, file, data)
}

func (cg cgroup) IsCgroupV2() bool {
	return false
}

// NewManager creates a new cgroup manager from the given cgroup root path
func NewManager(dryRun bool, pid uint32, cgroupMount string, log *zap.SugaredLogger) (Manager, error) {
	manager, err := cgroupManager(fmt.Sprintf("/proc/%d/cgroup", pid), cgroupMount)
	if err != nil {
		return nil, err
	}

	isCgroupV2 := cgroups.PathExists("/sys/fs/cgroup/cgroup.controllers")
	if isCgroupV2 {
		return cgroupV2{
			manager: &manager,
			log:     log,
		}, nil
	}

	return cgroup{
		manager: &manager,
		dryRun:  dryRun,
		log:     log,
	}, nil
}
