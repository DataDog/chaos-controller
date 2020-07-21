// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package main

import "github.com/spf13/cobra"

var networkDisruptionCmd = &cobra.Command{
	Use:   "network-disruption",
	Short: "Network disruption subcommand",
	Run:   nil,
}

func init() {
	networkDisruptionCmd.AddCommand(networkDisruptionInjectCmd)
	networkDisruptionCmd.AddCommand(networkDisruptionCleanCmd)
	networkDisruptionCmd.PersistentFlags().String("container-id", "", "ID of the container to inject/clean")
	networkDisruptionCmd.PersistentFlags().StringSlice("hosts", []string{}, "List of hosts (hostname, single IP or IP block) to apply delay to. If not specified, the delay applies to all the outgoing traffic")
	_ = cobra.MarkFlagRequired(networkDisruptionCmd.PersistentFlags(), "container-id")
}
