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
