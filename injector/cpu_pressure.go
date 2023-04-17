// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package injector

import (
	"fmt"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/process"
	"github.com/DataDog/chaos-controller/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type cpuPressureInjector struct {
	config                    Config
	spec                      v1beta1.CPUPressureSpec
	count                     int
	backgroundProcessManager  process.BackgroundProcessManager
	backgroundProcess         process.BackgroundProcess
	cpuStresserCommandBuilder func(int) []string
}

// NewCPUPressureInjector creates a CPU pressure injector with the given config
func NewCPUPressureInjector(config Config, spec v1beta1.CPUPressureSpec, backgroundProcessManager process.BackgroundProcessManager, cpuStresserCommandBuilder func(int) []string) Injector {
	return &cpuPressureInjector{
		config:                    config,
		spec:                      spec,
		backgroundProcessManager:  backgroundProcessManager,
		cpuStresserCommandBuilder: cpuStresserCommandBuilder,
	}
}

func (i *cpuPressureInjector) GetDisruptionKind() types.DisruptionKindName {
	return types.DisruptionKindCPUPressure
}

func (i *cpuPressureInjector) Inject() error {
	var (
		percentage int
		err        error
	)

	if i.spec.Count.Type == intstr.Int { // if a number is provided, calculate a percentage against the whole cpus assigned to current container TODO should we enforce max to 100%?
		if assignedCPUs, err := i.config.Cgroup.ReadCPUSet(); err != nil {
			return fmt.Errorf("unable to read CPUSet for current container: %w", err)
		} else if percentage, err = intstr.GetScaledValueFromIntOrPercent(i.spec.Count, assignedCPUs.Size(), true); err != nil {
			return fmt.Errorf("unable to caculate stress percentage from number of cpu: %w", err)
		}
	} else if percentage, err = intstr.GetScaledValueFromIntOrPercent(i.spec.Count, 100, true); err != nil { // if a percentage is provided, keep it as is
		return fmt.Errorf("unable to caculate stress percentage from percentage: %w", err)
	}

	if i.backgroundProcess, err = i.backgroundProcessManager.Start(
		i.config.TargetContainer.Name(),
		i.cpuStresserCommandBuilder(percentage)...,
	); err != nil {
		return fmt.Errorf("unable to start background process for injector: %w", err)
	}

	i.config.Log.Info("all routines have been created successfully, now stressing in background")

	return nil
}

func (i *cpuPressureInjector) UpdateConfig(config Config) {
	i.config = config
}

func (i *cpuPressureInjector) Clean() error {
	i.backgroundProcess.Stop()

	return nil
}
