// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package container

import (
	"fmt"
	"runtime"

	"github.com/DataDog/chaos-fi-controller/containerruntime"
	"github.com/vishvananda/netns"
)

var rootNetworkNamespace netns.NsHandle

// Container describes a container
type Container struct {
	ID               string
	PID              uint32
	NetworkNamespace netns.NsHandle
}

// New creates a new container from the given container ID, retrieving it's main PID and network namespace
func New(runtimeContainerID string) (Container, error) {
	// retreive containerID based on the container runtime
	cr, err := containerruntime.New(runtimeContainerID)
	if err != nil {
		return Container{}, fmt.Errorf("unsupported container runtime. docker or containerd are supported")
	}
	pid, err := cr.GetPID()
	if err != nil {
		return Container{}, fmt.Errorf("error while retrieving container PID: %w", err)
	}

	// retrieve container network namespace
	ns, err := getNetworkNamespace(pid)
	if err != nil {
		return Container{}, fmt.Errorf("error while retrieving the container network namespace: %w", err)
	}

	c := Container{
		ID:               cr.GetContainerID(),
		PID:              pid,
		NetworkNamespace: ns,
	}

	return c, nil
}

// getNetworkNamespace gets the given container network namespace file from its task PID
func getNetworkNamespace(pid uint32) (netns.NsHandle, error) {
	// retrieve container network namespace file
	ns, err := netns.GetFromPath(fmt.Sprintf("/mnt/proc/%d/ns/net", pid))
	if err != nil {
		return 0, fmt.Errorf("error while retrieving the given container network namespace: %w", err)
	}

	return ns, nil
}

// ExitNetworkNamespace returns into the root network namespace
func ExitNetworkNamespace() error {
	// re-enter into the root network namespace
	err := netns.Set(rootNetworkNamespace)
	if err != nil {
		return fmt.Errorf("error while re-entering the root network namespace: %w", err)
	}

	// unlock the goroutine so it can be moved to another thread
	// if needed
	runtime.UnlockOSThread()

	return nil
}

// EnterNetworkNamespace saves the actual namespace and enters the given container network namespace
func (c Container) EnterNetworkNamespace() error {
	// lock actual goroutine on the thread it is running on
	// to avoid it to be moved to another thread which would cause
	// the network namespace to change (and leak)
	runtime.LockOSThread()

	// get the current (root) network namespace to re-enter it later
	var err error
	rootNetworkNamespace, err = netns.Get()
	if err != nil {
		return fmt.Errorf("error while saving the root network namespace: %w", err)
	}

	// enter the container network namespace
	err = netns.Set(c.NetworkNamespace)
	if err != nil {
		return fmt.Errorf("error while entering the container network namespace: %w", err)
	}

	return nil
}
