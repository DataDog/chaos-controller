// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package main

import "github.com/spf13/cobra"

var cpuPressureCmd = &cobra.Command{
	Use:   "cpu-pressure",
	Short: "CPU pressure subcommands",
	Run:   nil,
}

func init() {
	cpuPressureCmd.AddCommand(cpuPressureInjectCmd)
}
