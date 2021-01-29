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

var networkDisruptionCmd = &cobra.Command{
	Use:   "network-disruption",
	Short: "Network disruption subcommand",
	Run:   injectAndWait,
	PreRun: func(cmd *cobra.Command, args []string) {
		hosts, _ := cmd.Flags().GetStringSlice("hosts")
		port, _ := cmd.Flags().GetInt("port")
		protocol, _ := cmd.Flags().GetString("protocol")
		flow, _ := cmd.Flags().GetString("flow")
		drop, _ := cmd.Flags().GetInt("drop")
		duplicate, _ := cmd.Flags().GetInt("duplicate")
		corrupt, _ := cmd.Flags().GetInt("corrupt")
		delay, _ := cmd.Flags().GetUint("delay")
		delayJitter, _ := cmd.Flags().GetUint("delay-jitter")
		bandwidthLimit, _ := cmd.Flags().GetInt("bandwidth-limit")

		// prepare injectors
		for i, config := range configs {
			var spec v1beta1.NetworkDisruptionSpec

			// only specify spec for the first injector because
			// the network namespace is shared accross all containers so we do not want
			// to inject more rules
			if i == 0 {
				spec = v1beta1.NetworkDisruptionSpec{
					Hosts:          hosts,
					Port:           port,
					Protocol:       protocol,
					Flow:           flow,
					Drop:           drop,
					Duplicate:      duplicate,
					Corrupt:        corrupt,
					Delay:          delay,
					DelayJitter:    delayJitter,
					BandwidthLimit: bandwidthLimit,
				}
			}

			// generate injector
			injectors = append(injectors, injector.NewNetworkDisruptionInjector(spec, injector.NetworkDisruptionInjectorConfig{Config: config}))
		}
	},
}

func init() {
	networkDisruptionCmd.Flags().StringSlice("hosts", []string{}, "List of hosts (hostname, single IP or IP block) to apply disruptions to. If not specified, the delay applies to all the outgoing traffic")
	networkDisruptionCmd.Flags().Int("port", 0, "Port to disrupt packets from and to")
	networkDisruptionCmd.Flags().String("protocol", "", "Protocol to filter packets on (tcp or udp)")
	networkDisruptionCmd.Flags().String("flow", "egress", "Flow direction to filter on (either egress or ingress)")
	networkDisruptionCmd.Flags().Int("drop", 100, "Percentage to drop packets (100 is a total drop)")
	networkDisruptionCmd.Flags().Int("duplicate", 100, "Percentage to duplicate packets (100 is duplicating each packet)")
	networkDisruptionCmd.Flags().Int("corrupt", 100, "Percentage to corrupt packets (100 is a total corruption)")
	networkDisruptionCmd.Flags().Uint("delay", 0, "Delay to add to the given container in ms")
	networkDisruptionCmd.Flags().Uint("delay-jitter", 0, "Sub-command for Delay; adds specified jitter to delay time")
	networkDisruptionCmd.Flags().Int("bandwidth-limit", 0, "Bandwidth limit in bytes")
}
