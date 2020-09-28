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

var networkDisruptionInjectCmd = &cobra.Command{
	Use:   "inject",
	Short: "Inject a network failure in the given container",
	Run: func(cmd *cobra.Command, args []string) {
		uid, _ := cmd.Flags().GetString("uid")
		containerID, _ := cmd.Flags().GetString("container-id")
		hosts, _ := cmd.Flags().GetStringSlice("hosts")
		port, _ := cmd.Flags().GetInt("port")
		protocol, _ := cmd.Flags().GetString("protocol")
		flow, _ := cmd.Flags().GetString("flow")
		drop, _ := cmd.Flags().GetInt("drop")
		corrupt, _ := cmd.Flags().GetInt("corrupt")
		delay, _ := cmd.Flags().GetUint("delay")
		bandwidthLimit, _ := cmd.Flags().GetInt("bandwidth-limit")

		// prepare container
		c, err := container.New(containerID)
		if err != nil {
			log.Fatalw("can't create container object", "error", err)
		}

		// check that at least one disruption has been specified
		if drop == 0 && corrupt == 0 && delay == 0 && bandwidthLimit == 0 {
			log.Fatal("at least one disruption must be specified")
		}

		// prepare injection object
		spec := v1beta1.NetworkDisruptionSpec{
			Hosts:          hosts,
			Port:           port,
			Protocol:       protocol,
			Flow:           flow,
			Drop:           drop,
			Corrupt:        corrupt,
			Delay:          delay,
			BandwidthLimit: bandwidthLimit,
		}
		i := injector.NewNetworkDisruptionInjector(uid, spec, c, log, ms)
		i.Inject()
	},
}

func init() {
	networkDisruptionInjectCmd.Flags().Int("port", 0, "Port to drop packets from and to")
	networkDisruptionInjectCmd.Flags().String("protocol", "", "Protocol to filter packets on (tcp or udp)")
	networkDisruptionInjectCmd.Flags().String("flow", "egress", "Flow direction to filter on (either egress or ingress)")
	networkDisruptionInjectCmd.Flags().Int("drop", 100, "Percentage to drop packets (100 is a total drop)")
	networkDisruptionInjectCmd.Flags().Int("corrupt", 100, "Percentage to corrupt packets (100 is a total corruption)")
	networkDisruptionInjectCmd.Flags().Uint("delay", 0, "Delay to add to the given container in ms")
	networkDisruptionInjectCmd.Flags().Int("bandwidth-limit", 0, "Bandwidth limit in bytes")
}
