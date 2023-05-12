// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package cgroup

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/DataDog/chaos-controller/cpuset"
	"go.uber.org/zap"
)

// Manager represents a cgroup manager able to join the given cgroup
type Manager interface {
	// Join adds the given PID to all available controllers of the cgroup
	Join(pid int) error
	// Read the given cgroup file data and returns the content as a string
	Read(controller, file string) (string, error)
	// ReadCPUSet returns defined CPUSet
	ReadCPUSet() (cpuset.CPUSet, error)
	// Write the given data to the given cgroup kind
	Write(controller, file, data string) error
	// IsCgroupV2 returns true if CGroups are using V2 implementation
	IsCgroupV2() bool
	// RelativePath returns the controller relative path
	RelativePath(controller string) string
}

type instCGroupManager interface {
	Path(string) string
	GetPaths() map[string]string
}

type pkgCGroupManager interface {
	PathExists(path string) bool
	EnterPid(cgroupPaths map[string]string, pid int) error
	ReadFile(dir, file string) (string, error)
	WriteFile(dir, file, data string) error
}

type allCGroupManager interface {
	instCGroupManager
	pkgCGroupManager
}

type manager struct {
	cgroups   allCGroupManager
	mountPath string
	isV2      bool
	dryRun    bool
	log       *zap.SugaredLogger
}

// NewManager creates a new cgroup manager from the given cgroup root path
func NewManager(dryRun bool, pid uint32, cgroupMount string, log *zap.SugaredLogger) (Manager, error) {
	cgroupManager, err := newAllCGroupManager(fmt.Sprintf("/proc/%d/cgroup", pid), cgroupMount, log)
	if err != nil {
		return nil, err
	}

	return manager{
		cgroupManager,
		cgroupMount,
		cgroupManager.PathExists("/sys/fs/cgroup/cgroup.controllers"),
		dryRun,
		log,
	}, nil
}

// Read reads the given cgroup file data and returns the content as a string
func (m manager) Read(controller, file string) (string, error) {
	controllerDir := m.cgroups.Path(controller)

	content, err := m.cgroups.ReadFile(controllerDir, file)
	if err != nil {
		return "", err
	}

	return strings.TrimSuffix(content, "\n"), nil
}

func (m manager) ReadCPUSet() (cpuset.CPUSet, error) {
	cpusetFile := "cpuset.effective_cpus"
	if m.IsCgroupV2() {
		cpusetFile = "cpuset.cpus.effective"
	}

	cpusetCores, err := m.Read("cpuset", cpusetFile)
	if err != nil {
		return cpuset.NewCPUSet(), fmt.Errorf("failed to read the target allocated cpus from the cpuset file '%s': %w", cpusetFile, err)
	}

	return cpuset.Parse(cpusetCores)
}

// Write writes the given data to the given cgroup kind
func (m manager) Write(controller, file, data string) error {
	controllerDir := m.cgroups.Path(controller)

	m.log.Infow("writing to cgroup file", "path", filepath.Join(controllerDir, file), "data", data)

	if m.dryRun {
		return nil
	}

	return m.cgroups.WriteFile(controllerDir, file, data)
}

// Join adds the given PID to all available controllers of the cgroup
func (m manager) Join(pid int) error {
	m.log.Infow("moving the pid to cgroup", "pid", pid)

	if m.dryRun {
		return nil
	}

	return m.cgroups.EnterPid(m.cgroups.GetPaths(), pid)
}

func (m manager) IsCgroupV2() bool {
	return m.isV2
}

// RelativePath returns the cgroup relative path (without the mount path)
func (m manager) RelativePath(controller string) string {
	return strings.TrimPrefix(m.cgroups.Path(controller), m.mountPath)
}
