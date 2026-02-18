// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/command"
	"github.com/DataDog/chaos-controller/types"
)

type MemoryStressArgsBuilder interface {
	GenerateArgs(targetPercent int, rampDuration time.Duration) []string
}

type memoryPressureInjector struct {
	config                  Config
	spec                    *v1beta1.MemoryPressureSpec
	injectorCmdFactory      InjectorCmdFactory
	backgroundCmd           command.BackgroundCmd
	cancel                  context.CancelFunc
	memoryStressArgsBuilder MemoryStressArgsBuilder
}

// NewMemoryPressureInjector creates a memory pressure injector with the given config
func NewMemoryPressureInjector(config Config, targetPercent string, rampDuration time.Duration, injectorCmdFactory InjectorCmdFactory, argsBuilder MemoryStressArgsBuilder) Injector {
	return &memoryPressureInjector{
		config: config,
		spec: &v1beta1.MemoryPressureSpec{
			TargetPercent: targetPercent,
			RampDuration:  v1beta1.DisruptionDuration(rampDuration.String()),
		},
		injectorCmdFactory:      injectorCmdFactory,
		memoryStressArgsBuilder: argsBuilder,
	}
}

func (i *memoryPressureInjector) TargetName() string {
	return i.config.TargetName()
}

func (i *memoryPressureInjector) GetDisruptionKind() types.DisruptionKindName {
	return types.DisruptionKindMemoryPressure
}

func (i *memoryPressureInjector) Inject() error {
	i.config.Log.Infow("creating process to stress target memory", "targetPercent", i.spec.TargetPercent, "rampDuration", i.spec.RampDuration)

	pct, err := v1beta1.ParseTargetPercent(i.spec.TargetPercent)
	if err != nil {
		return fmt.Errorf("unable to parse target percent %q: %w", i.spec.TargetPercent, err)
	}

	// clamp to valid range
	if pct < 0 {
		pct = 0
	} else if pct > 100 {
		pct = 100
	}

	rampDuration := i.spec.RampDuration.Duration()

	var cmdErr error
	if i.backgroundCmd, i.cancel, cmdErr = i.injectorCmdFactory.NewInjectorBackgroundCmd(
		i.config.DisruptionDeadline,
		i.config.Disruption,
		i.config.TargetName(),
		i.memoryStressArgsBuilder.GenerateArgs(pct, rampDuration),
	); cmdErr != nil {
		return fmt.Errorf("unable to create new process definition for injector: %w", cmdErr)
	}

	if err := i.backgroundCmd.Start(); err != nil {
		defer i.cancel()

		return fmt.Errorf("unable to start process for injector: %w", err)
	}

	i.backgroundCmd.KeepAlive()

	i.config.Log.Infow("memory stress process started successfully", "targetPercent", pct, "rampDuration", rampDuration)

	return nil
}

func (i *memoryPressureInjector) UpdateConfig(config Config) {
	i.config = config
}

func (i *memoryPressureInjector) Clean() error {
	if i.backgroundCmd == nil {
		return nil
	}

	defer i.cancel()

	if err := i.backgroundCmd.Stop(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return fmt.Errorf("unable to stop background process: %w", err)
	}

	return nil
}
