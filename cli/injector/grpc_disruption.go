// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package main

import (
	"strconv"
	"strings"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/injector"
	"github.com/spf13/cobra"
)

var grpcDisruptionCmd = &cobra.Command{
	Use:   "grpc-disruption",
	Short: "GRPC disruption subcommand",
	Run:   injectAndWait,
	PreRun: func(cmd *cobra.Command, args []string) {
		rawEndpointAlterations, _ := cmd.Flags().GetStringArray("endpoint-alterations")

		var endpointAlterations []v1beta1.EndpointAlteration

		// Each value passed to --endpoint-alterations should be of the form `endpoint;alterationtype;alterationvalue`, e.g.
		// `/chaos_dogfood.ChaosDogfood/order;error;ALREADY_EXISTS`
		// `/chaos_dogfood.ChaosDogfood/order;override;{}`

		log.Infow("arguments to grpcDisruptionCmd", "endpoint-alterations", rawEndpointAlterations)

		for _, line := range rawEndpointAlterations {
			split := strings.Split(line, ";")
			if len(split) != 4 {
				log.Fatalw("could not parse --endpoint-alterations argument to grpc-disruption", "offending argument", line)
				continue
			}

			var endpointAlteration v1beta1.EndpointAlteration

			queryPercent, err := strconv.Atoi(split[3])
			if err != nil {
				log.Fatalw("could not parse --endpoint-alterations argument to grpc-disruption", "parsing failed for queryPercent", split[3])
				continue
			}

			if split[1] == "error" {
				endpointAlteration = v1beta1.EndpointAlteration{
					TargetEndpoint: split[0],
					ErrorToReturn:  split[2],
					QueryPercent:   queryPercent,
				}
			} else if split[1] == "override" {
				endpointAlteration = v1beta1.EndpointAlteration{
					TargetEndpoint:   split[0],
					OverrideToReturn: split[2],
					QueryPercent:     queryPercent,
				}
			} else {
				log.Fatalw("GRPC injector does not understand alteration type", "type", split[1])
			}

			endpointAlterations = append(endpointAlterations, endpointAlteration)
		}

		port, _ := cmd.Flags().GetInt("port")

		spec := v1beta1.GRPCDisruptionSpec{
			Port:      port,
			Endpoints: endpointAlterations,
		}
		// create injectors
		for i, config := range configs {
			if i == 0 {
				inj, err := injector.NewGRPCDisruptionInjector(spec, injector.GRPCDisruptionInjectorConfig{Config: config})
				if err != nil {
					log.Fatalw("error initializing the gRPC injector", "error", err)
				}

				injectors = append(injectors, inj)
			}
		}
	},
}

func init() {
	// We must use a StringArray rather than StringSlice here, because our ip values can contain commas. StringSlice will split on commas.
	grpcDisruptionCmd.Flags().StringArray("endpoint-alterations", []string{}, "list of endpoint,alteration_type,alteration_value,optional_query_percent tuples as strings") // `/chaos_dogfood.ChaosDogfood/order;override;{}`
	grpcDisruptionCmd.Flags().Int("port", 0, "port to disrupt on target pod")
}
