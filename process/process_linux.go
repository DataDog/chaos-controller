// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

//go:build linux
// +build linux

package process

import (
	"fmt"
	"os"
	"syscall"
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

// Prioritize set the priority of the current process group to the max value (-20)
func (p manager) Prioritize() error {
	pgid := syscall.Getpgrp()
	if err := syscall.Setpriority(syscall.PRIO_PGRP, pgid, maxPriorityValue); err != nil {
		return err
	}

	return nil
}

// ThreadID returns the caller thread PID
func (p manager) ThreadID() int {
	return syscall.Gettid()
}

// Find looks for a running process by its pid
func (p manager) Find(pid int) (*os.Process, error) {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil, fmt.Errorf("unable to find process: %w", err)
	}

	return proc, nil
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
