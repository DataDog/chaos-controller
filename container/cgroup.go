// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package container

import (
	"fmt"
	"os"
	"strconv"
	"syscall"
)

const (
	cgroupBasePath = "/mnt/cgroup"
)

// Cgroup represents a cgroup manager able to join the given cgroup
type Cgroup interface {
	JoinCPU(path string) error
}

type cgroup struct{}

func newCgroup() Cgroup {
	return cgroup{}
}

func (c cgroup) procsFilePath(path string) string {
	return fmt.Sprintf("%s/cpu/%s/cgroup.procs", cgroupBasePath, path)
}

func (c cgroup) JoinCPU(path string) error {
	tgid, err := syscall.Getpgid(os.Getpid())
	if err != nil {
		return fmt.Errorf("error joining CPU cgroup: %w", err)
	}

	// write TGID to cgroup procs file
	file, err := os.OpenFile(c.procsFilePath(path), os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error joining CPU cgroup: %w", err)
	}

	if _, err := file.WriteString(strconv.Itoa(tgid)); err != nil {
		return fmt.Errorf("error joining CPU cgroup: %w", err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("error joining CPU cgroup: %w", err)
	}

	return nil
}
