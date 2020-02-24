// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package main

import (
	"github.com/DataDog/chaos-fi-controller/api/v1beta1"
	"github.com/DataDog/chaos-fi-controller/container"
	"github.com/DataDog/chaos-fi-controller/injector"
	"github.com/spf13/cobra"
)

var networkLatencyCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean injected network latency",
	Run: func(cmd *cobra.Command, args []string) {
		uid, _ := cmd.Flags().GetString("uid")
		containerID, _ := cmd.Flags().GetString("container-id")
		hosts, _ := cmd.Flags().GetStringSlice("hosts")

		// prepare container
		c, err := container.New(containerID)
		if err != nil {
			log.Fatalw("can't create container object", "error", err)
		}

		// prepare spec
		spec := v1beta1.NetworkLatencySpec{
			Hosts: hosts,
		}

		// clean
		i := injector.NewNetworkLatencyInjector(uid, spec, c, log)
		i.Clean()
	},
}
