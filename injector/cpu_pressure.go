// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package injector

import (
	"fmt"
	"runtime"
	"sync"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/process"
	"github.com/DataDog/chaos-controller/stress"
	"github.com/DataDog/chaos-controller/types"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
)

type cpuPressureInjector struct {
	spec     v1beta1.CPUPressureSpec
	config   CPUPressureInjectorConfig
	routines int
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
		config.Stresser = stress.NewCPU(config.DryRun)
	}

	if config.StresserExit == nil {
		config.StresserExit = make(chan struct{})
	}

	// create process manager
	if config.ProcessManager == nil {
		config.ProcessManager = process.NewManager(config.DryRun)
	}

	return cpuPressureInjector{
		spec:   spec,
		config: config,
	}
}

func (i cpuPressureInjector) GetDisruptionKind() types.DisruptionKindName {
	return types.DisruptionKindCPUPressure
}

func (i cpuPressureInjector) Inject() error {
	// read cpuset allocated cores
	i.config.Log.Infow("retrieving target cpuset allocated cores")

	cpusetCores, err := i.config.Cgroup.Read("cpuset", "cpuset.cpus")
	if err != nil {
		return fmt.Errorf("failed to read the target allocated cpus from the cpuset cgroup: %w", err)
	}

	// parse allocated cores
	cores, err := cpuset.Parse(cpusetCores)
	if err != nil {
		return fmt.Errorf("error parsing cpuset allocated cores: %w", err)
	}

	i.config.Log.Infow(fmt.Sprintf("target identified to be running on %d cores", cores.Size()), "cores", cores.ToSlice())

	// set new GOMAXPROCS value
	oldMaxProcs := runtime.GOMAXPROCS(cores.Size())
	i.config.Log.Infof("changed GOMAXPROCS value from %d to %d", oldMaxProcs, cores.Size())

	wg := sync.WaitGroup{}
	succeeded := true
	tids := []int{}

	// create one stress goroutine per allocated core
	// each goroutine is locked on its current thread, without any other routines running on it
	// it allows to have a 1 routine = 1 thread pattern
	// each thread is then moved to the target cpu and cpuset cgroups so it can be schedule on the target allocated cores
	// each thread is also niced to the highest priority
	// because of linux scheduling, each thread will occupy a different core of allocated cores when stressing the cpu
	for _, core := range cores.ToSlice() {
		wg.Add(1)

		go func(core int) {
			var err error

			defer func() {
				if err != nil {
					succeeded = false

					wg.Done()
				}
			}()

			// lock the routine on the current thread
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()

			// retrieve current thread PID
			pid := i.config.ProcessManager.ThreadID()

			// join target CPU cgroup
			i.config.Log.Infow("joining target CPU cgroup", "core", core, "pid", pid)

			if err = i.config.Cgroup.Join("cpu", pid, false); err != nil {
				i.config.Log.Errorw("failed join the target CPU cgroup", "error", err, "core", core, "pid", pid)

				return
			}

			// join target cpuset cgroup in case it is used to pin the target on specific cores
			i.config.Log.Infow("joining target cpuset cgroup", "core", core, "pid", pid)

			if err = i.config.Cgroup.Join("cpuset", pid, false); err != nil {
				i.config.Log.Errorw("failed to join the target cpuset cgroup", "error", err, "core", core, "pid", pid)

				return
			}

			// prioritize the current process
			i.config.Log.Infow("highering current process priority", "core", core, "pid", pid)

			if err = i.config.ProcessManager.Prioritize(); err != nil {
				i.config.Log.Errorw("error highering the current process priority", "error", err, "core", core, "pid", pid)

				return
			}

			i.config.Log.Infow("starting the stresser", "core", core, "pid", pid)

			tids = append(tids, pid)
			i.routines++

			wg.Done()
			i.config.Stresser.Stress(i.config.StresserExit)
		}(core)
	}

	// wait for stress routines to finish initializing
	wg.Wait()

	if !succeeded {
		return fmt.Errorf("at least one stresser routine failed to execute")
	}

	i.config.Log.Infow("all routines have been created successfully, now stressing", "routinesPID", tids)

	return nil
}

func (i cpuPressureInjector) Clean() error {
	i.config.Log.Info("killing routines")

	// exit the stress routines
	for r := 0; r < i.routines; r++ {
		i.config.StresserExit <- struct{}{}
	}

	i.config.Log.Info("all routines has been killed, exiting")

	return nil
}
