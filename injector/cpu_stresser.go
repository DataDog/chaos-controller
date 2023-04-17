// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.
package injector

import (
	"fmt"
	"math"
	"time"

	"github.com/DataDog/chaos-controller/process"
	"github.com/DataDog/chaos-controller/types"
)

const (
	CPUStressedCommandName = "cpu-stresser"
)

type cpuStresserInjector struct {
	config         *Config
	processManager process.Manager
	runtime        process.Runtime
	percentage     int
	exiters        chan struct{}
}

// NewCPUPressureInjector creates a CPU pressure injector with the given config
func NewCPUStresserInjector(config Config, percentage int) Injector {
	return &cpuStresserInjector{
		config:         &config,
		percentage:     percentage,
		processManager: process.NewManager(config.Disruption.DryRun),
		runtime:        process.NewRuntime(config.Disruption.DryRun),
	}
}

func (*cpuStresserInjector) GetDisruptionKind() types.DisruptionKindName {
	return types.DisruptionKindCPUPressure
}

func (c *cpuStresserInjector) UpdateConfig(config Config) {
	c.config = &config
}

func (c *cpuStresserInjector) Inject() error {
	if lenExiters := len(c.exiters); lenExiters != 0 {
		return fmt.Errorf("Injector contains %d unexited stressers, all stressers should be clean before re-injecting", lenExiters)
	}

	cpuset, err := c.config.Cgroup.ReadCPUSet()
	if err != nil {
		return fmt.Errorf("unable to read CPUSet: %w", err)
	}

	stresserPID := c.processManager.ProcessID()
	stresserCount := cpuset.Size()
	c.config.Log = c.config.Log.With("stresser_pid", stresserPID, "stresser_cpuset", cpuset)

	if err := c.config.Cgroup.Join(stresserPID); err != nil {
		return fmt.Errorf("unable to join cgroup for process '%d': %w", stresserPID, err)
	}

	if err := c.processManager.Prioritize(); err != nil {
		return fmt.Errorf("unable to prioritize process: %w", err)
	}

	oldCount := c.runtime.GOMAXPROCS(stresserCount)
	if oldCount != stresserCount {
		c.config.Log.Infof("Changed GOMAXPROCS from %d to %d", oldCount, stresserCount)
	}

	cpus := cpuset.ToSlice()
	c.exiters = make(chan struct{}, len(cpus))
	for _, cpu := range cpus {
		c.stress(cpu)
	}

	return nil
}

func (c *cpuStresserInjector) Clean() error {
	c.config.Log.Info("Stopping all cpu stressers", "stressers_count", len(c.exiters))

	for len(c.exiters) < cap(c.exiters) {
		c.exiters <- struct{}{}
	}
	close(c.exiters)
	c.exiters = nil

	c.config.Log.Info("All CPU stressers are now stopped")

	return nil
}

// stress run a cpu intensive operation on cpu until an exit signal is received
func (c *cpuStresserInjector) stress(cpu int) {
	logger := c.config.Log.With("cpu", cpu, "percentage", c.percentage)

	go func() {
		if c.config.Disruption.DryRun {
			logger.Debug("Stresser dry run mode activated, skipping stress, just waiting...")

			<-c.exiters

			return
		}

		c.runtime.LockOSThread()
		defer c.runtime.UnlockOSThread()

		logger.Debug("Stresser locked on OSThread")

		if err := c.processManager.SetAffinity([]int{cpu}); err != nil {
			logger.Warnw("unable to set affinity to a specific cpu, thread might move to another CPU", "err", err)
		}

		totalDuration := 100 * time.Millisecond
		cpuPressureOnDuration := totalDuration * time.Duration(c.percentage/100)
		cpuPressureOffDuration := totalDuration - cpuPressureOnDuration

		logger.Infow("Stresser is starting", "stress_duration", cpuPressureOnDuration, "pause_duration", cpuPressureOffDuration)

	foreverLoop:
		for {
			stressUntilOff := time.After(cpuPressureOnDuration)

		stressLoop:
			for i := uint64(0); i < math.MaxUint64; i++ {
				select {
				case <-c.exiters:
					break foreverLoop
				case <-stressUntilOff:
					<-time.After(cpuPressureOffDuration)
					break stressLoop
				default:
				}
			}
		}

		logger.Info("Stresser is now stopped")
	}()
}
