// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package cgroup

import "github.com/DataDog/chaos-controller/cpuset"

// Manager represents a cgroup manager able to join the given cgroup
type Manager interface {
	// Join adds the given PID to all available controllers of the cgroup
	Join(pid int) error
	// Read the given cgroup file data and returns the content as a string
	Read(controller, file string) (string, error)
	// ReadCPUSet returns defined CPUSet
	ReadCPUSet() (cpuset.CPUSet, error)
	// Write the given data to the given cgroup kind
	Write(controller, file, data string) error
	// IsCgroupV2 returns true if CGroups are using V2 implementation
	IsCgroupV2() bool
	// RelativePath returns the controller relative path
	RelativePath(controller string) string
}
