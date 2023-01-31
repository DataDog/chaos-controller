// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package cgroup

import (
	"fmt"

	"github.com/opencontainers/runc/libcontainer/cgroups"
)

func parse(cgroupFile string) (map[string]string, error) {
	return nil, fmt.Errorf("not implemented")
}

func pathExists(path string) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

func cgroupManager(cgroupFile string) (cgroups.Manager, error) {
	return nil, fmt.Errorf("not implemented")
}
