// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package main

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/container"
	"github.com/DataDog/chaos-controller/injector"
	"github.com/spf13/cobra"
)

var diskPressureCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean a disk pressure on the actual node",
	Run: func(cmd *cobra.Command, args []string) {
		uid, _ := cmd.Flags().GetString("uid")
		containerid, _ := cmd.Flags().GetString("container-id")
		path, _ := cmd.Flags().GetString("path")

		// prepare container
		ctn, err := container.New(containerid)
		if err != nil {
			log.Fatalw("can't create container object", "error", err)
		}

		// prepare spec
		spec := v1beta1.DiskPressureSpec{
			Path: path,
		}

		// inject
		i, err := injector.NewDiskPressureInjector(uid, spec, ctn, log, ms)
		if err != nil {
			log.Fatalw("error initializing the disk pressure injector", "error", err)
		}

		i.Clean()
	},
}
