// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package main

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/process"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var cpuPressureCmd = &cobra.Command{
	Use:   "cpu-pressure",
	Short: "CPU pressure subcommands",
	Run:   injectAndWait,
	PreRun: func(cmd *cobra.Command, args []string) {
		countStr, _ := cmd.Flags().GetString("count")
		count := intstr.FromString(countStr)

		spec := v1beta1.CPUPressureSpec{
			Count: &count,
		}

		backgroundProcessManager := process.NewBackgroundProcessManager(log, process.NewManager(disruptionArgs.DryRun), disruptionArgs, deadline)

		for _, config := range configs {
			injectors = append(injectors, injector.NewCPUPressureInjector(config, spec, backgroundProcessManager, cpuStresserCommandBuilder))
		}
	},
}

func init() {
	cpuPressureCmd.Flags().String("count", "", "number of cpus to target, either an integer form or a percentage form appended with a %")
}
