// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package container

import (
	"fmt"
	"os"
	"strconv"

	"github.com/DataDog/chaos-controller/env"
)

// Cgroup represents a cgroup manager able to join the given cgroup
type Cgroup interface {
	Join(kind string, pid int) error
	Write(kind, file, data string) error
	Exists(kind string) (bool, error)
	DiskThrottleRead(identifier, bps int) error
	DiskThrottleWrite(identifier, bps int) error
}

type cgroup struct {
	path        string
	cgroupMount string
}

func newCgroup(path string) (Cgroup, error) {
	cgroupMount, ok := os.LookupEnv(env.InjectorMountCgroup)
	if !ok {
		return nil, fmt.Errorf("environment variable %s doesn't exist", env.InjectorMountCgroup)
	}

	return cgroup{
		path:        path,
		cgroupMount: cgroupMount,
	}, nil
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

// generatePath generates a path within the cgroup like /<mount>/<kind>/<path (kubepods)>
func (c cgroup) generatePath(kind string) string {
	return fmt.Sprintf("%s%s/%s", c.cgroupMount, kind, c.path)
}

// Write writes the given data to the given cgroup kind
func (c cgroup) Write(kind, file, data string) error {
	path := fmt.Sprintf("%s/%s", c.generatePath(kind), file)

	return c.writeCgroupFile(path, data)
}

// Exists returns true if the given cgroup exists, false otherwise
func (c cgroup) Exists(kind string) (bool, error) {
	path := fmt.Sprintf("%s/cgroup.procs", c.generatePath(kind))
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// Join adds the given PID to the given cgroup
func (c cgroup) Join(kind string, pid int) error {
	path := fmt.Sprintf("%s/cgroup.procs", c.generatePath(kind))

	return c.writeCgroupFile(path, strconv.Itoa(pid))
}

// diskThrottle writes a disk throttling rule to the given blkio cgroup file
func (c cgroup) diskThrottle(path string, identifier, bps int) error {
	data := fmt.Sprintf("%d:0 %d", identifier, bps)

	return c.writeCgroupFile(path, data)
}

// DiskThrottleRead adds a disk throttle on read operations to the given disk identifier
func (c cgroup) DiskThrottleRead(identifier, bps int) error {
	path := fmt.Sprintf("%s/blkio.throttle.read_bps_device", c.generatePath("blkio"))

	return c.diskThrottle(path, identifier, bps)
}

// DiskThrottleWrite adds a disk throttle on write operations to the given disk identifier
func (c cgroup) DiskThrottleWrite(identifier, bps int) error {
	path := fmt.Sprintf("%s/blkio.throttle.write_bps_device", c.generatePath("blkio"))

	return c.diskThrottle(path, identifier, bps)
}
