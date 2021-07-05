// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package cmd

import (
	"fmt"

	ddmark "github.com/DataDog/chaos-controller/ddmark"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		path, _ := cmd.Flags().GetString("path")
		return ValidateDisruption(path)
	},
}

func init() {
	validateCmd.Flags().String("path", "", "The path to the disruption file to be validated.")
}

func ValidateDisruption(path string) error {
	marshalledStruct, err := DisruptionFromFile(path)
	if err != nil {
		return fmt.Errorf("%v: %v", path, err)
	}

	ddmark.ValidateStruct(marshalledStruct, path,
		"github.com/DataDog/chaos-controller/api/v1beta1",
	)

	// disruption, err := DisruptionFromFile(path)
	// if err != nil {
	// 	return err
	// }

	// err = disruption.Spec.Validate()
	// if err != nil {
	// 	return err
	// }

	return nil
}
