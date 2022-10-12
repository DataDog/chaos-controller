// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package injector

import (
	"fmt"
	"runtime"
	"sync"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/process"
	"github.com/DataDog/chaos-controller/stress"
	"github.com/DataDog/chaos-controller/types"
)

type cpuPressureInjector struct {
	spec     v1beta1.CPUPressureSpec
	config   CPUPressureInjectorConfig
	routines int
}

// CPUPressureInjectorConfig is the CPU pressure injector config
type CPUPressureInjectorConfig struct {
	Config
	Stresser        stress.Stresser
	StresserExit    chan struct{}
	ProcessManager  process.Manager
	StresserManager StresserManager
}

// NewCPUPressureInjector creates a CPU pressure injector with the given config
func NewCPUPressureInjector(spec v1beta1.CPUPressureSpec, config CPUPressureInjectorConfig) (Injector, error) {
	// create stresser
	if config.Stresser == nil {
		config.Stresser = stress.NewCPU(config.DryRun)
	}

	if config.StresserExit == nil {
		config.StresserExit = make(chan struct{})
	}

	// create process manager
	if config.ProcessManager == nil {
		config.ProcessManager = process.NewManager(config.DryRun)
	}

	if config.StresserManager == nil {
		return nil, fmt.Errorf("StresserManager does not exist")
	}

	return &cpuPressureInjector{
		spec:   spec,
		config: config,
	}, nil
}

func (i *cpuPressureInjector) GetDisruptionKind() types.DisruptionKindName {
	return types.DisruptionKindCPUPressure
}

func (i *cpuPressureInjector) Inject() error {
	cores, err := i.config.StresserManager.TrackInjectorCores(i.config)

	if err != nil {
		return fmt.Errorf("failed to parse CPUSet %w", err)
	}

	wg := sync.WaitGroup{}
	succeeded := true
	mutex := sync.Mutex{}

	// create one stress goroutine per allocated core
	// each goroutine is locked on its current thread, without any other routines running on it
	// it allows to have a 1 routine = 1 thread pattern
	// each thread is then moved to the target cpu and cpuset cgroups so it can be schedule on the target allocated cores
	// each thread is also niced to the highest priority
	// because of linux scheduling, each thread will occupy a different core of allocated cores when stressing the cpu
	for _, core := range cores.ToSlice() {
		if i.config.StresserManager.IsCoreAlreadyStressed(core) {
			i.config.Log.Infof("core %d is already stressed, skipping", core)
			continue
		}

		wg.Add(1)

		go func(core int) {
			// lock the routine on the current thread
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()

			// retrieve current thread PID
			pid := i.config.ProcessManager.ThreadID()

			if i.joinCgroup(core, pid) && i.prioritizeStresserProcess(core, pid) {
				mutex.Lock()
				i.routines++
				mutex.Unlock()
				wg.Done()
				i.startStresser(core, pid)
			} else {
				succeeded = false
				wg.Done()
			}
		}(core)
	}

	// wait for stress routines to finish initializing
	wg.Wait()

	if !succeeded {
		return fmt.Errorf("at least one stresser routine failed to execute")
	}

	i.config.Log.Infow("all routines have been created successfully, now stressing", "stresserPIDPerCore", i.config.StresserManager.StresserPIDs())

	return nil
}

func (i *cpuPressureInjector) joinCgroup(core int, pid int) bool {
	// join target CPU cgroup
	i.config.Log.Infow("joining target CPU cgroup", "core", core, "pid", pid)

	if err := i.config.Cgroup.Join("cpu", pid, false); err != nil {
		i.config.Log.Errorw("failed join the target CPU cgroup", "error", err, "core", core, "pid", pid)
		return false
	}

	// join target cpuset cgroup in case it is used to pin the target on specific cores
	i.config.Log.Infow("joining target cpuset cgroup", "core", core, "pid", pid)

	if err := i.config.Cgroup.Join("cpuset", pid, false); err != nil {
		i.config.Log.Errorw("failed to join the target cpuset cgroup", "error", err, "core", core, "pid", pid)
		return false
	}

	return true
}

func (i *cpuPressureInjector) startStresser(core int, pid int) {
	i.config.Log.Infow("starting the stresser", "core", core, "pid", pid)
	i.config.StresserManager.TrackCoreAlreadyStressed(core, pid)
	i.config.Stresser.Stress(i.config.StresserExit)
}

func (i *cpuPressureInjector) prioritizeStresserProcess(core int, pid int) bool {
	i.config.Log.Infow("prioritizing the current stresser process", "core", core, "pid", pid)

	if err := i.config.ProcessManager.Prioritize(); err != nil {
		i.config.Log.Errorw("failed to prioritize the current stresser process", "error", err, "core", core, "pid", pid)
		return false
	}

	return true
}

func (i *cpuPressureInjector) UpdateConfig(config Config) {
	i.config.Config = config
}

func (i *cpuPressureInjector) Clean() error {
	i.config.Log.Info("killing %d routines", i.routines)

	// exit the stress routines
	for r := 0; r < i.routines; r++ {
		i.config.StresserExit <- struct{}{}
	}

	i.config.Log.Info("all routines has been killed, exiting")

	return nil
}
