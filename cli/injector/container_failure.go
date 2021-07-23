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

var containerFailureCmd = &cobra.Command{
	Use:   "container-failure",
	Short: "Container failure subcommands",
	Run:   injectAndWait,
	PreRun: func(cmd *cobra.Command, args []string) {
		forced, _ := cmd.Flags().GetBool("forced")

		// prepare spec
		spec := v1beta1.ContainerFailureSpec{
			Forced: forced,
		}

		// create injector
		for _, config := range configs {
			inj := injector.NewContainerFailureInjector(spec, injector.ContainerFailureInjectorConfig{Config: config})

			injectors = append(injectors, inj)
		}
	},
}

func init() {
	containerFailureCmd.Flags().Bool("forced", false, "If set to false, the SIGKILL signal will be sent to the container. By default we send the SIGTERM signal.")
}
