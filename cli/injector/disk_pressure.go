// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package main

import "github.com/spf13/cobra"

var diskPressureCmd = &cobra.Command{
	Use:   "disk-pressure",
	Short: "Disk pressure subcommands",
	Run:   nil,
}

func init() {
	diskPressureCmd.AddCommand(diskPressureInjectCmd)
	diskPressureCmd.AddCommand(diskPressureCleanCmd)
	diskPressureCmd.PersistentFlags().String("path", "", "Path to apply/clean disk pressure to/from (will be applied to the whole disk)")

	_ = cobra.MarkFlagRequired(diskPressureCmd.PersistentFlags(), "path")
}
