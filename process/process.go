// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package process

import "os"

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
