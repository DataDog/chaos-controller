// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package cgroup

import (
	"fmt"
	"os"

	"github.com/DataDog/chaos-controller/env"
	"go.uber.org/zap"
)

type cgroup struct {
	dryRun bool
	paths  map[string]string
	mount  string
	log    *zap.SugaredLogger
}

func parse(cgroupFile string) (map[string]string, error) {
	return nil, fmt.Errorf("not implemented")
}

func pathExists(path string) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

// Read reads the given cgroup file data and returns the content as a string
func (m cgroup) Read(controller, file string) (string, error) {
	return "", nil
}

// Write writes the given data to the given cgroup kind
func (m cgroup) Write(controller, file, data string) error {
	return nil
}

// Exists returns true if the given cgroup exists, false otherwise
func (m cgroup) Exists(kind string) (bool, error) {
	return true, nil
}

// Join adds the given PID to the given cgroup
// If inherit is set to true, all PID of the same group will be moved to the cgroup (writing to cgroup.procs file)
// Otherwise, only the given PID will be moved to the cgroup (writing to tasks file)
func (m cgroup) Join(kind string, pid int, inherit bool) error {
	return nil
}

// DiskThrottleRead adds a disk throttle on read operations to the given disk identifier
func (m cgroup) DiskThrottleRead(identifier, bps int) error {
	return nil
}

// DiskThrottleWrite adds a disk throttle on write operations to the given disk identifier
func (m cgroup) DiskThrottleWrite(identifier, bps int) error {
	return nil
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

	if isCgroupV2 {
		return cgroupV2{
			log: log,
		}, nil
	}

	return cgroup{
		dryRun: dryRun,
		paths:  cgroupPaths,
		mount:  mount,
		log:    log,
	}, nil
}
