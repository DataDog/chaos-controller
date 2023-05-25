// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package main

import (
	"fmt"

	"github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/process"
	"github.com/spf13/cobra"
)

const (
	percentageFlagName   = "percentage"
	cpuStressCommandName = "cpu-stress"
)

var cpuPressureStressCmd = &cobra.Command{
	Use:   cpuStressCommandName,
	Short: "CPU stress subcommands",
	Run:   injectAndWait,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(configs) != 1 {
			return fmt.Errorf("%s expect a single target configuration, found %d", cpuStressCommandName, len(configs))
		}
		config := configs[0]

		percentage, _ := cmd.Flags().GetInt(percentageFlagName)

		log = log.With("percentage", percentage)
		log.Infow("stressing every CPU allocated to target", "disruption_target", config.TargetName())

		runtime := process.NewRuntime(config.Disruption.DryRun)
		process := process.NewManager(config.Disruption.DryRun)

		injectors = append(
			injectors,
			injector.NewCPUStressInjector(
				config,
				percentage,
				process,
				runtime,
			))

		return nil
	},
}

func init() {
	cpuPressureStressCmd.Flags().Int(percentageFlagName, 100, "percentage of stress to perform on a single cpu")
}

type cpuStressArgsBuilder struct{}

func (c cpuStressArgsBuilder) GenerateArgs(percentage int) []string {
	return []string{
		cpuStressCommandName,
		fmt.Sprintf("--%s=%d", percentageFlagName, percentage),
	}
}
