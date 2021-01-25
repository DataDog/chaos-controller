// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package main

import (
	"strings"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/injector"
	"github.com/spf13/cobra"
)

var dnsDisruptionCmd = &cobra.Command{
	Use:   "dns-disruption",
	Short: "DNS disruption subcommand",
	Run:   injectAndWait,
	PreRun: func(cmd *cobra.Command, args []string) {
		var err error
		rawHostRecordPairs, _ := cmd.Flags().GetStringArray("host-record-pairs")

		var hostRecordPairs []v1beta1.HostRecordPair

		config.Log.Infow("arguments to dnsDisruptionCmd", "host-record-pairs", rawHostRecordPairs)

		for _, line := range rawHostRecordPairs {
			split := strings.Split(line, ";")

			hostRecordPair := v1beta1.HostRecordPair{
				Host: split[0],
				Record: v1beta1.DNSRecord{
					Type:  split[1],
					Value: split[2],
				},
			}
			hostRecordPairs = append(hostRecordPairs, hostRecordPair)
		}

		spec := hostRecordPairs

		inj, err = injector.NewDNSDisruptionInjector(spec, injector.DNSDisruptionInjectorConfig{Config: config})
		if err != nil {
			log.Fatalw("error initializing the dns disruption injector", "error", err)
		}
	},
}

func init() {
	// We must use a StringArray rather than StringSlice here, because our ip values can contain commas. StringSlice will split on commas.
	dnsDisruptionCmd.Flags().StringArray("host-record-pairs", []string{}, "list of host,record,value tuples as strings") // Array of strings, where each string is a host,record,value tuple??
}
