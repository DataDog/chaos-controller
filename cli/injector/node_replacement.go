// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package main

import (
	"strconv"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/injector"
	"github.com/spf13/cobra"
)

var nodeReplacementCmd = &cobra.Command{
	Use:   "node-replacement",
	Short: "Node replacement subcommands",
	Run:   injectAndWait,
	PreRun: func(cmd *cobra.Command, args []string) {
		deleteStorage, _ := cmd.Flags().GetBool("delete-storage")
		forceDelete, _ := cmd.Flags().GetBool("force-delete")
		gracePeriodSecondsStr, _ := cmd.Flags().GetString("grace-period-seconds")

		// prepare spec
		spec := v1beta1.NodeReplacementSpec{
			DeleteStorage: deleteStorage,
			ForceDelete:   forceDelete,
		}

		if gracePeriodSecondsStr != "" {
			gracePeriodSeconds, err := strconv.ParseInt(gracePeriodSecondsStr, 10, 64)
			if err != nil {
				log.Fatalw("invalid grace-period-seconds value", "value", gracePeriodSecondsStr, "error", err)
			}
			spec.GracePeriodSeconds = &gracePeriodSeconds
		}

		// create injector
		for _, config := range configs {
			nodeReplacementConfig := injector.NodeReplacementInjectorConfig{
				Config: config,
			}

			inj, err := injector.NewNodeReplacementInjector(spec, nodeReplacementConfig)
			if err != nil {
				log.Fatalw("error creating the node replacement injector", "error", err)
			}

			injectors = append(injectors, inj)
		}
	},
}

func init() {
	nodeReplacementCmd.Flags().Bool("delete-storage", true, "If specified, PVCs associated with pods on the node will be deleted")
	nodeReplacementCmd.Flags().Bool("force-delete", false, "If specified, pods will be force deleted (grace period 0)")
	nodeReplacementCmd.Flags().String("grace-period-seconds", "", "Grace period in seconds for pod deletion")
}