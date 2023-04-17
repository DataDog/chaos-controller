// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package cgroup

import "github.com/DataDog/chaos-controller/cpuset"

// Manager represents a cgroup manager able to join the given cgroup
type Manager interface {
	Join(pid int) error
	Read(controller, file string) (string, error)
	ReadCPUSet() (cpuset.CPUSet, error)
	Write(controller, file, data string) error
	IsCgroupV2() bool
	RelativePath(controller string) string
}
