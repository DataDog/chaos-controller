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

// Join adds the given PID to all available controllers of the cgroup
func (cg cgroup) Join(pid int) error {
	return fmt.Errorf("not implemented")
}

func (cg cgroup) IsCgroupV2() bool {
	return false
}

func (cg cgroup) RelativePath() string {
	return ""
}

func (cg cgroup) Subsystems() map[string]string {
	return nil
}

// NewManager creates a new cgroup manager from the given cgroup root path
func NewManager(dryRun bool, pid uint32, cgroupMount string, log *zap.SugaredLogger) (Manager, error) {
	return cgroup{
		dryRun: dryRun,
		log:    log,
	}, nil
}
