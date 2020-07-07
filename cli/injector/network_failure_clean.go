// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package main

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/container"
	"github.com/DataDog/chaos-controller/injector"
	"github.com/spf13/cobra"
)

var networkFailureCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean injected network failures",
	Run: func(cmd *cobra.Command, args []string) {
		uid, _ := cmd.Flags().GetString("uid")
		containerID, _ := cmd.Flags().GetString("container-id")
		hosts, _ := cmd.Flags().GetStringSlice("hosts")

		// prepare container
		c, err := container.New(containerID)
		if err != nil {
			log.Fatalw("can't create container object", "error", err)
		}
		
		spec := v1beta1.NetworkFailureSpec{
			Hosts:    hosts,
		}

		i := injector.NewNetworkFailureInjector(uid, spec, c, log, ms)
		i.Clean()
	},
}
