// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package cgroup

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/DataDog/chaos-controller/env"
	"github.com/opencontainers/runc/libcontainer/cgroups"
	"github.com/opencontainers/runc/libcontainer/cgroups/fs"
	"github.com/opencontainers/runc/libcontainer/cgroups/fs2"
	"github.com/opencontainers/runc/libcontainer/configs"
	"go.uber.org/zap"
)

func parse(cgroupFile string) (map[string]string, error) {
	return cgroups.ParseCgroupFile(cgroupFile)
}

func pathExists(path string) (bool, error) {
	return cgroups.PathExists(path), nil
}

func cgroupManager(cgroupFile string) (cgroups.Manager, error) {
	cg := &configs.Cgroup{
		Resources: &configs.Resources{},
	}
	cgroupPaths, err := parse(cgroupFile)

	if err != nil {
		return nil, err
	}

	if cgroups.IsCgroup2UnifiedMode() {
		// Note that for cgroup v2 unified hierarchy, there are no per-controller
		// cgroup paths, so the resulting map will have a single element where the key
		// is empty string ("") and the value is the cgroup path the <pid> is in.
		cgroupPath := cgroupPaths[""]
		return fs2.NewManager(cg, cgroupPath)
	}

	for subsystem, path := range cgroupPaths {
		if subsystem != "" {
			// Process ID 1 is usually the init process primarily responsible for starting and shutting down the system.
			// Originally, process ID 1 was not specifically reserved for init by any technical measures:
			// it simply had this ID as a natural consequence of being the first process invoked by the kernel.
			cgroupPaths[subsystem] = filepath.Join("/proc/1/root/sys/fs/cgroup", subsystem, path)
		}
	}

	return fs.NewManager(cg, cgroupPaths)
}

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
func (m cgroup) Exists(kind string) (bool, error) {
	kindPath, err := m.generatePath(kind)
	if err != nil {
		return false, err
	}

	path := fmt.Sprintf("%s/cgroup.procs", kindPath)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// Join adds the given PID to the given cgroup
// If inherit is set to true, all PID of the same group will be moved to the cgroup (writing to cgroup.procs file)
// Otherwise, only the given PID will be moved to the cgroup (writing to tasks file)
func (m cgroup) Join(kind string, pid int, inherit bool) error {
	file := "tasks"

	if inherit {
		file = "cgroup.procs"
	}

	kindPath, err := m.generatePath(kind)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("%s/%s", kindPath, file)

	return m.write(path, strconv.Itoa(pid))
}

// diskThrottle writes a disk throttling rule to the given blkio cgroup file
func (m cgroup) diskThrottle(path string, identifier, bps int) error {
	data := fmt.Sprintf("%d:0 %d", identifier, bps)

	return m.write(path, data)
}

// DiskThrottleRead adds a disk throttle on read operations to the given disk identifier
func (m cgroup) DiskThrottleRead(identifier, bps int) error {
	kindPath, err := m.generatePath("blkio")
	if err != nil {
		return err
	}

	path := fmt.Sprintf("%s/blkio.throttle.read_bps_device", kindPath)

	return m.diskThrottle(path, identifier, bps)
}

// DiskThrottleWrite adds a disk throttle on write operations to the given disk identifier
func (m cgroup) DiskThrottleWrite(identifier, bps int) error {
	kindPath, err := m.generatePath("blkio")
	if err != nil {
		return err
	}

	path := fmt.Sprintf("%s/blkio.throttle.write_bps_device", kindPath)

	return m.diskThrottle(path, identifier, bps)
}

func (m cgroup) IsCgroupV2() bool {
	return false
}

// NewManager creates a new cgroup manager from the given cgroup root path
func NewManager(dryRun bool, pid uint32, log *zap.SugaredLogger) (Manager, error) {
	mount, ok := os.LookupEnv(env.InjectorMountCgroup)
	if !ok {
		return nil, fmt.Errorf("environment variable %s doesn't exist", env.InjectorMountCgroup)
	}

	// create cgroups manager
	cgroupPaths, err := parse(fmt.Sprintf("/proc/%d/cgroup", pid))
	if err != nil {
		return nil, err
	}

	isCgroupV2, err := pathExists("/sys/fs/cgroup/cgroup.controllers")
	if err != nil {
		return nil, err
	}

	manager, err := cgroupManager(fmt.Sprintf("/proc/%d/cgroup", pid))
	if err != nil {
		return nil, err
	}

	if isCgroupV2 {
		return cgroupV2{
			manager: &manager,
			log:     log,
		}, nil
	}

	return cgroup{
		manager: &manager,
		dryRun:  dryRun,
		paths:   cgroupPaths,
		mount:   mount,
		log:     log,
	}, nil
}
