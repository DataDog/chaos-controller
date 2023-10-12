// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package injector

import (
	"fmt"
	"os"
	"syscall"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/process"
	"github.com/DataDog/chaos-controller/types"
)

// containerFailureInjector describes a container failure injector
type containerFailureInjector struct {
	spec   v1beta1.ContainerFailureSpec
	config ContainerFailureInjectorConfig
}

// ContainerFailureInjectorConfig contains needed drivers to
// create a ContainerFailureInjector
type ContainerFailureInjectorConfig struct {
	Config
	ProcessManager process.Manager
}

// NewContainerFailureInjector creates a ContainerFailureInjector object with the given config,
// missing fields being initialized with the defaults
func NewContainerFailureInjector(spec v1beta1.ContainerFailureSpec, config ContainerFailureInjectorConfig) Injector {
	if config.ProcessManager == nil {
		config.ProcessManager = process.NewManager(config.Disruption.DryRun)
	}

	return &containerFailureInjector{
		spec:   spec,
		config: config,
	}
}

func (i *containerFailureInjector) TargetName() string {
	return i.config.TargetName()
}

func (i *containerFailureInjector) GetDisruptionKind() types.DisruptionKindName {
	return types.DisruptionKindContainerFailure
}

// Inject sends a SIGKILL/SIGTERM signal to the container's PID
func (i *containerFailureInjector) Inject() error {
	containerPid := int(i.config.TargetContainer.PID())

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
	i.config.Log.Infow("injecting a container failure", "signal", sig, "container", containerPid)

	if err := i.config.ProcessManager.Signal(proc, sig); err != nil {
		return fmt.Errorf("error while sending the %s signal to container with PID %d: %w", sig, containerPid, err)
	}

	return nil
}

func (i *containerFailureInjector) UpdateConfig(config Config) {
	i.config.Config = config
}

func (i *containerFailureInjector) Clean() error {
	return nil
}
