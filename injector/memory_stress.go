// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/DataDog/chaos-controller/o11y/tags"
	"github.com/DataDog/chaos-controller/process"
	"github.com/DataDog/chaos-controller/types"
)

type memoryStressInjector struct {
	config        *Config
	process       process.Manager
	targetPercent int
	rampDuration  time.Duration
	allocations   [][]byte
	exitCh        chan struct{}
	exitCompleted chan struct{}
}

// NewMemoryStressInjector creates a memory stress injector with the given config
func NewMemoryStressInjector(config Config, targetPercent int, rampDuration time.Duration, process process.Manager) Injector {
	return &memoryStressInjector{
		config:        &config,
		targetPercent: targetPercent,
		rampDuration:  rampDuration,
		process:       process,
	}
}

func (m *memoryStressInjector) TargetName() string {
	return m.config.TargetName()
}

func (*memoryStressInjector) GetDisruptionKind() types.DisruptionKindName {
	return types.DisruptionKindMemoryStress
}

func (m *memoryStressInjector) UpdateConfig(config Config) {
	m.config = &config
}

func (m *memoryStressInjector) Inject() error {
	if len(m.allocations) > 0 {
		return fmt.Errorf("injector contains %d existing allocations, all allocations should be cleaned before re-injecting", len(m.allocations))
	}

	stressPID := m.process.ProcessID()

	m.config.Log.Infow("joining target cgroup for memory stress", tags.PidKey, stressPID)

	if err := m.config.Cgroup.Join(stressPID); err != nil {
		return fmt.Errorf("unable to join cgroup for process '%d': %w", stressPID, err)
	}

	// read memory limit from cgroup
	memoryLimit, err := m.readMemoryLimit()
	if err != nil {
		return fmt.Errorf("unable to read memory limit: %w", err)
	}

	// read current memory usage from cgroup
	currentUsage, err := m.readMemoryUsage()
	if err != nil {
		return fmt.Errorf("unable to read memory usage: %w", err)
	}

	targetBytes := (memoryLimit * int64(m.targetPercent) / 100) - currentUsage
	if targetBytes <= 0 {
		m.config.Log.Infow("current memory usage already exceeds target, nothing to allocate",
			tags.MemoryLimitKey, memoryLimit, tags.CurrentUsageKey, currentUsage, tags.TargetPercentKey, m.targetPercent)

		return nil
	}

	m.config.Log.Infow("starting memory stress",
		tags.MemoryLimitKey, memoryLimit, tags.CurrentUsageKey, currentUsage,
		tags.TargetBytesKey, targetBytes, tags.TargetPercentKey, m.targetPercent, tags.RampDurationKey, m.rampDuration)

	m.exitCh = make(chan struct{}, 1)
	m.exitCompleted = make(chan struct{}, 1)

	stressReady := make(chan struct{}, 1)

	go m.stress(targetBytes, stressReady)

	<-stressReady

	return nil
}

