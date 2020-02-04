// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package main

import (
	"github.com/DataDog/chaos-fi-controller/api/v1beta1"
	"github.com/DataDog/chaos-fi-controller/injector"
	"github.com/spf13/cobra"
)

var networkLatencyInjectCmd = &cobra.Command{
	Use:   "inject",
	Short: "Inject network latency in the given container",
	Run: func(cmd *cobra.Command, args []string) {
		uid, _ := cmd.Flags().GetString("uid")
		containerID, _ := cmd.Flags().GetString("container-id")
		delay, _ := cmd.Flags().GetUint("delay")
		hosts, _ := cmd.Flags().GetStringSlice("hosts")

		// Prepare injection object
		i := injector.NetworkLatencyInjector{
			ContainerInjector: injector.ContainerInjector{
				Injector: injector.Injector{
					UID: uid,
					Log: log,
				},
				ContainerID: containerID,
			},
			Spec: &v1beta1.NetworkLatencySpec{
				Delay: delay,
				Hosts: hosts,
			},
		}
		i.Inject()
	},
}

func init() {
	networkLatencyInjectCmd.Flags().Uint("delay", 0, "Delay to add to the given container in ms")
	_ = networkLatencyInjectCmd.MarkFlagRequired("delay")
}
