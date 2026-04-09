// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package main

import (
	"errors"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/o11y/tags"
)

var diskFullCmd = &cobra.Command{
	Use:   "disk-full",
	Short: "Disk full (ENOSPC) subcommand",
	Run:   injectAndWait,
	PreRun: func(cmd *cobra.Command, args []string) {
		path, _ := cmd.Flags().GetString("path")
		capacity, _ := cmd.Flags().GetString("capacity")
		remaining, _ := cmd.Flags().GetString("remaining")

		spec := v1beta1.DiskFullSpec{
			Path:      path,
			Capacity:  capacity,
			Remaining: remaining,
		}

		for _, config := range configs {
			inj, err := injector.NewDiskFullInjector(spec, injector.DiskFullInjectorConfig{Config: config})
			if err != nil {
				if errors.Is(errors.Unwrap(err), os.ErrNotExist) || strings.Contains(err.Error(), "No such file or directory") {
					log.Errorw("error initializing the disk full injector because the given path does not exist", tags.ErrorKey, err)
				} else if errors.Is(errors.Unwrap(err), os.ErrPermission) {
					log.Errorw("error initializing the disk full injector because the given path is not accessible", tags.ErrorKey, err)
				} else {
					log.Fatalw("error initializing the disk full injector", tags.ErrorKey, err)
				}
			}

			if inj == nil {
				log.Debugln("skipping this injector because path cannot be found on specified container")
				continue
			}

			injectors = append(injectors, inj)
		}
	},
}

func init() {
	diskFullCmd.Flags().String("path", "", "Path to apply disk full disruption to")
	diskFullCmd.Flags().String("capacity", "", "Target fill percentage of total volume capacity (e.g., 95%)")
	diskFullCmd.Flags().String("remaining", "", "Amount of free space to leave on the volume (e.g., 50Mi)")

	_ = cobra.MarkFlagRequired(diskFullCmd.PersistentFlags(), "path")
}
