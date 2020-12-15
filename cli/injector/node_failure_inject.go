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

var nodeFailureInjectCmd = &cobra.Command{
	Use:   "inject",
	Short: "Inject a node failure on the actual node",
	Run: func(cmd *cobra.Command, args []string) {
		shutdown, _ := cmd.Flags().GetBool("shutdown")

		// prepare spec
		spec := v1beta1.NodeFailureSpec{
			Shutdown: shutdown,
		}

		// inject
		i, err := injector.NewNodeFailureInjector(spec, injector.NodeFailureInjectorConfig{Config: config})
		if err != nil {
			log.Fatalw("error creating the node injector", "error", err)
		}

		i.Inject()
	},
}

func init() {
	nodeFailureInjectCmd.Flags().Bool("shutdown", false, "If specified, the host will shut down instead of reboot")
}
