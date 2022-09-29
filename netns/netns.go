// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package netns

import (
	"fmt"
	"os"
	"runtime"

	"github.com/DataDog/chaos-controller/env"
	"github.com/vishvananda/netns"
	"go.uber.org/zap"
)

// Manager is an interface being able to interact with a process network namespace
type Manager interface {
	Enter() error
	Exit() error
}

type manager struct {
	log      *zap.SugaredLogger
	rootns   int
	targetns int
}

// NewManager creates a new network namespace manager for the given PID
func NewManager(log *zap.SugaredLogger, pid uint32) (Manager, error) {
	// retrieve proc mount point
	mountProc, ok := os.LookupEnv(env.InjectorMountProc)
	if !ok {
		return nil, fmt.Errorf("environment variable %s doesn't exist", env.InjectorMountProc)
	}

	// retrieve current (root) network namespace
	rootns, err := netns.Get()
	if err != nil {
		return nil, fmt.Errorf("error getting current (root) network namespace: %w", err)
	}

	// retrieve network namespace inode
	targetns, err := netns.GetFromPath(fmt.Sprintf("%s%d/ns/net", mountProc, pid))
	if err != nil {
		return nil, fmt.Errorf("error getting given PID (%d) network namespace: %w", pid, err)
	}

	log.Debugw("Retrieved root namespace and target namespace", "rootns", int(rootns), "targetns", int(targetns), "targetnsPath", fmt.Sprintf("%s%d/ns/net", mountProc, pid))

	// build manager
	mgr := manager{
		log:      log,
		rootns:   int(rootns),
		targetns: int(targetns),
	}

	return mgr, nil
}

func (m manager) Enter() error {
	runtime.LockOSThread()

	m.log.Debugf("Entering network namespace of %d", m.targetns)

	return netns.Set(netns.NsHandle(m.targetns))
}

func (m manager) Exit() error {
	runtime.UnlockOSThread()

	m.log.Debugf("Leaving network namespace of %d", m.targetns)

	return netns.Set(netns.NsHandle(m.rootns))
}
