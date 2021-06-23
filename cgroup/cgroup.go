// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package cgroup

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/DataDog/chaos-controller/env"
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

type manager struct {
	dryRun bool
	path   string
	mount  string
}

// NewManager creates a new cgroup manager from the given cgroup root path
func NewManager(dryRun bool, path string) (Manager, error) {
	mount, ok := os.LookupEnv(env.InjectorMountCgroup)
	if !ok {
		return nil, fmt.Errorf("environment variable %s doesn't exist", env.InjectorMountCgroup)
	}

	return manager{
		dryRun: dryRun,
		path:   path,
		mount:  mount,
	}, nil
}

// read reads the given cgroup file data and returns it as a string
func (m manager) read(path string) (string, error) {
	//nolint:gosec
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("error reading cgroup file %s: %w", path, err)
	}

	return strings.TrimSuffix(string(data), "\n"), nil
}

// write appends the given data to the given cgroup file path
// NOTE: depending on the cgroup file, the append will result in an overwrite
func (m manager) write(path, data string) error {
	// early exit if dry-run mode is enabled
	if m.dryRun {
		return nil
	}

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
func (m manager) generatePath(kind string) string {
	return fmt.Sprintf("%s%s/%s", m.mount, kind, m.path)
}

// Write writes the given data to the given cgroup kind
func (m manager) Read(kind, file string) (string, error) {
	path := fmt.Sprintf("%s/%s", m.generatePath(kind), file)

	return m.read(path)
}

// Write writes the given data to the given cgroup kind
func (m manager) Write(kind, file, data string) error {
	path := fmt.Sprintf("%s/%s", m.generatePath(kind), file)

	return m.write(path, data)
}

// Exists returns true if the given cgroup exists, false otherwise
func (m manager) Exists(kind string) (bool, error) {
	path := fmt.Sprintf("%s/cgroup.procs", m.generatePath(kind))
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// Join adds the given PID to the given cgroup
// If inherit is set to true, all PID of the same group will be moved to the cgroup (writing to cgroup.procs file)
// Otherwise, only the given PID will be moved to the cgroup (writing to tasks file)
func (m manager) Join(kind string, pid int, inherit bool) error {
	file := "tasks"

	if inherit {
		file = "cgroup.procs"
	}

	path := fmt.Sprintf("%s/%s", m.generatePath(kind), file)

	return m.write(path, strconv.Itoa(pid))
}

// diskThrottle writes a disk throttling rule to the given blkio cgroup file
func (m manager) diskThrottle(path string, identifier, bps int) error {
	data := fmt.Sprintf("%d:0 %d", identifier, bps)

	return m.write(path, data)
}

// DiskThrottleRead adds a disk throttle on read operations to the given disk identifier
func (m manager) DiskThrottleRead(identifier, bps int) error {
	path := fmt.Sprintf("%s/blkio.throttle.read_bps_device", m.generatePath("blkio"))

	return m.diskThrottle(path, identifier, bps)
}

// DiskThrottleWrite adds a disk throttle on write operations to the given disk identifier
func (m manager) DiskThrottleWrite(identifier, bps int) error {
	path := fmt.Sprintf("%s/blkio.throttle.write_bps_device", m.generatePath("blkio"))

	return m.diskThrottle(path, identifier, bps)
}
