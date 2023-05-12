// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

//go:build !linux
// +build !linux

package process

import (
	"errors"
)

// Prioritize set the priority of the current process group to the max value (-20)
func (p manager) Prioritize() error {
	return errors.New("unsupported")
}

// ThreadID returns the caller thread PID
func (p manager) ThreadID() int {
	return -1
}

// ProcessID returns the caller PID
func (p manager) ProcessID() int {
	return -1
}

func (p manager) SetAffinity([]int) error {
	return errors.New("unsupported")
}
