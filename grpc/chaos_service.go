// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package grpc

import (
	"context"
	"fmt"
	"math/rand"
	"sync"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	grpccalc "github.com/DataDog/chaos-controller/grpc/calculations"
	pb "github.com/DataDog/chaos-controller/grpc/disruptionlistener"
)

// ChaosDisruptionListener is a gRPC Service that can disrupt endpoints of a gRPC server.
// The interface it is implementing was generated in the grpc/disruptionlistener package.
type ChaosDisruptionListener struct {
	pb.UnimplementedDisruptionListenerServer
	configuration grpccalc.DisruptionConfiguration
	mutex         sync.Mutex
	logger        *zap.SugaredLogger
}

// NewDisruptionListener creates a new DisruptionListener Service with the logger instantiated and DisruptionConfiguration set to be empty
func NewDisruptionListener(logger *zap.SugaredLogger) *ChaosDisruptionListener {
	d := ChaosDisruptionListener{}

	d.logger = logger
	d.configuration = grpccalc.DisruptionConfiguration{}

	return &d
}

// Disrupt receives a disruption specification and configures the interceptor to spoof responses to specified endpoints.
func (d *ChaosDisruptionListener) Disrupt(ctx context.Context, disruptionSpec *pb.DisruptionSpec) (*emptypb.Empty, error) {
	if disruptionSpec == nil {
		d.logger.Error("cannot execute Disrupt when DisruptionSpec is nil")
		return nil, status.Error(codes.InvalidArgument, "cannot execute Disrupt when DisruptionSpec is nil")
	}

	d.logger.Debugw("launching interceptor", "nb_endpoints", len(disruptionSpec.GetEndpoints()))

	config := grpccalc.DisruptionConfiguration{}

	// from list of endpoints and alterations, build definitive list of alterations
	for _, endpointSpec := range disruptionSpec.Endpoints {
		if endpointSpec.TargetEndpoint == "" {
			d.logger.Error("disruptionSpec does not specify TargetEndpoint for at least one endpointAlteration")
			return nil, status.Error(codes.InvalidArgument, "cannot execute Disrupt without specifying TargetEndpoint for all endpointAlterations")
		}

		// build array of alterations based on queryPercent
		// basically each alteration appears in the array a number of times equal to its QueryPercent
		// this array contains 100 elements max
		alterations, err := grpccalc.ConvertSpecifications(endpointSpec.Alterations)
		if err != nil {
			return nil, err
		}

		// add endpoint to main configuration
		targetEndpoint := grpccalc.TargetEndpoint(endpointSpec.TargetEndpoint)

		d.logger.Debugw("adding endpoint",
			"target_endpoint", targetEndpoint,
			"nb_alterations", len(alterations),
		)

		config[targetEndpoint] = grpccalc.EndpointConfiguration{
			TargetEndpoint: targetEndpoint,
			Alterations:    alterations,
		}
	}

	if len(d.configuration) > 0 {
		d.logger.Error("cannot apply new DisruptionSpec when DisruptionListener is already configured")
		return nil, status.Error(codes.AlreadyExists, "Cannot apply new DisruptionSpec when DisruptionListener is already configured")
	}

	d.mutex.Lock()

	select {
	case <-ctx.Done():
		d.logger.Error("cannot apply new DisruptionSpec, gRPC request was canceled")
	default:
		d.configuration = config
	}

	d.mutex.Unlock()

	return &emptypb.Empty{}, nil
}

// ResetDisruptions removes all configured endpoint alterations for DisruptionListener.
func (d *ChaosDisruptionListener) ResetDisruptions(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	d.mutex.Lock()
	d.configuration = map[grpccalc.TargetEndpoint]grpccalc.EndpointConfiguration{}
	d.mutex.Unlock()

	return &emptypb.Empty{}, nil
}

// ChaosServerInterceptor is a function which can be registered on instantiation of a gRPC server
// to intercept all traffic to the server and crosscheck their endpoints to disrupt them.
func (d *ChaosDisruptionListener) ChaosServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (response interface{}, err error) {
	d.logger.Debugw("intercept method with specified alterations", "grpc_method", info.FullMethod, "nb_alterations", len(d.configuration))

	targetEndpoint := grpccalc.TargetEndpoint(info.FullMethod)

	endpointConfiguration, ok := d.configuration[targetEndpoint]
	if !ok {
		d.logger.Debugw("endpoint is not configured. Skipping",
			"grpc_method", info.FullMethod,
			"target_endpoint", targetEndpoint,
		)

		return handler(ctx, req)
	}

	// we pick a random index to determine which alteration to apply
	// this index is between 0 and 99
	// the alteration array contains as much elements as the sum of all queryPercent (max 100)
	randomAlterationIndex := rand.Intn(100)
	if randomAlterationIndex >= len(endpointConfiguration.Alterations) {
		d.logger.Debugw("index picked is out of bounds. Skipping",
			"grpc_method", info.FullMethod,
			"target_endpoint", targetEndpoint,
			"alteration_index", randomAlterationIndex,
			"nb_alterations", len(endpointConfiguration.Alterations),
		)

		return handler(ctx, req)
	}

	alteration := endpointConfiguration.Alterations[randomAlterationIndex]

	d.logger.Debugw("endpoint is configured. Picking which alteration to apply",
		"grpc_method", info.FullMethod,
		"target_endpoint", targetEndpoint,
		"alteration_error", alteration.ErrorToReturn,
		"alteration_override", alteration.OverrideToReturn,
	)

	if alteration.ErrorToReturn != "" {
		d.logger.Debugw("error to return found. Injecting error",
			"grpc_method", info.FullMethod,
			"target_endpoint", targetEndpoint,
			"error", v1beta1.ErrorMap[alteration.ErrorToReturn])

		return nil, status.Error(
			v1beta1.ErrorMap[alteration.ErrorToReturn],
			// Future Work: interview users about this message
			fmt.Sprintf("Chaos Controller injected this error: %s", alteration.ErrorToReturn),
		)
	} else if alteration.OverrideToReturn != "" {
		d.logger.Debugw("override to return found. Injecting override",
			"grpc_method", info.FullMethod,
			"target_endpoint", targetEndpoint,
			"override", alteration.OverrideToReturn)

		return &emptypb.Empty{}, nil
	}

	d.logger.Errorw("endpoint should define either an ErrorToReturn or OverrideToReturn but does not. Skipping",
		"grpc_method", info.FullMethod,
		"target_endpoint", targetEndpoint,
		"alteration_error", alteration.ErrorToReturn,
		"alteration_override", alteration.OverrideToReturn,
	)

	return handler(ctx, req)
}
