// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package container

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"syscall"

	"github.com/DataDog/chaos-controller/env"
	ps "github.com/mitchellh/go-ps"
)

// Cgroup represents a cgroup manager able to join the given cgroup
type Cgroup interface {
	Create(kind, name string) error
	Remove(kind, name string) error
	Join(kind, name string, pid int) error
	Write(kind, name, file, data string) error
	Empty(kind, from, to string) error
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

// readCgroupFile reads data from the given path and returns a slice containing lines
func (c cgroup) readCgroupFile(path string) ([]string, error) {
	var lines []string

	file, err := os.OpenFile(path, os.O_RDONLY, 0444)
	if err != nil {
		return nil, fmt.Errorf("error opening cgroup file %s: %w", path, err)
	}

	reader := bufio.NewScanner(file)
	for reader.Scan() {
		lines = append(lines, reader.Text())
	}

	if err := file.Close(); err != nil {
		return nil, fmt.Errorf("error closing cgroup file %s: %w", path, err)
	}

	return lines, nil
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

// generatePath generates a path within the cgroup like /<mount>/<kind>/<path (kubepods)>/<name>
func (c cgroup) generatePath(kind, name string) string {
	return fmt.Sprintf("%s%s/%s/%s", c.cgroupMount, kind, c.path, name)
}

// Write writes the given data to the given cgroup kind and name
func (c cgroup) Write(kind, name, file, data string) error {
	path := fmt.Sprintf("%s/%s", c.generatePath(kind, name), file)

	return c.writeCgroupFile(path, data)
}

// Create creates a new cgroup of the given kind and with the given name
func (c cgroup) Create(kind, name string) error {
	path := c.generatePath(kind, name)

	// create cgroup folder
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			//nolint:gosec
			return os.Mkdir(path, 0755)
		}

		return fmt.Errorf("error creating cgroup: %w", err)
	}

	return nil
}

// Empty moves every process from the given cgroup to the given cgroup in order to fully empty it
func (c cgroup) Empty(kind, from, to string) error {
	fromPath := fmt.Sprintf("%s/cgroup.procs", c.generatePath(kind, from))
	toPath := fmt.Sprintf("%s/cgroup.procs", c.generatePath(kind, to))

	// read procs in from
	procs, err := c.readCgroupFile(fromPath)
	if err != nil {
		return fmt.Errorf("error reading procs from %s: %w", fromPath, err)
	}

	// write procs in to
	for _, proc := range procs {
		if err := c.writeCgroupFile(toPath, proc); err != nil {
			// ensure the process still exists before throwing an error because
			// writing a non-existing PID in a cgroup.procs would throw an error we don't care about
			pid, err := strconv.Atoi(proc)
			if err != nil {
				return fmt.Errorf("error writing proc %s to %s: unexpected PID format", proc, toPath)
			}

			// skip if the process does not exist anymore
			if psProcess, err := ps.FindProcess(pid); psProcess == nil && err == nil {
				continue
			}

			return fmt.Errorf("error writing proc %s to %s: %w", proc, toPath, err)
		}
	}

	return nil
}

// Remove removes the given cgroup
func (c cgroup) Remove(kind, name string) error {
	return syscall.Rmdir(c.generatePath(kind, name))
}

// Join adds the given PID to the given cgroup
func (c cgroup) Join(kind, name string, pid int) error {
	path := fmt.Sprintf("%s/cgroup.procs", c.generatePath(kind, name))

	return c.writeCgroupFile(path, strconv.Itoa(pid))
}

// diskThrottle writes a disk throttling rule to the given blkio cgroup file
func (c cgroup) diskThrottle(path string, identifier, bps int) error {
	data := fmt.Sprintf("%d:0 %d", identifier, bps)

	return c.writeCgroupFile(path, data)
}

// DiskThrottleRead adds a disk throttle on read operations to the given disk identifier
func (c cgroup) DiskThrottleRead(identifier, bps int) error {
	path := fmt.Sprintf("%s/blkio.throttle.read_bps_device", c.generatePath("blkio", ""))

	return c.diskThrottle(path, identifier, bps)
}

// DiskThrottleWrite adds a disk throttle on write operations to the given disk identifier
func (c cgroup) DiskThrottleWrite(identifier, bps int) error {
	path := fmt.Sprintf("%s/blkio.throttle.write_bps_device", c.generatePath("blkio", ""))

	return c.diskThrottle(path, identifier, bps)
}
