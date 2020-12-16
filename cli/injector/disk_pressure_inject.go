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

var diskPressureInjectCmd = &cobra.Command{
	Use:   "inject",
	Short: "Inject a disk pressure on the actual node",
	Run: func(cmd *cobra.Command, args []string) {
		path, _ := cmd.Flags().GetString("path")
		writeBytesPerSec, _ := cmd.Flags().GetInt("write-bytes-per-sec")
		readBytesPerSec, _ := cmd.Flags().GetInt("read-bytes-per-sec")

		// prepare spec
		var writeBytesPerSecP *int
		if writeBytesPerSec != 0 {
			writeBytesPerSecP = &writeBytesPerSec
		}

		var readBytesPerSecP *int
		if readBytesPerSec != 0 {
			readBytesPerSecP = &readBytesPerSec
		}

		spec := v1beta1.DiskPressureSpec{
			Path: path,
			Throttling: v1beta1.DiskPressureThrottlingSpec{
				ReadBytesPerSec:  readBytesPerSecP,
				WriteBytesPerSec: writeBytesPerSecP,
			},
		}

		// inject
		i, err := injector.NewDiskPressureInjector(spec, injector.DiskPressureInjectorConfig{Config: config})
		if err != nil {
			log.Fatalw("error initializing the disk pressure injector", "error", err)
		}

		i.Inject()
	},
}

func init() {
	diskPressureInjectCmd.Flags().Int("write-bytes-per-sec", 0, "Bytes per second throttling limit")
	diskPressureInjectCmd.Flags().Int("read-bytes-per-sec", 0, "Bytes per second throttling limit")
}
