// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package cgroup

import (
	"fmt"

	"github.com/DataDog/chaos-controller/cpuset"
	"go.uber.org/zap"
)

type cgroup struct {
	dryRun bool
	log    *zap.SugaredLogger
}

func (cg cgroup) Read(controller, file string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (cg cgroup) ReadCPUSet() (cpuset.CPUSet, error) {
	return cpuset.NewCPUSet(), fmt.Errorf("not implemented")
}

func (cg cgroup) Write(controller, file, data string) error {
	return fmt.Errorf("not implemented")
}

func (cg cgroup) Join(pid int) error {
	return fmt.Errorf("not implemented")
}

func (cg cgroup) IsCgroupV2() bool {
	return false
}

func (cg cgroup) RelativePath(controller string) string {
	return ""
}

// NewManager creates a new cgroup manager from the given cgroup root path
func NewManager(dryRun bool, pid uint32, cgroupMount string, log *zap.SugaredLogger) (Manager, error) {
	return cgroup{
		dryRun: dryRun,
		log:    log,
	}, nil
}
