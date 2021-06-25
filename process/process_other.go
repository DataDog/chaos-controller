// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

// +build !linux

package process

import "errors"

type manager struct{}

// NewManager creates a new process manager
func NewManager() Manager {
	return manager{}
}

// Prioritize set the priority of the current process group to the max value (-20)
func (p manager) Prioritize() error {
	return errors.New("unsupported")
}

// ThreadID returns the caller thread PID
func (p manager) ThreadID() int {
	return -1
}
