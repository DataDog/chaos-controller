// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

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
	_ = cobra.MarkFlagRequired(networkFailureCmd.PersistentFlags(), "container-id")
}
