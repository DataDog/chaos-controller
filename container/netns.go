// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package container

import (
	"fmt"
	"os"

	"github.com/DataDog/chaos-controller/env"
	"github.com/vishvananda/netns"
)

// Netns is an interface being able to interact with a process
// network namespace
type Netns interface {
	Set(ns int) error
	GetCurrent() (int, error)
	GetFromPID(pid uint32) (int, error)
}

type netnsDriver struct{}

func (d netnsDriver) Set(ns int) error {
	return netns.Set(netns.NsHandle(ns))
}

func (d netnsDriver) GetCurrent() (int, error) {
	ns, err := netns.Get()
	return int(ns), err
}

func (d netnsDriver) GetFromPID(pid uint32) (int, error) {
	mountProc, ok := os.LookupEnv(env.InjectorMountProc)
	if !ok {
		return -1, fmt.Errorf("environment variable %s doesn't exist", env.InjectorMountProc)
	}

	ns, err := netns.GetFromPath(fmt.Sprintf("%s%d/ns/net", mountProc, pid))
	return int(ns), err
}
