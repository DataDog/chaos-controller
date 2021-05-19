// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package main

import (
	"strconv"
	"strings"

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
		services, _ := cmd.Flags().GetStringSlice("services")
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

			// Only specify spec for the first injector because the network namespace
			// is shared across all containers so we do not want to inject more rules.
			// We must still tag outgoing packets from all containers with a classid
			// in order for the disruption to take effect, so injectors must be created for each container.
			if i == 0 {
				parsedHosts := []v1beta1.NetworkDisruptionHostSpec{}
				parsedServices := []v1beta1.NetworkDisruptionServiceSpec{}

				// parse given hosts
				for _, host := range hosts {
					// parse host with format <host>;<port>;<protocol>
					parsedHost := strings.SplitN(host, ";", 3)

					// cast port to int
					port, err := strconv.Atoi(parsedHost[1])
					if err != nil {
						log.Fatalw("unexpected port parameter", "error", err, "host", host)
					}

					// generate host spec
					parsedHosts = append(parsedHosts, v1beta1.NetworkDisruptionHostSpec{
						Host:     parsedHost[0],
						Port:     port,
						Protocol: parsedHost[2],
					})
				}

				// parse given services
				for _, service := range services {
					// parse service with format <name>;<namespace>
					parsedService := strings.Split(service, ";")
					if len(parsedService) != 2 {
						log.Fatalw("unexpected service", "service", service)
					}

					// generate service spec
					parsedServices = append(parsedServices, v1beta1.NetworkDisruptionServiceSpec{
						Name:      parsedService[0],
						Namespace: parsedService[1],
					})
				}

				spec = v1beta1.NetworkDisruptionSpec{
					Hosts:          parsedHosts,
					Services:       parsedServices,
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
	networkDisruptionCmd.Flags().StringSlice("hosts", []string{}, "List of hosts (hostname, single IP or IP block) with port and protocol to apply disruptions to (format: <host>;<port>;<protocol>)")
	networkDisruptionCmd.Flags().StringSlice("services", []string{}, "List of services to apply disruptions to (format: <name>;<namespace>)")
	networkDisruptionCmd.Flags().String("flow", "egress", "Flow direction to filter on (either egress or ingress)")
	networkDisruptionCmd.Flags().Int("drop", 100, "Percentage to drop packets (100 is a total drop)")
	networkDisruptionCmd.Flags().Int("duplicate", 100, "Percentage to duplicate packets (100 is duplicating each packet)")
	networkDisruptionCmd.Flags().Int("corrupt", 100, "Percentage to corrupt packets (100 is a total corruption)")
	networkDisruptionCmd.Flags().Uint("delay", 0, "Delay to add to the given container in ms")
	networkDisruptionCmd.Flags().Uint("delay-jitter", 0, "Sub-command for Delay; adds specified jitter to delay time")
	networkDisruptionCmd.Flags().Int("bandwidth-limit", 0, "Bandwidth limit in bytes")
}
