// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector

import (
	"fmt"
	"os"
	"runtime"
	"syscall"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/process"
	"github.com/DataDog/chaos-controller/stress"
)

type cpuPressureInjector struct {
	spec   v1beta1.CPUPressureSpec
	config CPUPressureInjectorConfig
}

// CPUPressureInjectorConfig is the CPU pressure injector config
type CPUPressureInjectorConfig struct {
	Config
	Stresser       stress.Stresser
	StresserExit   chan struct{}
	ProcessManager process.Manager
}

// NewCPUPressureInjector creates a CPU pressure injector with the given config
func NewCPUPressureInjector(spec v1beta1.CPUPressureSpec, config CPUPressureInjectorConfig) Injector {
	// create stresser
	if config.Stresser == nil {
		config.Stresser = stress.NewCPU(config.DryRun, runtime.NumCPU())
	}

	if config.StresserExit == nil {
		config.StresserExit = make(chan struct{})
	}

	// create process manager
	if config.ProcessManager == nil {
		config.ProcessManager = process.NewManager()
	}

	return cpuPressureInjector{
		spec:   spec,
		config: config,
	}
}

func (i cpuPressureInjector) Inject() error {
	// retrieve thread group ID
	tgid, err := syscall.Getpgid(os.Getpid())
	if err != nil {
		return fmt.Errorf("error retrieving thread group ID: %w", err)
	}

	// join container CPU cgroup
	i.config.Log.Infow("joining target CPU cgroup")

	if err := i.config.Cgroup.Join("cpu", tgid); err != nil {
		return fmt.Errorf("failed to inject CPU pressure: %w", err)
	}

	// prioritize the current process
	i.config.Log.Info("highering current process priority")

	if err := i.config.ProcessManager.Prioritize(); err != nil {
		return fmt.Errorf("error highering the current process priority: %w", err)
	}

	// start eating CPU in separate goroutines
	// we start one goroutine per available CPU
	i.config.Log.Infow("initializing load generator routines", "routines", runtime.NumCPU())

	go i.config.Stresser.Stress(i.config.StresserExit)

	return nil
}

func (i cpuPressureInjector) Clean() error {
	// exit the stresser
	i.config.StresserExit <- struct{}{}

	i.config.Log.Info("all routines has been killed, exiting")

	return nil
}
