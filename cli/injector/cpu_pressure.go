// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package main

import (
	"github.com/DataDog/chaos-controller/command"
	"github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/process"
	"github.com/spf13/cobra"
)

var cpuPressureCmd = &cobra.Command{
	Use:   "cpu-pressure",
	Short: "CPU pressure subcommands",
	Run:   injectAndWait,
	PreRun: func(cmd *cobra.Command, args []string) {
		countStr, _ := cmd.Flags().GetString("count")

		cmdFactory := command.NewFactory(disruptionArgs.DryRun)
		processManager := process.NewManager(disruptionArgs.DryRun)
		injectorCmdFactory := injector.NewInjectorCmdFactory(log, processManager, cmdFactory)
		cpuStressArgsBuilder := cpuStressArgsBuilder{}

		for _, config := range configs {
			injectors = append(
				injectors,
				injector.NewCPUPressureInjector(
					config,
					countStr,
					injectorCmdFactory,
					cpuStressArgsBuilder,
				),
			)
		}
	},
}

func init() {
	cpuPressureCmd.Flags().String("count", "", "number of cpus to target, either an integer form or a percentage form appended with a %")
}
