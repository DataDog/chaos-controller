// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package main

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/injector"
	"github.com/spf13/cobra"
)

var podFailureCmd = &cobra.Command{
	Use:   "pod-failure",
	Short: "Pod failure subcommands",
	Run:   injectAndWait,
	PreRun: func(cmd *cobra.Command, args []string) {
		kill, _ := cmd.Flags().GetBool("kill")

		// prepare spec
		spec := v1beta1.PodFailureSpec{
			Kill: kill,
		}

		// create injector
		for _, config := range configs {
			inj := injector.NewPodFailureInjector(spec, injector.PodFailureInjectorConfig{Config: config})

			injectors = append(injectors, inj)
		}
	},
}

func init() {
	podFailureCmd.Flags().Bool("kill", false, "If specified, the container will be killed instead of interrupted")
}
