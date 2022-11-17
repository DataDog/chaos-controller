// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package main

import (
	"errors"
	"os"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/injector"
	"github.com/spf13/cobra"
)

var diskFailureCmd = &cobra.Command{
	Use:   "disk-failure",
	Short: "Disk failure subcommands",
	Run:   injectAndWait,
	PreRun: func(cmd *cobra.Command, args []string) {
		spec := v1beta1.DiskFailureSpec{}

		// create injectors
		for _, config := range configs {
			inj, err := injector.NewDiskFailureInjector(spec, injector.DiskFailureInjectorConfig{Config: config})
			if err != nil {
				if errors.Is(errors.Unwrap(err), os.ErrNotExist) {
					log.Errorw("error initializing the disk failure injector because the given path does not exist", "error", err)
				} else {
					log.Fatalw("error initializing the disk failure injector", "error", err)
				}
			}

			if inj == nil {
				// skipping this injector because path cannot be found on specified container
				continue
			}

			injectors = append(injectors, inj)
		}
	},
}

func init() {
	_ = cobra.MarkFlagRequired(diskFailureCmd.PersistentFlags(), "path")
}
