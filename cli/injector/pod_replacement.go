// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package main

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/o11y/tags"
	chaostypes "github.com/DataDog/chaos-controller/types"
)

var podReplacementCmd = &cobra.Command{
	Use:   chaostypes.DisruptionKindPodReplacement,
	Short: "Pod replacement subcommands",
	Run:   injectAndWait,
	PreRun: func(cmd *cobra.Command, args []string) {
		deleteStorage, _ := cmd.Flags().GetBool("delete-storage")
		forceDelete, _ := cmd.Flags().GetBool("force-delete")
		gracePeriodSecondsStr, _ := cmd.Flags().GetString("grace-period-seconds")

		// prepare spec
		spec := v1beta1.PodReplacementSpec{
			DeleteStorage: deleteStorage,
			ForceDelete:   forceDelete,
		}

		if gracePeriodSecondsStr != "" {
			gracePeriodSeconds, err := strconv.ParseInt(gracePeriodSecondsStr, 10, 64)
			if err != nil {
				log.Fatalw("invalid grace-period-seconds value", tags.ValueKey, gracePeriodSecondsStr, tags.ErrorKey, err)
			}
			spec.GracePeriodSeconds = &gracePeriodSeconds
		}

		// create injector
		for _, config := range configs {
			podReplacementConfig := injector.PodReplacementInjectorConfig{
				Config: config,
			}

			inj, err := injector.NewPodReplacementInjector(spec, podReplacementConfig)
			if err != nil {
				log.Fatalw("error creating the pod replacement injector", tags.ErrorKey, err)
			}

			injectors = append(injectors, inj)
		}
	},
}

func init() {
	podReplacementCmd.Flags().Bool("delete-storage", true, "If specified, PVCs associated with the target pod will be deleted")
	podReplacementCmd.Flags().Bool("force-delete", false, "If specified, the pod will be force deleted (grace period 0)")
	podReplacementCmd.Flags().String("grace-period-seconds", "", "Grace period in seconds for pod deletion")
}
