// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package grpc

import (
	"context"
	"fmt"
	"math/rand"
	"sync"

	pb "github.com/DataDog/chaos-controller/grpc/disruption_listener"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

/*
 * When a random integer from 0 to 100 is randomly selected, the PercentSlotToAlteration mapping is referenced to
 * identify the disruption to apply to a query. The mapping represents user preference the proportion of queries
 * affected by each alteration. See ALGORITHM.md for examples.
 */

// DisruptionListener is a gRPC Service that can disrupt endpoints of a gRPC server.
type DisruptionListener struct {
	pb.UnimplementedDisruptionListenerServer
	Configuration DisruptionConfiguration
}

// DisruptionConfiguration configures the DisruptionListener to chaos test endpoints of a gRPC server.
type DisruptionConfiguration map[TargetEndpoint]EndpointConfiguration

// EndpointConfiguration configures endpoints that the DisruptionListener chaos tests on a gRPC server.
// The AlterationHash maps integers from 0 to 100 to alteration configurations.
type EndpointConfiguration struct {
	TargetEndpoint TargetEndpoint
	Alterations    map[AlterationConfiguration]PercentAffected
	AlterationHash map[PercentSlot]AlterationConfiguration
}

// AlterationConfiguration contains either an ErrorToReturn or an OverrideToReturn for a given
// gRPC query to the disrupted service.
type AlterationConfiguration struct {
	ErrorToReturn    string
	OverrideToReturn string
}

// PercentAffected is an integer represented the number of queries out of 100 queries which should
// be affected by a certain endpoint alteration.
type PercentAffected int

// PercentSlot is an integer used to make a random, weighted assignment to an endpoint based on the
// configured PercentAffected per endpoint in a GrpcDisruptionConfiguration.
type PercentSlot int

// TargetEndpoint is a string of the format /package.service/method.
type TargetEndpoint string

var (
	log *zap.SugaredLogger

	errorMap = map[string]codes.Code{
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

	mutex sync.Mutex
)

// SendDisruption makes a gRPC call to DisruptionListener.
func (d *DisruptionListener) SendDisruption(ctx context.Context, ds *pb.DisruptionSpec) (*emptypb.Empty, error) {
	if ds == nil {
		log.Error("Cannot execute SendDisruption when DisruptionSpec is nil")
		return nil, status.Error(codes.InvalidArgument, "Cannot execute SendDisruption when DisruptionSpec is nil")
	}

	config := DisruptionConfiguration{}

	for _, endpointSpec := range ds.Endpoints {
		targeted := TargetEndpoint(endpointSpec.TargetEndpoint)

		if endpointSpec.TargetEndpoint == "" {
			log.Error("DisruptionSpec does not specify TargetEndpoint for at least one endpointAlteration")
			return nil, status.Error(codes.InvalidArgument, "Cannot execute SendDisruption without specifying TargetEndpoint for all endpointAlterations")
		}

		alterationToPercentAffected, err := GetAlterationToPercentAffected(endpointSpec.Alterations, targeted)
		if err != nil {
			return nil, err
		}

		percentSlotToAlteration, err := GetPercentSlotToAlteration(alterationToPercentAffected)
		if err != nil {
			return nil, err
		}

		// add endpoint to main configuration
		config[targeted] = EndpointConfiguration{
			TargetEndpoint: targeted,
			Alterations:    alterationToPercentAffected,
			AlterationHash: percentSlotToAlteration,
		}
	}

	mutex.Lock()
	d.Configuration = config
	mutex.Unlock()

	return &emptypb.Empty{}, nil
}

// DisruptionStatus lists all configured endpoint alterations for DisruptionListener.
func (d *DisruptionListener) DisruptionStatus(context.Context, *emptypb.Empty) (*pb.DisruptionSpec, error) {
	ds := pb.DisruptionSpec{Endpoints: []*pb.EndpointSpec{}}

	for _, endptConfig := range d.Configuration {
		endptSpec := &pb.EndpointSpec{
			TargetEndpoint: string(endptConfig.TargetEndpoint),
			Alterations:    make([]*pb.AlterationSpec, 0),
		}

		for altConfig, pctAffected := range endptConfig.Alterations {
			altSpec := &pb.AlterationSpec{
				ErrorToReturn:    altConfig.ErrorToReturn,
				OverrideToReturn: altConfig.OverrideToReturn,
				QueryPercent:     int32(pctAffected),
			}

			endptSpec.Alterations = append(endptSpec.Alterations, altSpec)
		}

		ds.Endpoints = append(ds.Endpoints, endptSpec)
	}

	return &ds, nil
}

// CleanDisruption removes all configured endpoint alterations for DisruptionListener.
func (d *DisruptionListener) CleanDisruption(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	mutex.Lock()
	d.Configuration = make(map[TargetEndpoint]EndpointConfiguration)
	mutex.Unlock()

	return &emptypb.Empty{}, nil
}

// ChaosServerInterceptor is a function which can be registered on instantiation of a gRPC server
// to intercept all traffic to the server and crosscheck their endpoints to disrupt them.
func (d *DisruptionListener) ChaosServerInterceptor(ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (response interface{}, err error) {
	log.Debug("comparing with %s with %d endpoints", info.FullMethod, len(d.Configuration))

	// FullMethod is the full RPC method string, i.e., /package.service/method.
	if endptConfig, ok := d.Configuration[TargetEndpoint(info.FullMethod)]; ok {
		if altConfig, ok := endptConfig.AlterationHash[PercentSlot(rand.Intn(100))]; ok {
			if altConfig.ErrorToReturn != "" {
				log.Debug("Error Code: %s", errorMap[altConfig.ErrorToReturn])

				return nil, status.Error(
					errorMap[altConfig.ErrorToReturn],
					// Future Work: interview users about this message
					fmt.Sprintf("Chaos-Controller injected this error: %s", altConfig.ErrorToReturn),
				)
			} else if altConfig.OverrideToReturn != "" {
				log.Debug("OverrideToReturn: %s", altConfig.OverrideToReturn)

				return &emptypb.Empty{}, nil
			}

			log.Error("Endpoint %s should define either an ErrorToReturn or OverrideToReturn but does not", endptConfig.TargetEndpoint)
		}
	}

	return handler(ctx, req)
}

// GetAlterationToPercentAffected takes a series of alterations configured for a Spec and maps them to the percentage of queries which should be affected
func GetAlterationToPercentAffected(endpointSpecList []*pb.AlterationSpec, targeted TargetEndpoint) (map[AlterationConfiguration]PercentAffected, error) {
	// object returned indicates, for a particular AlterationConfiguration, what percentage of queries to which it should apply
	mapping := make(map[AlterationConfiguration]PercentAffected)

	// unquantified is a holding variable used later to calculate and assign percentages to alterations which do not specify queryPercent
	unquantifiedAlts := make([]AlterationConfiguration, 0)

	pctClaimed := 0

	for _, altSpec := range endpointSpecList {
		if altSpec.ErrorToReturn == "" && altSpec.OverrideToReturn == "" {
			log.Error("For endpoint %s, neither ErrorToReturn nor OverrideToReturn are specified which is not allowed", targeted)
			return nil, status.Error(codes.InvalidArgument, "Cannot execute SendDisruption without specifying either ErrorToReturn or OverrideToReturn for all target endpoints")
		}

		if altSpec.ErrorToReturn != "" && altSpec.OverrideToReturn != "" {
			log.Error("For endpoint %s, both ErrorToReturn and OverrideToReturn are specified which is not allowed", targeted)
			return nil, status.Error(codes.InvalidArgument, "Cannot execute SendDisruption where ErrorToReturn or OverrideToReturn are both specified for a target endpoints")
		}

		alterationConfig := AlterationConfiguration{
			ErrorToReturn:    altSpec.ErrorToReturn,
			OverrideToReturn: altSpec.OverrideToReturn,
		}

		// Intuition:
		// (1) add all endpoints where queryPercent is specified
		// (2) track percentage of queries already claimed by an endpointConfiguration
		// (3) track endpoints where queryPercent is missing and calculate them later
		if altSpec.QueryPercent > 0 {
			mapping[alterationConfig] = PercentAffected(altSpec.QueryPercent)
			pctClaimed += int(altSpec.QueryPercent)
		} else {
			unquantifiedAlts = append(unquantifiedAlts, alterationConfig)
		}
	}

	if len(unquantifiedAlts) > 0 {
		if pctClaimed == 100 {
			log.Info("Alterations with specified percentQuery sum to cover all of the queries; "+
				"%d alterations that do not specify queryPercent will not get applied at all for endpoint %s",
				len(unquantifiedAlts), targeted,
			)
		}

		// add all endpoints where queryPercent is not specified by splitting the remaining queries equally by alteration
		pctPerAlt := (100 - pctClaimed) / len(unquantifiedAlts)
		if pctPerAlt < 1 {
			log.Info("Alterations with specified percentQuery sum to cover almost all queries; "+
				"%d alterations that do not specify queryPercent will not get applied at all for endpoint %s",
				len(unquantifiedAlts)-1, targeted,
			)
		}

		for i, altConfig := range unquantifiedAlts {
			if i == len(unquantifiedAlts)-1 {
				mapping[altConfig] = PercentAffected(100 - pctClaimed) // grab the remaining
			} else {
				mapping[altConfig] = PercentAffected(pctPerAlt)
				pctClaimed += pctPerAlt
			}
		}
	}

	return mapping, nil
}

// GetPercentSlotToAlteration Converts a mapping from alterationConfiguration to the percentage of requests which should return this altered response
func GetPercentSlotToAlteration(endpointSpecList map[AlterationConfiguration]PercentAffected) (map[PercentSlot]AlterationConfiguration, error) {
	hashMap := make(map[PercentSlot]AlterationConfiguration)
	currPct := 0

	log.Debug("configuring percentile map")

	for altConfig, pct := range endpointSpecList {
		for i := 0; i < int(pct); i++ {
			hashMap[PercentSlot(currPct)] = altConfig

			// log as we go
			if altConfig.ErrorToReturn != "" {
				log.Debug("%d: %s", currPct, altConfig.ErrorToReturn)
			} else {
				log.Debug("%d: %s", currPct, altConfig.OverrideToReturn)
			}
			currPct++
		}
	}

	return hashMap, nil
}
