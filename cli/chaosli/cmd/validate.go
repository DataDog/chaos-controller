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
	RunE: func(cmd *cobra.Command, args []string) error {
		path, _ := cmd.Flags().GetString("path")
		return ValidateDisruption(path)
	},
}

func init() {
	validateCmd.Flags().String("path", "", "The path to the disruption file to be validated.")

	if err := validateCmd.MarkFlagRequired("path"); err != nil {
		return
	}
}

func ValidateDisruption(path string) error {
	marshalledStruct, err := DisruptionFromFile(path)
	if err != nil {
		return fmt.Errorf("error reading from disruption at %v: %v", path, err)
	}

	errorList := ddmark.ValidateStruct(marshalledStruct, path,
		"github.com/DataDog/chaos-controller/api/v1beta1",
	)

	err = marshalledStruct.Spec.Validate()
	if err != nil {
		errorList = append(errorList, err)
	}

	ddmark.PrintErrorList(errorList)

	return nil
}
