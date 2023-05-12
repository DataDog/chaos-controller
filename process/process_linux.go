// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package process

import (
	"golang.org/x/sys/unix"
)

const (
	maxPriorityValue = -20
)

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
