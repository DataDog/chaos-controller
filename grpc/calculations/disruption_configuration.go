// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package calculations

// DisruptionConfiguration configures the DisruptionListener to chaos test endpoints of a gRPC server.
type DisruptionConfiguration map[TargetEndpoint]EndpointConfiguration

// EndpointConfiguration configures endpoints that the DisruptionListener chaos tests on a gRPC server.
// The Alterations maps integers from 0 to 100 to alteration configurations.
type EndpointConfiguration struct {
	TargetEndpoint TargetEndpoint
	Alterations    []AlterationConfiguration
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
