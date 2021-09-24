// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package calculations

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/DataDog/chaos-controller/grpc/disruption_listener"
)

// DisruptionConfiguration configures the DisruptionListener to chaos test endpoints of a gRPC server.
type DisruptionConfiguration map[TargetEndpoint]EndpointConfiguration

// EndpointConfiguration configures endpoints that the DisruptionListener chaos tests on a gRPC server.
// The AlterationMap maps integers from 0 to 100 to alteration configurations.
type EndpointConfiguration struct {
	TargetEndpoint TargetEndpoint
	AlterationMap  []AlterationConfiguration
}

// AlterationConfiguration contains either an ErrorToReturn or an OverrideToReturn for a given
// gRPC query to the disrupted service.
type AlterationConfiguration struct {
	ErrorToReturn    string
	OverrideToReturn string
}

// QueryPercent is an integer representing the percentage odds that a query for an endpoint is affected by a certain alteration.
type QueryPercent int

// TargetEndpoint is a string of the format /package.service/method.
type TargetEndpoint string

// GetAlterationMapFromAlterationSpec takes a series of alterations configured for a target endpoint where
// assignments are distributed based on percentage odds (QueryPercent) expected for different return alterations and
// returns a mapping from integers between 0 and some number less than 100 to the Alterations assigned to them
func GetAlterationMapFromAlterationSpec(endpointSpecList []*pb.AlterationSpec) ([]AlterationConfiguration, error) {
	alterationToQueryPercent, err := ConvertAltSpecToQueryPercentByAltConfig(endpointSpecList)
	if err != nil {
		return nil, err
	}

	return ConvertQueryPercentByAltConfigToAlterationMap(alterationToQueryPercent), nil
}

// ConvertAltSpecToQueryPercentByAltConfig takes a series of alterations configured for a target endpoint
// and maps them to the percentage of queries which will be altered by it
func ConvertAltSpecToQueryPercentByAltConfig(endpointSpecList []*pb.AlterationSpec) (map[AlterationConfiguration]QueryPercent, error) {
	// object returned indicates, for a particular AlterationConfiguration, what percentage of queries to which it should apply
	mapping := make(map[AlterationConfiguration]QueryPercent)

	// unquantified is a holding variable used later to calculate and assign percentages to alterations which do not specify queryPercent
	unquantifiedAlts := make([]AlterationConfiguration, 0)

	pctClaimed := 0

	for _, altSpec := range endpointSpecList {
		if altSpec.ErrorToReturn == "" && altSpec.OverrideToReturn == "" {
			return nil, status.Error(codes.InvalidArgument, "cannot map alteration to assigned percentage without specifying either ErrorToReturn or OverrideToReturn for a target endpoint")
		}

		if altSpec.ErrorToReturn != "" && altSpec.OverrideToReturn != "" {
			return nil, status.Error(codes.InvalidArgument, "cannot map alteration to assigned percentage when ErrorToReturn and OverrideToReturn are both specified for a target endpoint")
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
				return nil, status.Error(codes.InvalidArgument, "assigned percentage for this endpoint exceeds 100% of possible queries")
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

// ConvertQueryPercentByAltConfigToAlterationMap takes a mapping from alterationConfiguration to the percentage of requests
// and returns a mapping from integers between 0 and some number less than 100 to Alterations assigned to them
func ConvertQueryPercentByAltConfigToAlterationMap(endpointSpecList map[AlterationConfiguration]QueryPercent) []AlterationConfiguration {
	mapping := make([]AlterationConfiguration, 0, 100)

	for altConfig, pct := range endpointSpecList {
		for i := 0; i < int(pct); i++ {
			mapping = append(mapping, altConfig)
		}
	}

	return mapping
}
