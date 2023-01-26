// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "display chaosli version",
	Long:  `shows the currently used version of the chaosli - upgrade with brew if necessary`,
	Run: func(cmd *cobra.Command, args []string) {
		if Version == "v0" {
			Version = "version unspecified"
		}
		fmt.Println("chaosli", Version)
	},
}
