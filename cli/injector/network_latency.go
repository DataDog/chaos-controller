// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package main

import "github.com/spf13/cobra"

var networkLatencyCmd = &cobra.Command{
	Use:   "network-latency",
	Short: "Network latency subcommand",
	Run:   nil,
}

func init() {
	networkLatencyCmd.AddCommand(networkLatencyInjectCmd)
	networkLatencyCmd.AddCommand(networkLatencyCleanCmd)
	networkLatencyCmd.PersistentFlags().String("container-id", "", "ID of the container to inject/clean")
	networkLatencyCmd.PersistentFlags().StringSlice("hosts", []string{}, "List of hosts (hostname, single IP or IP block) to apply delay to. If not specified, the delay applies to all the outgoing traffic")
	_ = cobra.MarkFlagRequired(networkFailureCmd.PersistentFlags(), "container-id")
}
