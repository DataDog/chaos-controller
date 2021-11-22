// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

//go:build !linux
// +build !linux

package process

import (
	"errors"
	"os"
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
	return errors.New("unsupported")
}

// ThreadID returns the caller thread PID
func (p manager) ThreadID() int {
	return -1
}

// Find looks for a running process by its pid
func (p manager) Find(pid int) (*os.Process, error) {
	return nil, errors.New("unsupported")
}

// Signal sends the provided signal to the given process
func (p manager) Signal(process *os.Process, signal os.Signal) error {
	return errors.New("unsupported")
}
