// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package cgroup

import (
	"fmt"

	"go.uber.org/zap"
)

type cgroupV2 struct {
	log *zap.SugaredLogger
}

// Read reads the given cgroup file data and returns the content as a string
func (m cgroupV2) Read(controller, file string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

// Write writes the given data to the given cgroup kind
func (m cgroupV2) Write(controller, file, data string) error {
	return fmt.Errorf("not implemented")
}

// Exists returns true if the given cgroup exists, false otherwise
func (m cgroupV2) Exists(controller string) bool {
	return false
}

func (m cgroupV2) Join(controller string, pid int, inherit bool) error {
	return fmt.Errorf("not implemented")
}

// DiskThrottleRead adds a disk throttle on read operations to the given disk identifier
func (m cgroupV2) DiskThrottleRead(identifier, bps int) error {
	return fmt.Errorf("not implemented")
}

// DiskThrottleWrite adds a disk throttle on write operations to the given disk identifier
func (m cgroupV2) DiskThrottleWrite(identifier, bps int) error {
	return fmt.Errorf("not implemented")
}

func (m cgroupV2) IsCgroupV2() bool {
	return true
}
