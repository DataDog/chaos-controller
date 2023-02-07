// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package cgroup

type cgroupV2 struct {
	cg cgroup
}

// Read reads the given cgroup file data and returns the content as a string
func (cg cgroupV2) Read(controller, file string) (string, error) {
	return cg.cg.Read(controller, file)
}

// Write writes the given data to the given cgroup kind
func (cg cgroupV2) Write(controller, file, data string) error {
	return cg.cg.Write(controller, file, data)
}

// Exists returns true if the given cgroup exists, false otherwise
func (cg cgroupV2) Exists(controller string) bool {
	return cg.cg.Exists(controller)
}

func (cg cgroupV2) Join(controller string, pid int, inherit bool) error {
	return cg.cg.Join(controller, pid, inherit)
}

// DiskThrottleRead adds a disk throttle on read operations to the given disk identifier
func (cg cgroupV2) DiskThrottleRead(identifier, bps int) error {
	return cg.cg.DiskThrottleRead(identifier, bps)
}

// DiskThrottleWrite adds a disk throttle on write operations to the given disk identifier
func (cg cgroupV2) DiskThrottleWrite(identifier, bps int) error {
	return cg.cg.DiskThrottleRead(identifier, bps)
}

func (cg cgroupV2) IsCgroupV2() bool {
	return true
}
