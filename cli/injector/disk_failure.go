// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package main

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/injector"
	"github.com/spf13/cobra"
)

var diskFailureCmd = &cobra.Command{
	Use:   "disk-failure",
	Short: "Disk failure subcommands",
	Run:   injectAndWait,
	PreRun: func(cmd *cobra.Command, args []string) {
		paths, _ := cmd.Flags().GetStringArray("path")
		exitCode, _ := cmd.Flags().GetString("exit-code")
		probability, _ := cmd.Flags().GetString("probability")

		spec := v1beta1.DiskFailureSpec{
			Paths: paths,
			OpenatSyscall: &v1beta1.OpenatSyscallSpec{
				ExitCode: exitCode,
			},
			Probability: probability,
		}

		// create injectors
		for _, config := range configs {
			inj, err := injector.NewDiskFailureInjector(spec, injector.DiskFailureInjectorConfig{Config: config})
			if err != nil {
				log.Fatalw("error initializing the disk failure injector", "error", err)
			}

			if inj == nil {
				log.Debugln("skipping this injector because path cannot be found on specified container")
				continue
			}

			injectors = append(injectors, inj)
		}
	},
}

func init() {
	diskFailureCmd.Flags().StringArray("path", []string{}, "Path to apply the disk failure")
	diskFailureCmd.Flags().String("probability", "100%", "Probability to apply disk failure")
	diskFailureCmd.Flags().String("exit-code", "", "Exit code to return")
}
