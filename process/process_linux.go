// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package process

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

const (
	maxPriorityValue = -20
)

type manager struct {
	dryRun bool
}

// NewManager creates a new process manager
func NewManager(dryRun bool) Manager {
	return manager{dryRun}
}

func (p manager) SetAffinity(cpus []int) error {
	affinitySet := unix.CPUSet{}
	for _, cpu := range cpus {
		affinitySet.Set(cpu)
	}
	return unix.SchedSetaffinity(0, &affinitySet)
}

// Prioritize set the priority of the current process group to the max value (-20)
func (p manager) Prioritize() error {
	pgid := unix.Getpgrp()
	if err := unix.Setpriority(unix.PRIO_PGRP, pgid, maxPriorityValue); err != nil {
		return err
	}

	return nil
}

// ThreadID returns the caller thread PID
func (p manager) ThreadID() int {
	return unix.Gettid()
}

// ProcessID returns the caller PID
func (p manager) ProcessID() int {
	return unix.Getpid()
}

// Find looks for a running process by its pid
func (p manager) Find(pid int) (*os.Process, error) {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil, fmt.Errorf("unable to find process: %w", err)
	}

	return proc, nil
}

func (p manager) Exists(pid int) bool {
	process, _ := p.Find(pid)
	err := p.Signal(process, unix.Signal(0))
	return err == nil || err == unix.EPERM
}

// Signal sends the provided signal to the given process
func (p manager) Signal(process *os.Process, signal os.Signal) error {
	// early exit if dry-run mode is enabled
	if p.dryRun {
		return nil
	}

	if err := process.Signal(signal); err != nil {
		return err
	}

	return nil
}
