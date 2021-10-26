// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package cmd

import (
	"fmt"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	ddmark "github.com/DataDog/chaos-controller/ddmark"
	"github.com/hashicorp/go-multierror"
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
	_, err := DisruptionFromFile(path)
	if err != nil {
		return fmt.Errorf("error reading from disruption at %v: %v", path, err)
	}

	fmt.Println("file is valid !")

	return nil
}

// RunAllValidation runs and concatenate the ddmark validation and the regular api validation
func RunAllValidation(disruption v1beta1.Disruption, rootPath string) error {
	var returnError *multierror.Error

	returnError = multierror.Append(returnError, multierror.Prefix(ddmark.ValidateStructMultierror(disruption, rootPath, "github.com/DataDog/chaos-controller/api/v1beta1"), "ddmark:  "))

	if err := disruption.Spec.Validate(); err != nil {
		returnError = multierror.Append(returnError, multierror.Prefix(err, "validate:"))
	}

	return returnError.ErrorOrNil()
}
