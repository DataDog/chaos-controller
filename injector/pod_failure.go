// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package injector

import (
	"fmt"
	"os"
	"syscall"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/process"
)

// podFailureInjector describes a pod failure injector
type podFailureInjector struct {
	spec   v1beta1.PodFailureSpec
	config PodFailureInjectorConfig
}

// PodFailureInjectorConfig contains needed drivers to
// create a PodFailureInjector
type PodFailureInjectorConfig struct {
	Config
	ProcessManager process.Manager
}

// NewPodFailureInjector creates a PodFailureInjector object with the given config,
// missing fields being initialized with the defaults
func NewPodFailureInjector(spec v1beta1.PodFailureSpec, config PodFailureInjectorConfig) Injector {
	if config.ProcessManager == nil {
		config.ProcessManager = process.NewManager(config.DryRun)
	}

	return podFailureInjector{
		spec:   spec,
		config: config,
	}
}

// Inject sends a SIGKILL/SIGTERM signal to the container's PID
func (i podFailureInjector) Inject() error {
	var err error

	containerPid := int(i.config.Container.PID())
	proc, err := i.config.ProcessManager.Find(containerPid)

	if err != nil {
		return fmt.Errorf("error while finding the process: %w", err)
	}

	var sig os.Signal
	if i.spec.Forced {
		sig = syscall.SIGKILL
	} else {
		sig = syscall.SIGTERM
	}

	// Send signal
	i.config.Log.Infow("injecting a pod failure", "signal", sig, "container", containerPid)

	if err = i.config.ProcessManager.Signal(proc, sig); err != nil {
		return fmt.Errorf("error while sending the %s signal to container with PID %d: %w", sig, containerPid, err)
	}

	return nil
}

func (i podFailureInjector) Clean() error {
	return nil
}
