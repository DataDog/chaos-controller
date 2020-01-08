package main

import "github.com/spf13/cobra"

var networkFailureCmd = &cobra.Command{
	Use:   "network-failure",
	Short: "Network failure subcommand",
	Run:   nil,
}

func init() {
	networkFailureCmd.AddCommand(networkFailureInjectCmd)
	networkFailureCmd.AddCommand(networkFailureCleanCmd)
	networkFailureCmd.PersistentFlags().String("container-id", "", "ID of the container to inject/clean")
	cobra.MarkFlagRequired(networkFailureCmd.PersistentFlags(), "container-id")
}
