// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package cgroup

import (
	"path/filepath"
	"strings"

	"github.com/opencontainers/runc/libcontainer/cgroups"
	"github.com/opencontainers/runc/libcontainer/cgroups/fs"
	"github.com/opencontainers/runc/libcontainer/cgroups/fs2"
	"github.com/opencontainers/runc/libcontainer/configs"
	"go.uber.org/zap"

	"github.com/DataDog/chaos-controller/o11y/tags"
)

type instManager struct {
	instCGroupManager
	pkgCGroupManager
}

type pkgManager struct{}

func newAllCGroupManager(cgroupFile string, cgroupMount string, log *zap.SugaredLogger) (allCGroupManager, error) {
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
		// Skip named hierarchies (e.g. "name=systemd"): they are not resource controllers
		// and their mount directory name differs from the subsystem key, making the path wrong.
		if strings.HasPrefix(subsystem, "name=") {
			delete(cgroupPaths, subsystem)
			continue
		}

		log.Debugw("adding cgroup subsystem path to manager", tags.SubsystemKey, subsystem, tags.PathKey, path)
		cgroupPaths[subsystem] = filepath.Join(cgroupMount, subsystem, path)
	}

	// for cgroup v2 unified hierarchy, there are no per-controller
	// cgroup paths, so the resulting map will have a single element where the key
	// is empty string ("") and the value is the cgroup path the <pid> is in.
	var externalCGroupManager cgroups.Manager
	if cgroups.IsCgroup2UnifiedMode() {
		externalCGroupManager, err = fs2.NewManager(cg, cgroupPaths[""])
	} else {
		// We don't want the empty subsystem if we're in v1
		delete(cgroupPaths, "")

		// cgroup v1 manager
		externalCGroupManager, err = fs.NewManager(cg, cgroupPaths)
	}

	if err != nil {
		return nil, err
	}

	return instManager{
		instCGroupManager: externalCGroupManager,
		pkgCGroupManager:  pkgManager{},
	}, nil
}

func (pkgManager) PathExists(path string) bool {
	return cgroups.PathExists(path)
}

func (pkgManager) EnterPid(cgroupPaths map[string]string, pid int) error {
	for _, path := range cgroupPaths {
		if err := cgroups.WriteCgroupProc(path, pid); err != nil {
			return err
		}
	}

	return nil
}

func (pkgManager) ReadFile(dir, file string) (string, error) {
	return cgroups.ReadFile(dir, file)
}

func (pkgManager) WriteFile(dir, file, data string) error {
	return cgroups.WriteFile(dir, file, data)
}
