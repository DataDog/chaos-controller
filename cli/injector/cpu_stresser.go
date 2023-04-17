// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package main

import (
	"fmt"

	"github.com/DataDog/chaos-controller/injector"
	"github.com/spf13/cobra"
)

const (
	percentageFlagName     = "percentage"
	cpuStressedCommandName = "cpu-stresser"
)

var cpuPressureStresserCmd = &cobra.Command{
	Use:   cpuStressedCommandName,
	Short: "CPU stresser subcommands",
	Run:   injectAndWait,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(configs) != 1 {
			return fmt.Errorf("%s expect a single container configuration, found %d", cpuStressedCommandName, len(configs))
		}

		percentage, _ := cmd.Flags().GetInt(percentageFlagName)
		log.Infof("%s will stress %d%% of allocated CPU for container %s", cpuStressedCommandName, percentage, configs[0].TargetContainer.Name())

		injectors = append(
			injectors,
			injector.NewCPUStresserInjector(
				configs[0],
				percentage,
			))

		return nil
	},
}

func init() {
	cpuPressureStresserCmd.Flags().Int(percentageFlagName, 100, "percentage of stress to perform on a single cpu")
}

func cpuStresserCommandBuilder(percentage int) []string {
	return []string{
		cpuStressedCommandName,
		fmt.Sprintf("--%s=%d", percentageFlagName, percentage),
	}
}
