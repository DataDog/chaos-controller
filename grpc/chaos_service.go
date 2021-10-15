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

	v1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	grpccalc "github.com/DataDog/chaos-controller/grpc/calculations"
	pb "github.com/DataDog/chaos-controller/grpc/disruptionlistener"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
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
func (d *ChaosDisruptionListener) Disrupt(ctx context.Context, ds *pb.DisruptionSpec) (*emptypb.Empty, error) {
	if ds == nil {
		d.logger.Error("cannot execute Disrupt when DisruptionSpec is nil")
		return nil, status.Error(codes.InvalidArgument, "Cannot execute Disrupt when DisruptionSpec is nil")
	}

	config := grpccalc.DisruptionConfiguration{}

	for _, endpointSpec := range ds.Endpoints {
		if endpointSpec.TargetEndpoint == "" {
			d.logger.Error("DisruptionSpec does not specify TargetEndpoint for at least one endpointAlteration")
			return nil, status.Error(codes.InvalidArgument, "Cannot execute Disrupt without specifying TargetEndpoint for all endpointAlterations")
		}

		Alterations, err := grpccalc.ConvertSpecifications(endpointSpec.Alterations)
		if err != nil {
			return nil, err
		}

		// add endpoint to main configuration
		targetEndpoint := grpccalc.TargetEndpoint(endpointSpec.TargetEndpoint)

		config[targetEndpoint] = grpccalc.EndpointConfiguration{
			TargetEndpoint: targetEndpoint,
			Alterations:    Alterations,
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
func (d *ChaosDisruptionListener) ChaosServerInterceptor(ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (response interface{}, err error) {
	d.logger.Debug("comparing with %s with %d endpoints", info.FullMethod, len(d.configuration))

	// FullMethod is the full RPC method string, i.e., /package.service/method.
	targetEndpoint := grpccalc.TargetEndpoint(info.FullMethod)

	if endptConfig, ok := d.configuration[targetEndpoint]; ok {
		randomPercent := rand.Intn(100)

		if len(endptConfig.Alterations) > randomPercent {
			altConfig := endptConfig.Alterations[randomPercent]

			if altConfig.ErrorToReturn != "" {
				d.logger.Debug("error code to return: %s", v1beta1.ErrorMap[altConfig.ErrorToReturn])

				return nil, status.Error(
					v1beta1.ErrorMap[altConfig.ErrorToReturn],
					// Future Work: interview users about this message //nolint:golint
					fmt.Sprintf("Chaos Controller injected this error: %s", altConfig.ErrorToReturn),
				)
			} else if altConfig.OverrideToReturn != "" {
				d.logger.Debug("override to return: %s", altConfig.OverrideToReturn)

				return &emptypb.Empty{}, nil
			}

			d.logger.Error("endpoint %s should define either an ErrorToReturn or OverrideToReturn but does not", endptConfig.TargetEndpoint)
		}
	}

	return handler(ctx, req)
}
