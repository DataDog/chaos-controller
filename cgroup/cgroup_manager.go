// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package cgroup

// Manager represents a cgroup manager able to join the given cgroup
type Manager interface {
	Join(kind string, pid int, inherit bool) error
	Read(controller, file string) (string, error)
	Write(controller, file, data string) error
	Exists(kind string) (bool, error)
	DiskThrottleRead(identifier, bps int) error
	DiskThrottleWrite(identifier, bps int) error
	IsCgroupV2() bool
}
