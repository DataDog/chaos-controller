// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package main

import (
	"github.com/DataDog/chaos-fi-controller/container"
	"github.com/DataDog/chaos-fi-controller/injector"
	"github.com/spf13/cobra"
)

var networkFailureCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean injected network failures",
	Run: func(cmd *cobra.Command, args []string) {
		uid, _ := cmd.Flags().GetString("uid")
		containerID, _ := cmd.Flags().GetString("container-id")

		// prepare container
		c, err := container.New(containerID)
		if err != nil {
			log.Fatalw("can't create container object", "error", err)
		}

		i := injector.NetworkFailureInjector{
			ContainerInjector: injector.ContainerInjector{
				Injector: injector.Injector{
					UID: uid,
					Log: log,
				},
				Container: c,
			},
		}
		i.Clean()
	},
}
