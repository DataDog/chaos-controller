// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package main

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/injector"
	"github.com/spf13/cobra"
)

var cpuPressureCmd = &cobra.Command{
	Use:   "cpu-pressure",
	Short: "CPU pressure subcommands",
	Run:   injectAndWait,
	PreRun: func(cmd *cobra.Command, args []string) {
		// prepare spec
		spec := v1beta1.CPUPressureSpec{}

		// create injector
		inj = injector.NewCPUPressureInjector(spec, injector.CPUPressureInjectorConfig{Config: config})
	},
}
