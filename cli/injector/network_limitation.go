// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package main

import "github.com/spf13/cobra"

var networkLimitationCmd = &cobra.Command{
	Use:   "network-limitation",
	Short: "Network limitation subcommands",
	Run:   nil,
}

func init() {
	networkLimitationCmd.AddCommand(networkLimitationInjectCmd)
	networkLimitationCmd.AddCommand(networkLimitationCleanCmd)
	networkLimitationCmd.PersistentFlags().String("container-id", "", "ID of the container to inject")
	_ = cobra.MarkFlagRequired(networkLimitationCmd.PersistentFlags(), "container-id")
}
