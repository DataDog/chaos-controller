// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package container

// Runtime is an interface abstracting a container runtime
// being able to return a container PID from its ID
type Runtime interface {
	PID(id string) (uint32, error)
	CgroupPath(id string) (string, error)
	HostPath(id, path string) (string, error)
	Name(id string) (string, error)
}
