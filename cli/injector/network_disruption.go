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
		allowedHosts, _ := cmd.Flags().GetStringSlice("allowed-hosts")
		services, _ := cmd.Flags().GetStringSlice("services")
		pods, _ := cmd.Flags().GetStringSlice("pods")
		nodes, _ := cmd.Flags().GetStringSlice("nodes")
		drop, _ := cmd.Flags().GetInt("drop")
		duplicate, _ := cmd.Flags().GetInt("duplicate")
		corrupt, _ := cmd.Flags().GetInt("corrupt")
		delay, _ := cmd.Flags().GetUint("delay")
		delayJitter, _ := cmd.Flags().GetUint("delay-jitter")
		bandwidthLimit, _ := cmd.Flags().GetInt("bandwidth-limit")

		// prepare injectors
		for i, config := range configs {
			var spec v1beta1.NetworkDisruptionSpec

			// Only specify spec for the first injector because the network namespace
			// is shared across all containers so we do not want to inject more rules.
			// We must still tag outgoing packets from all containers with a classid
			// in order for the disruption to take effect, so injectors must be created for each container.
			if i == 0 {
				parsedHosts, err := v1beta1.NetworkDisruptionHostSpecFromString(hosts)
				if err != nil {
					log.Fatalw("error parsing hosts", "error", err)
				}

				parsedAllowedHosts, err := v1beta1.NetworkDisruptionHostSpecFromString(allowedHosts)
				if err != nil {
					log.Fatalw("error parsing allowed hosts", "error", err)
				}

				parsedServices, err := v1beta1.NetworkDisruptionServiceSpecFromString(services)
				if err != nil {
					log.Fatalw("error parsing services", "error", err)
				}

				parsedPods, err := v1beta1.NetworkDisruptionPodsSpecFromString(pods)
				if err != nil {
					log.Fatalw("error parsing pods", "error", err)
				}

				log.Infof("%s", pods)

				parsedNodes, err := v1beta1.NetworkDisruptionNodesSpecFromString(nodes)
				if err != nil {
					log.Fatalw("error parsing nodes", "error", err)
				}

				spec = v1beta1.NetworkDisruptionSpec{
					Hosts:          parsedHosts,
					AllowedHosts:   parsedAllowedHosts,
					Services:       parsedServices,
					Pods:           parsedPods,
					Nodes:          parsedNodes,
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
	networkDisruptionCmd.Flags().StringSlice("hosts", []string{}, "List of hosts (hostname, single IP or IP block) with port and protocol to apply disruptions to (format: <host>;<port>;<protocol>;<flow>)")
	networkDisruptionCmd.Flags().StringSlice("allowed-hosts", []string{}, "List of allowed hosts not being impacted by the disruption (hostname, single IP or IP block) with port and protocol to apply disruptions to (format: <host>;<port>;<protocol>;<flow>)")
	networkDisruptionCmd.Flags().StringSlice("services", []string{}, "List of services to apply disruptions to (format: <name>;<namespace>)")
	networkDisruptionCmd.Flags().StringSlice("pods", []string{}, "List of pods selectors to apply disruptions to (format: ;<namespace>)")
	networkDisruptionCmd.Flags().StringSlice("nodes", []string{}, "List of nodes selectors to apply disruptions to (format: <name>;<namespace>)")
	networkDisruptionCmd.Flags().Int("drop", 100, "Percentage to drop packets (100 is a total drop)")
	networkDisruptionCmd.Flags().Int("duplicate", 100, "Percentage to duplicate packets (100 is duplicating each packet)")
	networkDisruptionCmd.Flags().Int("corrupt", 100, "Percentage to corrupt packets (100 is a total corruption)")
	networkDisruptionCmd.Flags().Uint("delay", 0, "Delay to add to the given container in ms")
	networkDisruptionCmd.Flags().Uint("delay-jitter", 0, "Sub-command for Delay; adds specified jitter to delay time")
	networkDisruptionCmd.Flags().Int("bandwidth-limit", 0, "Bandwidth limit in bytes")
}