func (m *memoryStressInjector) stress(targetBytes int64, ready chan<- struct{}) {
	defer func() {
		m.exitCompleted <- struct{}{}
	}()

	if m.config.Disruption.DryRun {
		m.config.Log.Debug("memory stress dry run mode activated, skipping allocation, just waiting...")

		ready <- struct{}{}

		<-m.exitCh

		return
	}

	// determine allocation strategy
	var steps int

	var stepDelay time.Duration

	if m.rampDuration > 0 {
		// allocate in 1-second intervals over the ramp duration
		steps = int(m.rampDuration.Seconds())
		if steps < 1 {
			steps = 1
		}

		stepDelay = m.rampDuration / time.Duration(steps)
	} else {
		steps = 1
		stepDelay = 0
	}

	chunkSize := int(targetBytes / int64(steps))
	if chunkSize <= 0 {
		chunkSize = int(targetBytes)
		steps = 1
	}

	m.config.Log.Infow("memory allocation plan", tags.StepsKey, steps, tags.ChunkSizeKey, chunkSize, tags.StepDelayKey, stepDelay)

	ready <- struct{}{}

	for i := 0; i < steps; i++ {
		select {
		case <-m.exitCh:
			m.config.Log.Infow("exit signal received during ramp, stopping allocation", tags.CompletedStepsKey, i, tags.TotalStepsKey, steps)
			return
		default:
		}

		// for the last step, allocate the remainder
		allocSize := chunkSize
		if i == steps-1 {
			allocSize = int(targetBytes) - (chunkSize * i)
		}

		if allocSize <= 0 {
			break
		}

		data, err := mmapAnonymous(allocSize)
		if err != nil {
			m.config.Log.Warnw("mmap allocation failed, stopping ramp", tags.ErrorKey, err, tags.StepKey, i, tags.AllocSizeKey, allocSize)
			break
		}

		m.allocations = append(m.allocations, data)

		if stepDelay > 0 && i < steps-1 {
			select {
			case <-m.exitCh:
				m.config.Log.Infow("exit signal received during ramp delay", tags.CompletedStepsKey, i+1, tags.TotalStepsKey, steps)
				return
			case <-time.After(stepDelay):
			}
		}
	}

	m.config.Log.Infow("memory allocation complete, waiting for exit signal", tags.AllocationsKey, len(m.allocations))

	<-m.exitCh
}

func (m *memoryStressInjector) Clean() error {
	m.config.Log.Infow("cleaning memory stress", tags.AllocationsKey, len(m.allocations))

	if m.exitCh != nil {
		m.exitCh <- struct{}{}

		<-m.exitCompleted

		close(m.exitCh)
		m.exitCh = nil

		close(m.exitCompleted)
		m.exitCompleted = nil
	}

	for _, alloc := range m.allocations {
		if err := munmapMemory(alloc); err != nil {
			m.config.Log.Warnw("failed to munmap allocation", tags.ErrorKey, err)
		}
	}

	m.allocations = nil

	m.config.Log.Info("memory stress cleaned")

	return nil
}

func (m *memoryStressInjector) readMemoryLimit() (int64, error) {
	if m.config.Cgroup.IsCgroupV2() {
		content, err := m.config.Cgroup.Read("", "memory.max")
		if err != nil {
			return 0, fmt.Errorf("unable to read memory.max: %w", err)
		}

		if strings.TrimSpace(content) == "max" {
			// no limit set, use total system memory as fallback
			return 0, fmt.Errorf("memory limit is 'max' (unlimited), cannot determine target bytes")
		}

		return strconv.ParseInt(strings.TrimSpace(content), 10, 64)
	}

	content, err := m.config.Cgroup.Read("memory", "memory.limit_in_bytes")
	if err != nil {
		return 0, fmt.Errorf("unable to read memory.limit_in_bytes: %w", err)
	}

	limit, err := strconv.ParseInt(strings.TrimSpace(content), 10, 64)
	if err != nil {
		return 0, err
	}

	// cgroupv1 reports PAGE_ALIGN(math.MaxInt64) = 9223372036854771712 when no memory limit is set.
	// Use a 4 PiB threshold to detect this sentinel: no real workload has that much RAM.
	const cgroupV1UnlimitedThreshold = int64(4 * 1024 * 1024 * 1024 * 1024 * 1024) // 4 PiB
	if limit >= cgroupV1UnlimitedThreshold {
		return 0, fmt.Errorf("memory limit is unlimited (memory.limit_in_bytes=%d), cannot determine target bytes", limit)
	}

	return limit, nil
}

func (m *memoryStressInjector) readMemoryUsage() (int64, error) {
	if m.config.Cgroup.IsCgroupV2() {
		content, err := m.config.Cgroup.Read("", "memory.current")
		if err != nil {
			return 0, fmt.Errorf("unable to read memory.current: %w", err)
		}

		return strconv.ParseInt(strings.TrimSpace(content), 10, 64)
	}

	content, err := m.config.Cgroup.Read("memory", "memory.usage_in_bytes")
	if err != nil {
		return 0, fmt.Errorf("unable to read memory.usage_in_bytes: %w", err)
	}

	return strconv.ParseInt(strings.TrimSpace(content), 10, 64)
}
