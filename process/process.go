// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package process

import "syscall"

const (
	maxPriorityValue = -20
)

// Manager manages the current process
type Manager interface {
	Prioritize() error
}

type manager struct{}

// NewManager creates a new process manager
func NewManager() Manager {
	return manager{}
}

// Prioritize set the priority of the current process group to the max value (-20)
func (p manager) Prioritize() error {
	pgid := syscall.Getpgrp()
	if err := syscall.Setpriority(syscall.PRIO_PGRP, pgid, maxPriorityValue); err != nil {
		return err
	}

	return nil
}
