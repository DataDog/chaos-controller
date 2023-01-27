// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package cgroup

import (
	"errors"
	"fmt"
	"github.com/DataDog/chaos-controller/env"
	"go.uber.org/zap"
	"os"
)

// Manager represents a cgroup manager able to join the given cgroup
type Manager interface {
	Join(kind string, pid int, inherit bool) error
	Read(kind, file string) (string, error)
	Write(kind, file, data string) error
	Exists(kind string) (bool, error)
	DiskThrottleRead(identifier, bps int) error
	DiskThrottleWrite(identifier, bps int) error
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

	if pathExists("/sys/fs/cgroup/cgroup.controllers") {
		return cgroupV2{log: log}, nil
	}

	return cgroup{
		dryRun: dryRun,
		paths:  cgroupPaths,
		mount:  mount,
		log:    log,
	}, nil
}

func pathExists(path string) bool {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return false
	}

	return true
}