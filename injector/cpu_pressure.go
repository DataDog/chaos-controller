// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package injector

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/command"
	"github.com/DataDog/chaos-controller/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type CPUStressArgsBuilder interface {
	GenerateArgs(int) []string
}

type cpuPressureInjector struct {
	config               Config
	spec                 *v1beta1.CPUPressureSpec
	injectorCmdFactory   InjectorCmdFactory
	backgroundCmd        command.BackgroundCmd
	cancel               context.CancelFunc
	cpuStressArgsBuilder CPUStressArgsBuilder
}

// NewCPUPressureInjector creates a CPU pressure injector with the given config
func NewCPUPressureInjector(config Config, count string, injectorCmdFactory InjectorCmdFactory, argsBuilder CPUStressArgsBuilder) Injector {
	intstrCount := intstr.Parse(count)

	return &cpuPressureInjector{
		config,
		&v1beta1.CPUPressureSpec{
			Count: &intstrCount,
		},
		injectorCmdFactory,
		nil,
		nil,
		argsBuilder,
	}
}

func (i *cpuPressureInjector) TargetName() string {
	return i.config.TargetName()
}

func (i *cpuPressureInjector) GetDisruptionKind() types.DisruptionKindName {
	return types.DisruptionKindCPUPressure
}

func (i *cpuPressureInjector) Inject() error {
	i.config.Log.Infow("creating processes to stress target", "count", i.spec.Count)

	var (
		percentage int
		err        error
	)

	if i.spec.Count != nil && i.spec.Count.Type == intstr.Int { // if a number is provided, calculate a percentage against the amount of cpus assigned to current target
		assignedCPUs, err := i.config.Cgroup.ReadCPUSet()
		if err != nil {
			return fmt.Errorf("unable to read CPUSet for current container: %w", err)
		}

		percentage = int(math.Floor(float64(i.spec.Count.IntValue()) / float64(assignedCPUs.Size()) * 100))

		i.config.Log.Infow("percentage calculated from number of cpu", "provided_value", i.spec.Count, "assigned_cpus", assignedCPUs, "percentage", percentage)
	} else if percentage, err = intstr.GetScaledValueFromIntOrPercent(i.spec.Count, 100, true); err != nil { // if a percentage is provided, keep it as is
		return fmt.Errorf("unable to calculate stress percentage for '%s': %w", i.spec.Count, err)
	} else {
		i.config.Log.Infow("percentage calculated from percentage", "provided_value", i.spec.Count, "percentage", percentage)
	}

	// If a range is expected, it should be checked earlier than here, let's not fail in the injector that is far away from our users
	if percentage < 0 {
		percentage = 0
	} else if 100 < percentage {
		percentage = 100
	}

	if i.backgroundCmd, i.cancel, err = i.injectorCmdFactory.NewInjectorBackgroundCmd(
		i.config.DisruptionDeadline,
		i.config.Disruption,
		i.config.TargetName(),
		i.cpuStressArgsBuilder.GenerateArgs(percentage),
	); err != nil {
		return fmt.Errorf("unable to create new process definition for injector: %w", err)
	}

	if err := i.backgroundCmd.Start(); err != nil {
		defer i.cancel()

		return fmt.Errorf("unable to start process for injector: %w", err)
	}

	i.backgroundCmd.KeepAlive()

	i.config.Log.Infow("all routines have been created successfully, now stressing in background", "percentage", percentage)

	return nil
}

func (i *cpuPressureInjector) UpdateConfig(config Config) {
	i.config = config
}

func (i *cpuPressureInjector) Clean() error {
	if i.backgroundCmd == nil {
		return nil
	}

	defer i.cancel()

	if err := i.backgroundCmd.Stop(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return fmt.Errorf("unable to stop background process: %w", err)
	}

	return nil
}
