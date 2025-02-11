// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package v1beta1

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/go-multierror"
	"google.golang.org/grpc/codes"
)

// ERROR represents the type of gRPC alteration where a response is spoofed with a gRPC error code
const ERROR = "error"

// OVERRIDE represents the type of gRPC alteration where a response is spoofed with a specified return value
const OVERRIDE = "override"

// ErrorMap is a mapping from string representation of gRPC error to the official error code
var ErrorMap = map[string]codes.Code{
	"OK":                  codes.OK,
	"CANCELED":            codes.Canceled,
	"UNKNOWN":             codes.Unknown,
	"INVALID_ARGUMENT":    codes.InvalidArgument,
	"DEADLINE_EXCEEDED":   codes.DeadlineExceeded,
	"NOT_FOUND":           codes.NotFound,
	"ALREADY_EXISTS":      codes.AlreadyExists,
	"PERMISSION_DENIED":   codes.PermissionDenied,
	"RESOURCE_EXHAUSTED":  codes.ResourceExhausted,
	"FAILED_PRECONDITION": codes.FailedPrecondition,
	"ABORTED":             codes.Aborted,
	"OUT_OF_RANGE":        codes.OutOfRange,
	"UNIMPLEMENTED":       codes.Unimplemented,
	"INTERNAL":            codes.Internal,
	"UNAVAILABLE":         codes.Unavailable,
	"DATA_LOSS":           codes.DataLoss,
	"UNAUTHENTICATED":     codes.Unauthenticated,
}

// GRPCDisruptionSpec represents a gRPC disruption
type GRPCDisruptionSpec struct {
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int `json:"port" chaos_validate:"gte=1,lte=65535"`
	// +kubebuilder:validation:Required
	Endpoints []EndpointAlteration `json:"endpoints" chaos_validate:"required,dive"`
}

// EndpointAlteration represents an endpoint to disrupt and the corresponding error to return
type EndpointAlteration struct {
	// +kubebuilder:validation:Required
	TargetEndpoint string `json:"endpoint" chaos_validate:"required"`
	// +kubebuilder:validation:Enum=OK;CANCELED;UNKNOWN;INVALID_ARGUMENT;DEADLINE_EXCEEDED;NOT_FOUND;ALREADY_EXISTS;PERMISSION_DENIED;RESOURCE_EXHAUSTED;FAILED_PRECONDITION;ABORTED;OUT_OF_RANGE;UNIMPLEMENTED;INTERNAL;UNAVAILABLE;DATA_LOSS;UNAUTHENTICATED
	ErrorToReturn string `json:"error,omitempty" chaos_validate:"omitempty,oneofci=OK CANCELED UNKNOWN INVALID_ARGUMENT DEADLINE_EXCEEDED NOT_FOUND ALREADY_EXISTS PERMISSION_DENIED RESOURCE_EXHAUSTED FAILED_PRECONDITION ABORTED OUT_OF_RANGE UNIMPLEMENTED INTERNAL UNAVAILABLE DATA_LOSS UNAUTHENTICATED"`
	// +kubebuilder:validation:Enum={}
	OverrideToReturn string `json:"override,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	QueryPercent int `json:"queryPercent,omitempty" chaos_validate:"omitempty,gte=0,lte=100"`
}

// Validate validates that all alterations have either an error or override to return and at least 1% chance of occurring,
// as well as that the sum of query percentages of all alterations assigned to a target endpoint do not exceed 100%
func (s GRPCDisruptionSpec) Validate() (retErr error) {
	queryPctByEndpoint := map[string]int{}
	unquantifiedAlts := map[string]int{}

	for _, alteration := range s.Endpoints {
		if alteration.QueryPercent == 0 {
			if count, ok := unquantifiedAlts[alteration.TargetEndpoint]; ok {
				unquantifiedAlts[alteration.TargetEndpoint] = count + 1

				pctClaimed := 100 - queryPctByEndpoint[alteration.TargetEndpoint]

				if pctClaimed < count+1 {
					retErr = multierror.Append(retErr, fmt.Errorf("alterations must have at least 1%% chance of occurring; %s will never return some alterations because alterations exceed 100%% of possible queries", alteration.TargetEndpoint))
				}
			} else {
				unquantifiedAlts[alteration.TargetEndpoint] = 1
			}
		} else {
			// check that endpoint is not already configured such that the sum of the queryPercents total to more than 100%
			if totalQueryPercent, ok := queryPctByEndpoint[alteration.TargetEndpoint]; ok {
				// always positive because of CRD limitations
				queryPctByEndpoint[alteration.TargetEndpoint] = totalQueryPercent + alteration.QueryPercent
				if queryPctByEndpoint[alteration.TargetEndpoint] > 100 {
					retErr = multierror.Append(retErr, fmt.Errorf("total queryPercent of all alterations applied to endpoint %s is over 100%%", alteration.TargetEndpoint))
				}
			} else {
				queryPctByEndpoint[alteration.TargetEndpoint] = alteration.QueryPercent
			}
		}

		// check that exactly one of ErrorToReturn or OverrideToReturn is configured
		if (alteration.ErrorToReturn == "" && alteration.OverrideToReturn == "") || (alteration.ErrorToReturn != "" && alteration.OverrideToReturn != "") {
			retErr = multierror.Append(retErr, fmt.Errorf("the gRPC disruption must have exactly one of ErrorToReturn or OverrideToReturn specified for endpoint %s", alteration.TargetEndpoint))
		}
	}

	return multierror.Prefix(retErr, "GRPC:")
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
			alterationType = ERROR
			alterationValue = endptAlt.ErrorToReturn
		}

		if endptAlt.OverrideToReturn != "" {
			alterationType = OVERRIDE
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

	// Each value passed to --endpoint-alterations should be of the form
	// `endpoint;alteration_type;alteration_value;optional_query_percent`
	// e.g.
	// `/chaosdogfood.ChaosDogfood/order;error;ALREADY_EXISTS;30`
	// `/chaosdogfood.ChaosDogfood/order;override;{};`
	args = append(args, "--endpoint-alterations")
	args = append(args, strings.Split(strings.Join(endpointAlterationArgs, " --endpoint-alterations "), " ")...)

	return args
}

func (s GRPCDisruptionSpec) Explain() []string {
	explanation := []string{
		"spec.grpc will activate the chaos interceptor on the targeted grpc server, replacing responses with specified errors.",
		"This disruption type can only work on grpc servers, not on the clients, " +
			"and requires [the server to cooperate by installing the chaos interceptor]" +
			"(https://github.com/DataDog/chaos-controller/blob/main/docs/grpc_disruption/instructions.md)",
		"The following endpoints will be intercepted:",
	}

	for _, endpt := range s.Endpoints {
		var spoof string

		if endpt.ErrorToReturn != "" {
			spoof = endpt.ErrorToReturn
		} else {
			spoof = endpt.OverrideToReturn
		}

		explanation = append(explanation, fmt.Sprintf("The endpoint %s will return %s %d%% of the time", endpt.TargetEndpoint, spoof, endpt.QueryPercent))
	}
	return explanation
}
