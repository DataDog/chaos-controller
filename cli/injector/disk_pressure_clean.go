// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package main

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/injector"
	"github.com/spf13/cobra"
)

var diskPressureCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean a disk pressure on the actual node",
	Run: func(cmd *cobra.Command, args []string) {
		path, _ := cmd.Flags().GetString("path")

		// prepare spec
		spec := v1beta1.DiskPressureSpec{
			Path: path,
		}

		// inject
		i, err := injector.NewDiskPressureInjector(spec, injector.DiskPressureInjectorConfig{Config: config})
		if err != nil {
			log.Fatalw("error initializing the disk pressure injector", "error", err)
		}

		i.Clean()
	},
}
