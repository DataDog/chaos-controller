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

var networkLimitationInjectCmd = &cobra.Command{
	Use:   "inject",
	Short: "Inject an artificial network bandwidth limit on the actual node",
	Run: func(cmd *cobra.Command, args []string) {
		uid, _ := cmd.Flags().GetString("uid")
		containerID, _ := cmd.Flags().GetString("container-id")
		bytesPerSec, _ := cmd.Flags().GetUint("bytes-per-sec")
		hosts, _ := cmd.Flags().GetStringSlice("hosts")

		// prepare container
		c, err := container.New(containerID)
		if err != nil {
			log.Fatalw("can't create container object", "error", err)
		}

		// prepare spec
		spec := v1beta1.NetworkLimitationSpec{
			BytesPerSec: bytesPerSec,
			Hosts:       hosts,
		}

		// inject
		i := injector.NewNetworkLimitationInjector(uid, spec, c, log, ms)
		i.Inject()
	},
}

func init() {
	networkLimitationInjectCmd.Flags().Uint("bytes-per-sec", 1000000000, "Bytes per second to limit bandwidth to")
	_ = networkLimitationInjectCmd.MarkFlagRequired("bytes-per-sec")
}
