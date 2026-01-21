// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package main

import (
	"strings"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/network"
	"github.com/spf13/cobra"
)

var dnsDisruptionCmd = &cobra.Command{
	Use:   "dns-disruption",
	Short: "DNS disruption subcommands",
	Run:   injectAndWait,
	PreRun: func(cmd *cobra.Command, args []string) {
		// Get flag values
		domainsStr, _ := cmd.Flags().GetString("domains")
		failureMode, _ := cmd.Flags().GetString("failure-mode")
		port, _ := cmd.Flags().GetInt("port")
		protocol, _ := cmd.Flags().GetString("protocol")

		// Parse domains (comma-separated)
		domains := []string{}
		if domainsStr != "" {
			domains = strings.Split(domainsStr, ",")
		}

		// Prepare spec
		spec := v1beta1.DNSDisruptionSpec{
			Domains:     domains,
			FailureMode: failureMode,
			Port:        port,
			Protocol:    protocol,
		}

		// Create injector for each config
		for _, config := range configs {
			// Create IPTables instance
			iptables, err := network.NewIPTables(config.Log, disruptionArgs.DryRun)
			if err != nil {
				log.Fatalw("error creating iptables", "error", err)
				return
			}

			// Create injector config
			injectorConfig := injector.DNSDisruptionInjectorConfig{
				Config:   config,
				IPTables: iptables,
			}

			// Create injector
			inj := injector.NewDNSDisruptionInjector(spec, injectorConfig)
			injectors = append(injectors, inj)
		}
	},
}

func init() {
	dnsDisruptionCmd.Flags().String("domains", "", "Comma-separated list of domains to disrupt (e.g., example.com,api.example.com)")
	dnsDisruptionCmd.Flags().String("failure-mode", "nxdomain", "DNS failure mode: nxdomain, drop, servfail, or random-ip")
	dnsDisruptionCmd.Flags().Int("port", 53, "DNS port to intercept (default: 53)")
	dnsDisruptionCmd.Flags().String("protocol", "both", "DNS protocol to disrupt: udp, tcp, or both (default: both)")
}
