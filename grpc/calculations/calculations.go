// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package calculations

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/DataDog/chaos-controller/grpc/disruptionlistener"
)

/*
 * When an integer from 0 to 100 is randomly selected, a slice is referenced to identify the disruption to apply
 * to a query. The slice represents the user's preference for the proportion of queries affected by each alteration.
 * See docs/grpc_disruption/interceptor_algorithm.md for examples.
 */

// ConvertSpecifications takes a series of alterations configured for a target endpoint where
// assignments are distributed based on percentage odds (QueryPercent) expected for different return alterations
// and returns a slice where the slice's "index" between 0 and some number less than 100 are assigned
// Alterations which reappear as many times as the requested query percentage
func ConvertSpecifications(endpointSpecList []*pb.AlterationSpec) ([]AlterationConfiguration, error) {
	alterationToQueryPercent, err := GetPercentagePerAlteration(endpointSpecList)
	if err != nil {
		return nil, err
	}

	return FlattenAlterationMap(alterationToQueryPercent), nil
}

// GetPercentagePerAlteration takes a series of alterations configured for a target endpoint and returns a mapping
// from the alteration to the percentage of queries which will be altered by it
func GetPercentagePerAlteration(endpointSpecList []*pb.AlterationSpec) (map[AlterationConfiguration]QueryPercent, error) {
	mapping := make(map[AlterationConfiguration]QueryPercent)

	// unquantifiedAlts is a holding variable used later to calculate and assign percentages to alterations which do not specify queryPercent
	unquantifiedAlts := []AlterationConfiguration{}

	pctClaimed := 0

	for _, altSpec := range endpointSpecList {
		if altSpec.ErrorToReturn == "" && altSpec.OverrideToReturn == "" {
			return nil, status.Error(codes.InvalidArgument, "cannot map alteration to assigned query percentage without specifying either ErrorToReturn or OverrideToReturn for a target endpoint")
		}

		if altSpec.ErrorToReturn != "" && altSpec.OverrideToReturn != "" {
			return nil, status.Error(codes.InvalidArgument, "cannot map alteration to assigned query percentage when ErrorToReturn and OverrideToReturn are both specified for a target endpoint")
		}

		alterationConfig := AlterationConfiguration{
			ErrorToReturn:    altSpec.ErrorToReturn,
			OverrideToReturn: altSpec.OverrideToReturn,
		}

		// Intuition:
		// (1) add all endpoints where queryPercent is specified
		// (2) track total percentage odds of queries already claimed by an AlterationConfiguration
		// (3) track alterations where percentage odds (queryPercent) is not specified (to be calculated later)
		if altSpec.QueryPercent > 0 {
			mapping[alterationConfig] = QueryPercent(altSpec.QueryPercent)
			pctClaimed += int(altSpec.QueryPercent)

			if pctClaimed > 100 {
				return nil, status.Error(codes.InvalidArgument, "assigned query percentages for this endpoint exceeds 100% of possible queries")
			}
		} else {
			unquantifiedAlts = append(unquantifiedAlts, alterationConfig)
		}
	}

	// for alterations where queryPercent were not specified, split the remaining percentage odds equally by alteration
	if len(unquantifiedAlts) > 0 {
		pctUnclaimed := 100 - pctClaimed
		numUnquantifiedAlts := len(unquantifiedAlts)

		if pctUnclaimed < numUnquantifiedAlts {
			return nil, status.Error(codes.InvalidArgument, "alterations must have at least 1% chance of occurring; endpoint has too many alterations configured so its total configurations exceed 100%")
		}

		// find percentage odds to assign to each alteration where queryPercent was not specified
		pctPerAlt := pctUnclaimed / numUnquantifiedAlts

		for i, altConfig := range unquantifiedAlts {
			if i == len(unquantifiedAlts)-1 {
				mapping[altConfig] = QueryPercent(100 - pctClaimed) // grab the remaining
			} else {
				mapping[altConfig] = QueryPercent(pctPerAlt)
				pctClaimed += pctPerAlt
			}
		}
	}

	return mapping, nil
}

// FlattenAlterationMap takes a mapping from alterationConfiguration to the percentage of requests
// and returns a slice where the slice's "index" between 0 and some number less than 100 are assigned
// Alterations which reappear as many times as the requested query percentage
func FlattenAlterationMap(alterationToQueryPercent map[AlterationConfiguration]QueryPercent) []AlterationConfiguration {
	slice := make([]AlterationConfiguration, 0, 100)

	for altConfig, pct := range alterationToQueryPercent {
		for i := 0; i < int(pct); i++ {
			slice = append(slice, altConfig)
		}
	}

	return slice
}
