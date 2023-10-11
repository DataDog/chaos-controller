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

type cpuStressInjector struct {
	config        *Config
	process       process.Manager
	runtime       process.Runtime
	percentage    int
	exiters       chan struct{}
	exitCompleted chan struct{}
}

// NewCPUPressureInjector creates a CPU pressure injector with the given config
func NewCPUStressInjector(config Config, percentage int, process process.Manager, runtime process.Runtime) Injector {
	return &cpuStressInjector{
		config:     &config,
		percentage: percentage,
		process:    process,
		runtime:    runtime,
	}
}

func (c *cpuStressInjector) TargetName() string {
	return c.config.TargetName()
}

func (*cpuStressInjector) GetDisruptionKind() types.DisruptionKindName {
	return types.DisruptionKindCPUStress
}

func (c *cpuStressInjector) UpdateConfig(config Config) {
	c.config = &config
}

func (c *cpuStressInjector) Inject() error {
	if capExiters := cap(c.exiters); capExiters != 0 {
		return fmt.Errorf("Injector contains %d unexited stresses, all stresses should be clean before re-injecting", capExiters)
	}

	cpuset, err := c.config.Cgroup.ReadCPUSet()
	if err != nil {
		return fmt.Errorf("unable to read CPUSet: %w", err)
	}

	stressPID := c.process.ProcessID()
	stressCount := cpuset.Size()
	c.config.Log = c.config.Log.With("stress_pid", stressPID, "stress_cpuset", cpuset)

	if err := c.config.Cgroup.Join(stressPID); err != nil {
		return fmt.Errorf("unable to join cgroup for process '%d': %w", stressPID, err)
	}

	if err := c.process.Prioritize(); err != nil {
		return fmt.Errorf("unable to prioritize process: %w", err)
	}

	oldCount := c.runtime.GOMAXPROCS(stressCount)
	if oldCount != stressCount {
		c.config.Log.Infof("Changed GOMAXPROCS from %d to %d", oldCount, stressCount)
	}

	cpus := cpuset.ToSlice()
	c.exiters = make(chan struct{}, len(cpus))

	for _, cpu := range cpus {
		c.stress(cpu)
	}

	return nil
}

func (c *cpuStressInjector) Clean() error {
	stressCount := cap(c.exiters)
	c.config.Log.Infow("Stopping all cpu stresses", "stress_count", stressCount)

	// we want to know when the stress are really completed
	// stress will signal themselves through this channel
	c.exitCompleted = make(chan struct{}, stressCount)

	// we send a message to each stress
	for i := 0; i < stressCount; i++ {
		c.exiters <- struct{}{}
	}

	// we wait for a message from each stress (a buffered channel guarantee senders blocks until receiver is available)
	for i := 0; i < stressCount; i++ {
		<-c.exitCompleted
	}

	close(c.exiters)
	c.exiters = nil

	close(c.exitCompleted)
	c.exitCompleted = nil

	c.config.Log.Info("All CPU stresses are now stopped")

	return nil
}

// stress run a cpu intensive operation on cpu until an exit signal is received
func (c *cpuStressInjector) stress(cpu int) {
	logger := c.config.Log.With("cpu", cpu, "percentage", c.percentage)

	stressConfigurationCompleted := make(chan struct{}, 1)

	go func() {
		logger := logger.With("thread_id", c.process.ThreadID())

		defer func() {
			logger.Infow("Stress is stopping...")

			c.exitCompleted <- struct{}{}

			logger.Infow("Signal sent stress is now fully stopped")
		}()

		if c.config.Disruption.DryRun {
			logger.Debug("stress dry run mode activated, skipping stress, just waiting...")

			stressConfigurationCompleted <- struct{}{}

			<-c.exiters

			return
		}

		c.runtime.LockOSThread()
		defer c.runtime.UnlockOSThread()

		logger.Debug("stress locked on OSThread")

		if err := c.process.SetAffinity([]int{cpu}); err != nil {
			logger.Warnw("unable to set affinity to a specific cpu, thread might move to another CPU", "error", err)
		}

		totalDuration := 100 * time.Millisecond
		cpuPressureOnDuration := time.Duration(math.Floor(float64(totalDuration) * float64(c.percentage) / float64(100)))
		cpuPressureOffDuration := totalDuration - cpuPressureOnDuration

		logger.Infow("stress is starting", "stress_duration", cpuPressureOnDuration, "pause_duration", cpuPressureOffDuration)

		stressConfigurationCompleted <- struct{}{}

		for {
			stressUntilOff := time.After(cpuPressureOnDuration)

		stressLoop:
			for i := uint64(0); i < math.MaxUint64; i++ {
				select {
				case <-stressUntilOff:
					<-time.After(cpuPressureOffDuration)
					break stressLoop
				case <-c.exiters:
					return
				default:
				}
			}
		}
	}()

	<-stressConfigurationCompleted
}
