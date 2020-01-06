package main

import "github.com/spf13/cobra"

var nodeFailureCmd = &cobra.Command{
	Use:   "node-failure",
	Short: "Node failure subcommands",
	Run:   nil,
}

func init() {
	nodeFailureCmd.AddCommand(nodeFailureInjectCmd)
}
