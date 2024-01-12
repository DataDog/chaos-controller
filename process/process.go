// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package process

import (
	"os"

	"golang.org/x/sys/unix"
)

const (
	NotFoundProcessPID = -1
)

// Manager manages a process
type Manager interface {
	Prioritize() error
	ThreadID() int
	ProcessID() int
	Exists(pid int) (bool, error)
	Find(pid int) (*os.Process, error)
	Signal(process *os.Process, signal os.Signal) error
	SetAffinity([]int) error
}

type manager struct {
	dryRun bool
}

// NewManager creates a new process manager
func NewManager(dryRun bool) Manager {
	return manager{
		dryRun: dryRun,
	}
}

// Find looks for a running process by its pid
func (p manager) Find(pid int) (*os.Process, error) {
	// unix based system never returns an error on find
	proc, _ := os.FindProcess(pid)
	return proc, nil
}

func (p manager) Exists(pid int) (bool, error) {
	// unix based system never returns an error on find
	process, _ := p.Find(pid)

	err := p.Signal(process, unix.Signal(0))
	if err != nil && err != unix.EPERM {
		return false, err
	}

	return true, nil
}

// Signal sends the provided signal to the given process
func (p manager) Signal(process *os.Process, signal os.Signal) error {
	// early exit if dry-run mode is enabled
	if p.dryRun {
		return nil
	}

	if process.Pid == 0 || process.Pid == NotFoundProcessPID {
		return nil
	}

	return process.Signal(signal)
}
