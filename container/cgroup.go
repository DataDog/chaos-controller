// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package container

import (
	"fmt"
	"os"
	"strconv"
	"syscall"
)

const (
	cgroupBasePath = "/mnt/cgroup"
)

// Cgroup represents a cgroup manager able to join the given cgroup
type Cgroup interface {
	JoinCPU() error
	DiskThrottleRead(identifier, bps int) error
	DiskThrottleWrite(identifier, bps int) error
}

type cgroup struct {
	path string
}

func newCgroup(path string) Cgroup {
	return cgroup{
		path: path,
	}
}

// writeCgroupFile appends the given data to the given cgroup file path
// NOTE: depending on the cgroup file, the append will result in an overwrite
func (c cgroup) writeCgroupFile(path, data string) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error opening cgroup file %s: %w", path, err)
	}

	if _, err := file.WriteString(data); err != nil {
		return fmt.Errorf("error writing to cgroup file %s: %w", path, err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("error closing cgroup file %s: %w", path, err)
	}

	return nil
}

// JoinCPU adds the current thread group to the instance CPU cgroup
func (c cgroup) JoinCPU() error {
	tgid, err := syscall.Getpgid(os.Getpid())
	if err != nil {
		return fmt.Errorf("error joining CPU cgroup: %w", err)
	}

	// write TGID to cgroup procs file
	path := fmt.Sprintf("%s/cpu/%s/cgroup.procs", cgroupBasePath, c.path)

	return c.writeCgroupFile(path, strconv.Itoa(tgid))
}

// diskThrottle writes a disk throttling rule to the given blkio cgroup file
func (c cgroup) diskThrottle(path string, identifier, bps int) error {
	data := fmt.Sprintf("%d:0 %d", identifier, bps)

	return c.writeCgroupFile(path, data)
}

// DiskThrottleRead adds a disk throttle on read operations to the given disk identifier
func (c cgroup) DiskThrottleRead(identifier, bps int) error {
	path := fmt.Sprintf("%s/blkio/%s/blkio.throttle.read_bps_device", cgroupBasePath, c.path)

	return c.diskThrottle(path, identifier, bps)
}

// DiskThrottleWrite adds a disk throttle on write operations to the given disk identifier
func (c cgroup) DiskThrottleWrite(identifier, bps int) error {
	path := fmt.Sprintf("%s/blkio/%s/blkio.throttle.write_bps_device", cgroupBasePath, c.path)

	return c.diskThrottle(path, identifier, bps)
}
