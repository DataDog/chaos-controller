// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package main

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/stress"
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

		// prepare spec
		spec := v1beta1.CPUPressureSpec{
			Count: &count,
		}

		stresserManager := stress.NewCPUStresserManager(log)
		// create injector
		for _, config := range configs {
			injector, _ := injector.NewCPUPressureInjector(spec, injector.CPUPressureInjectorConfig{
				Config:          config,
				StresserManager: stresserManager,
			})
			injectors = append(injectors, injector)
		}
	},
}

func init() {
	cpuPressureCmd.Flags().String("count", "", "number of cores to target, either an integer form or a percentage form appended with a %")
}
