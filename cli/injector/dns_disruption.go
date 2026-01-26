// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/network"
	"github.com/DataDog/chaos-controller/o11y/tags"
)

var dnsDisruptionCmd = &cobra.Command{
	Use:   "dns-disruption",
	Short: "DNS disruption subcommands",
	Run:   injectAndWait,
	PreRun: func(cmd *cobra.Command, args []string) {
		// Get flag values
		recordsStr, _ := cmd.Flags().GetString("records")
		port, _ := cmd.Flags().GetInt("port")
		protocol, _ := cmd.Flags().GetString("protocol")

		var spec v1beta1.DNSDisruptionSpec

		// Parse new record-based format if provided
		records, err := parseRecords(recordsStr)
		if err != nil {
			log.Fatalw("error parsing records", "error", err)

			return
		}

		spec = v1beta1.DNSDisruptionSpec{
			Records:  records,
			Port:     port,
			Protocol: protocol,
		}

		// Create injector for each config
		for _, config := range configs {
			// Create IPTables instance
			iptables, err := network.NewIPTables(config.Log, disruptionArgs.DryRun)
			if err != nil {
				log.Fatalw("error creating iptables", tags.ErrorKey, err)

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

// parseRecords parses the records string format: hostname:type:value:ttl;hostname:type:value:ttl
// Handles IPv6 addresses in value field by splitting carefully to avoid breaking on colons within the value
func parseRecords(recordsStr string) ([]v1beta1.DNSRecord, error) {
	if recordsStr == "" {
		return nil, fmt.Errorf("--records flag is required and cannot be empty")
	}

	recordEntries := strings.Split(recordsStr, ";")
	records := make([]v1beta1.DNSRecord, 0, len(recordEntries))

	for _, entry := range recordEntries {
		// Find the first colon (separates hostname from rest)
		firstColon := strings.Index(entry, ":")
		if firstColon == -1 {
			return nil, fmt.Errorf("invalid record format: %s (expected hostname:type:value:ttl)", entry)
		}

		hostname := entry[:firstColon]
		remainder := entry[firstColon+1:]

		// Find the second colon (separates type from value:ttl)
		secondColon := strings.Index(remainder, ":")
		if secondColon == -1 {
			return nil, fmt.Errorf("invalid record format: %s (expected hostname:type:value:ttl)", entry)
		}

		recordType := remainder[:secondColon]
		valueAndTTL := remainder[secondColon+1:]

		// Find the last colon (separates value from TTL)
		lastColon := strings.LastIndex(valueAndTTL, ":")
		if lastColon == -1 {
			return nil, fmt.Errorf("invalid record format: %s (expected hostname:type:value:ttl)", entry)
		}

		value := valueAndTTL[:lastColon]
		ttlStr := valueAndTTL[lastColon+1:]

		// Parse TTL
		ttl, err := strconv.ParseUint(ttlStr, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid TTL value for %s: %s", hostname, ttlStr)
		}

		records = append(records, v1beta1.DNSRecord{
			Hostname: hostname,
			Record: v1beta1.DNSRecordConfig{
				Type:  recordType,
				Value: value,
				TTL:   uint32(ttl),
			},
		})
	}

	return records, nil
}

func init() {
	dnsDisruptionCmd.Flags().String("records", "", "DNS records in format hostname:type:value:ttl;hostname:type:value:ttl")
	dnsDisruptionCmd.Flags().Int("port", 53, "DNS port to intercept (default: 53)")
	dnsDisruptionCmd.Flags().String("protocol", "both", "DNS protocol to disrupt: udp, tcp, or both (default: both)")
}
