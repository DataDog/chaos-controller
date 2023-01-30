// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package cgroup

import (
	"github.com/opencontainers/runc/libcontainer/cgroups"
	"github.com/opencontainers/runc/libcontainer/cgroups/fs"
	"github.com/opencontainers/runc/libcontainer/cgroups/fs2"
	"github.com/opencontainers/runc/libcontainer/configs"
	"path/filepath"
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
