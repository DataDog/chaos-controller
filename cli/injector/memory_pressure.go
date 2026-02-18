// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package main

import (
	"time"

	"github.com/DataDog/chaos-controller/command"
	"github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/process"
	"github.com/spf13/cobra"
)

var memoryPressureCmd = &cobra.Command{
	Use:   "memory-pressure",
	Short: "Memory pressure subcommands",
	Run:   injectAndWait,
	PreRun: func(cmd *cobra.Command, args []string) {
		targetPercent, _ := cmd.Flags().GetString("target-percent")
		rampDuration, _ := cmd.Flags().GetDuration("ramp-duration")

		cmdFactory := command.NewFactory(disruptionArgs.DryRun)
		processManager := process.NewManager(disruptionArgs.DryRun)
		injectorCmdFactory := injector.NewInjectorCmdFactory(log, processManager, cmdFactory)
		memoryStressArgsBuilder := memoryStressArgsBuilder{}

		for _, config := range configs {
			injectors = append(
				injectors,
				injector.NewMemoryPressureInjector(
					config,
					targetPercent,
					rampDuration,
					injectorCmdFactory,
					memoryStressArgsBuilder,
				),
			)
		}
	},
}

func init() {
	memoryPressureCmd.Flags().String("target-percent", "", "target memory utilization percentage (e.g., '76%')")
	memoryPressureCmd.Flags().Duration("ramp-duration", time.Duration(0), "duration to ramp up memory usage")
}
