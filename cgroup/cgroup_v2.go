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
	// TODO: implement missing features for cgroupV2
	// https://docs.google.com/document/d/1kNKA5k-oLA5rZDk72gyZvBbP_c5uUgLH1xXb3HtAo7s/edit#heading=h.2tvwjppghz7
	log    *zap.SugaredLogger
}

// Read reads the given cgroup file data and returns the content as a string
func (m cgroupV2) Read(kind, file string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

// Write writes the given data to the given cgroup kind
func (m cgroupV2) Write(kind, file, data string) error {
	return fmt.Errorf("not implemented")
}

// Exists returns true if the given cgroup exists, false otherwise
func (m cgroupV2) Exists(kind string) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

func (m cgroupV2) Join(kind string, pid int, inherit bool) error {
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
