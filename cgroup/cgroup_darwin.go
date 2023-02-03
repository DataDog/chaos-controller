// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package cgroup

import (
	"fmt"
	"go.uber.org/zap"
)

type cgroup struct {
	dryRun bool
	log    *zap.SugaredLogger
}

// Read reads the given cgroup file data and returns the content as a string
func (cg cgroup) Read(controller, file string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

// Write writes the given data to the given cgroup kind
func (cg cgroup) Write(controller, file, data string) error {
	return fmt.Errorf("not implemented")
}

// Exists returns true if the given cgroup exists, false otherwise
func (cg cgroup) Exists(controller string) bool {
	return false
}

// Join adds the given PID to the given cgroup
// If inherit is set to true, all PID of the same group will be moved to the cgroup (writing to cgroup.procs file)
// Otherwise, only the given PID will be moved to the cgroup (writing to tasks file)
func (cg cgroup) Join(controller string, pid int, inherit bool) error {
	return fmt.Errorf("not implemented")
}

// DiskThrottleRead adds a disk throttle on read operations to the given disk identifier
func (cg cgroup) DiskThrottleRead(identifier, bps int) error {
	return fmt.Errorf("not implemented")
}

// DiskThrottleWrite adds a disk throttle on write operations to the given disk identifier
func (cg cgroup) DiskThrottleWrite(identifier, bps int) error {
	return fmt.Errorf("not implemented")
}

func (cg cgroup) IsCgroupV2() bool {
	return false
}

// NewManager creates a new cgroup manager from the given cgroup root path
func NewManager(dryRun bool, pid uint32, log *zap.SugaredLogger) (Manager, error) {
	return cgroup{
		dryRun: dryRun,
		log:    log,
	}, nil
}
