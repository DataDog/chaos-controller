// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "validate disruption config",
	Long:  `validates the yaml of the disruption for structure.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		path, _ := cmd.Flags().GetString("path")
		return validatePath(path)
	},
	RunE: ValidateDisruption,
}

func init() {
	validateCmd.Flags().String("path", "", "The path to the disruption file to be validated.")
}

func ValidateDisruption(cmd *cobra.Command, args []string) error {
	path, _ := cmd.Flags().GetString("path")
	disruption, _ := DisruptionFromFile(path)

	err := disruption.Spec.Validate()
	if err != nil {
		return err
	}

	return nil
}

func validatePath(filePath string) error {
	if filePath == "" {
		return fmt.Errorf("no path given, exiting")
	}

	return nil
}
