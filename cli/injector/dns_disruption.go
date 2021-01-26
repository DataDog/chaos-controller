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

		// Each value passed to --host-record-pairs should be of the form `hostname;type;value`, e.g.
		// `foo.bar.svc.cluster.local;A;10.0.0.0,10.0.0.13`
		log.Infow("arguments to dnsDisruptionCmd", "host-record-pairs", rawHostRecordPairs)

		for _, line := range rawHostRecordPairs {
			split := strings.Split(line, ";")
			if len(split) != 3 {
				log.Fatalw("could not parse --host-record-pairs argument to dns-disruption", "offending argument", line)
				continue
			}

			hostRecordPair := v1beta1.HostRecordPair{
				Hostname: split[0],
				Record: v1beta1.DNSRecord{
					Type:  split[1],
					Value: split[2],
				},
			}
			hostRecordPairs = append(hostRecordPairs, hostRecordPair)
		}

		inj, err = injector.NewDNSDisruptionInjector(hostRecordPairs, injector.DNSDisruptionInjectorConfig{Config: config})
		if err != nil {
			log.Fatalw("error initializing the dns disruption injector", "error", err)
		}
	},
}

func init() {
	// We must use a StringArray rather than StringSlice here, because our ip values can contain commas. StringSlice will split on commas.
	dnsDisruptionCmd.Flags().StringArray("host-record-pairs", []string{}, "list of host,record,value tuples as strings") // `foo.bar.svc.cluster.local;A;10.0.0.0,10.0.0.13`
}
