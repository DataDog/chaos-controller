// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package main

import "github.com/spf13/cobra"

var networkLimitationCommand = &cobra.Command{
	Use:   "network-limitation",
	Short: "Network limitation subcommands",
	Run:   nil,
}

func init() {
	networkLimitationCommand.AddCommand(networkLimitationInjectCmd)
	networkLimitationCommand.PersistentFlags().String("container-id", "", "ID of the container to inject")
	_ = cobra.MarkFlagRequired(networkLimitationCommand.PersistentFlags(), "container-id")
}
