// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package v1beta1

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// GRPCDisruptionSpec represents a gRPC disruption
type GRPCDisruptionSpec struct {
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port      int                  `json:"port,omitempty"`
	Endpoints []EndpointAlteration `json:"endpoints,omitempty"`
}

// EndpointAlteration represents an endpoint to disrupt and the corresponding error to return
type EndpointAlteration struct {
	TargetEndpoint string `json:"endpoint,omitempty"`
	// +kubebuilder:validation:Enum=OK;CANCELED;UNKNOWN;INVALID_ARGUMENT;DEADLINE_EXCEEDED;NOT_FOUND;ALREADY_EXISTS;PERMISSION_DENIED;RESOURCE_EXHAUSTED;FAILED_PRECONDITION;ABORTED;OUT_OF_RANGE;UNIMPLEMENTED;INTERNAL;UNAVAILABLE;DATALOSS;UNAUTHENTICATED
	ErrorToReturn string `json:"error,omitempty"`
	// +kubebuilder:validation:Enum={}
	OverrideToReturn string `json:"override,omitempty"`
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	QueryPercent int `json:"query_pct,omitempty"`
}

// Validate validates that there are no missing hostnames or records for the given grpc disruption spec
func (s GRPCDisruptionSpec) Validate() error {
	if s.Port == 0 {
		return errors.New("some list items in gRPC disruption are missing endpoints; specify an endpoint for each item in the list")
	}

	if len(s.Endpoints) == 0 {
		return errors.New("the gRPC disruption was selected with no endpoints specified, but endpoints must be specified")
	}

	queryPctByEndpoint := make(map[string]int)

	for _, alteration := range s.Endpoints {
		if alteration.TargetEndpoint == "" {
			return errors.New("some list items in gRPC disruption are missing endpoints; specify an endpoint for each item in the list")
		}

		// check that endpoint is not already configured such that the sum of mangled queryPercents total to more than 100%
		if totalQueryPercent, ok := queryPctByEndpoint[alteration.TargetEndpoint]; ok {
			if alteration.QueryPercent > 0 {
				if totalQueryPercent+alteration.QueryPercent >= 100 {
					return errors.New("total queryPercent of all alterations applied to endpoint %s is over 100%; modify them to so their total is 100% or less")
				}
				queryPctByEndpoint[alteration.TargetEndpoint] += alteration.QueryPercent
			}
		}

		// check that exactly one of ErrorToReturn or OverrideToReturn is configured
		if alteration.ErrorToReturn != "" && alteration.OverrideToReturn != "" {
			return fmt.Errorf("the gRPC disruption has ErrorToReturn and OverrideToReturn specified for endpoint %s, but it can only have one", alteration.TargetEndpoint)
		} else if alteration.ErrorToReturn == "" && alteration.OverrideToReturn == "" {
			return fmt.Errorf("the gRPC disruption must have either ErrorToReturn or OverrideToReturn specified for endpoint %s", alteration.TargetEndpoint)
		}
	}
	return nil
}

// GenerateArgs generates injection pod arguments for the given spec
func (s GRPCDisruptionSpec) GenerateArgs() []string {
	args := []string{
		"grpc-disruption",
	}

	endpointAlterationArgs := []string{}

	for _, endptAlt := range s.Endpoints {
		var alterationType, alterationValue string
		if endptAlt.ErrorToReturn != "" {
			alterationType = "error"
			alterationValue = endptAlt.ErrorToReturn
		}
		if endptAlt.OverrideToReturn != "" {
			alterationType = "override"
			alterationValue = endptAlt.OverrideToReturn
		}

		arg := fmt.Sprintf(
			"%s;%s;%s;%s",
			endptAlt.TargetEndpoint,
			alterationType,
			alterationValue,
			strconv.Itoa(endptAlt.QueryPercent),
		)

		endpointAlterationArgs = append(endpointAlterationArgs, arg)
	}

	args = append(args, []string{"--port", strconv.Itoa(s.Port)}...)

	// Each value passed to --host-record-pairs should be of the form
	// `endpoint;alteration_type;alteration_value;optional_query_percent`
	// e.g.
	// `/chaos_dogfood.ChaosDogfood/order;error;ALREADY_EXISTS;30`
	// `/chaos_dogfood.ChaosDogfood/order;override;{};`
	args = append(args, "--endpoint-alterations")
	args = append(args, strings.Split(strings.Join(endpointAlterationArgs, " --endpoint-alterations "), " ")...)

	return args
}
