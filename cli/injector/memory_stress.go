// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/process"
)

const (
	memoryTargetPercentFlagName = "target-percent"
	memoryRampDurationFlagName  = "ramp-duration"
	memoryStressCommandName     = "memory-stress"
)

var memoryPressureStressCmd = &cobra.Command{
	Use:   memoryStressCommandName,
	Short: "Memory stress subcommands",
	Run:   injectAndWait,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(configs) != 1 {
			return fmt.Errorf("%s expects a single target configuration, found %d", memoryStressCommandName, len(configs))
		}

		config := configs[0]

		targetPercent, _ := cmd.Flags().GetInt(memoryTargetPercentFlagName)
		rampDuration, _ := cmd.Flags().GetDuration(memoryRampDurationFlagName)

		log.Infow("stressing memory allocated to target",
			"targetPercent", targetPercent,
			"rampDuration", rampDuration,
			"targetName", config.TargetName(),
		)

		processManager := process.NewManager(config.Disruption.DryRun)

		injectors = append(
			injectors,
			injector.NewMemoryStressInjector(
				config,
				targetPercent,
				rampDuration,
				processManager,
			))

		return nil
	},
}

func init() {
	memoryPressureStressCmd.Flags().Int(memoryTargetPercentFlagName, 50, "target memory utilization percentage")
	memoryPressureStressCmd.Flags().Duration(memoryRampDurationFlagName, time.Duration(0), "duration to ramp up memory usage")
}

type memoryStressArgsBuilder struct{}

func (m memoryStressArgsBuilder) GenerateArgs(targetPercent int, rampDuration time.Duration) []string {
	args := []string{
		memoryStressCommandName,
		fmt.Sprintf("--%s=%d", memoryTargetPercentFlagName, targetPercent),
	}

	if rampDuration > 0 {
		args = append(args, fmt.Sprintf("--%s=%s", memoryRampDurationFlagName, rampDuration.String()))
	}

	return args
}
