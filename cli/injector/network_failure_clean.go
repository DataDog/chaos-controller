// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package main

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/container"
	"github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/metrics"
	"github.com/spf13/cobra"
)

var networkFailureCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean injected network failures",
	Run: func(cmd *cobra.Command, args []string) {
		uid, _ := cmd.Flags().GetString("uid")
		containerID, _ := cmd.Flags().GetString("container-id")

		// prepare container
		c, err := container.New(containerID)
		if err != nil {
			log.Fatalw("can't create container object", "error", err)
		}

		metrics, err := metrics.GetSink("noop")
		if err != nil {
			log.Fatalw("error while creating metric sink", "error", err)
		}

		i, err := injector.NewNetworkFailureInjector(uid, v1beta1.NetworkFailureSpec{}, c, log, metrics)
		if err != nil {
			log.Fatalw("can't initialize the injector", "error", err)
		}
		i.Clean()
	},
}
