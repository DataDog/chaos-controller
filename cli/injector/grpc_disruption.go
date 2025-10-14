// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package main

import (
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/o11y/tags"
)

var grpcDisruptionCmd = &cobra.Command{
	Use:   "grpc-disruption",
	Short: "GRPC disruption subcommand",
	Run:   injectAndWait,
	PreRun: func(cmd *cobra.Command, args []string) {
		rawEndpointAlterations, _ := cmd.Flags().GetStringArray("endpoint-alterations")
		port, _ := cmd.Flags().GetInt("port")

		// Each value passed to --endpoint-alterations should be of the form `endpoint;alterationtype;alterationvalue`, e.g.
		// `/chaosdogfood.ChaosDogfood/order;error;ALREADY_EXISTS`
		// `/chaosdogfood.ChaosDogfood/order;override;{}`

		log.Infow("arguments to grpcDisruptionCmd", tags.EndpointAlterationsKey, rawEndpointAlterations)

		var endpointAlterations []v1beta1.EndpointAlteration

		for _, line := range rawEndpointAlterations {
			split := strings.Split(line, ";")
			if len(split) != 4 {
				log.Fatalw("could not parse --endpoint-alterations argument to grpc-disruption", tags.OffendingArgumentKey, line)
				continue
			}

			var endpointAlteration v1beta1.EndpointAlteration

			queryPercent, err := strconv.Atoi(split[3])
			if err != nil {
				log.Fatalw("could not parse --endpoint-alterations argument to grpc-disruption", tags.QueryPercentParsingKey, split[3])
				continue
			}
			switch split[1] {
			case v1beta1.ERROR:
				endpointAlteration = v1beta1.EndpointAlteration{
					TargetEndpoint: split[0],
					ErrorToReturn:  split[2],
					QueryPercent:   queryPercent,
				}
			case v1beta1.OVERRIDE:
				endpointAlteration = v1beta1.EndpointAlteration{
					TargetEndpoint:   split[0],
					OverrideToReturn: split[2],
					QueryPercent:     queryPercent,
				}
			default:
				log.Fatalw("GRPC injector does not understand alteration type", tags.TypeKey, split[1])
			}

			endpointAlterations = append(endpointAlterations, endpointAlteration)
		}

		spec := v1beta1.GRPCDisruptionSpec{
			Port:      port,
			Endpoints: endpointAlterations,
		}

		// create injectors
		for i, config := range configs {
			if i == 0 {
				injectors = append(injectors, injector.NewGRPCDisruptionInjector(spec, injector.GRPCDisruptionInjectorConfig{Config: config}))
			}
		}
	},
}

func init() {
	grpcDisruptionCmd.Flags().StringArray("endpoint-alterations", []string{}, "list of endpoint;alteration_type;alteration_value;optional_query_percent tuples as strings") // `/chaosdogfood.ChaosDogfood/order;override;{}`
	grpcDisruptionCmd.Flags().Int("port", 0, "port to disrupt on target pod")

	_ = cobra.MarkFlagRequired(grpcDisruptionCmd.PersistentFlags(), "port")
}
