// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "create a disruption.",
	Long:  `creates a disruption given input from the user.`,
	Run: func(cmd *cobra.Command, args []string) {
		path, _ := cmd.Flags().GetString("path")
		fmt.Println(create(path))
	},
}

func create(filePath string) string {
	if filePath == "" {
		return pathError
	}

	return "Creation TODO"
}

func init() {
	createCmd.Flags().String("path", "", "The path to create the disruption config.")
}
