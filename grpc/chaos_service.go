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
	grpc_calcapi "github.com/DataDog/chaos-controller/grpc/calculations"
	pb "github.com/DataDog/chaos-controller/grpc/disruptionlistener"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// DisruptionListener is a gRPC Service that can disrupt endpoints of a gRPC server.
// The interface it is implementing was generated in the grpc/disruptionlistener package.
type ChaosDisruptionListener struct {
	pb.UnimplementedDisruptionListenerServer
	Configuration grpc_calcapi.DisruptionConfiguration
	Logger        *zap.SugaredLogger
}

var mutex sync.Mutex

// NewDisruptionListener creates a new DisruptionListener Service with the logger instantiated and DisruptionConfiguration set to be empty
func NewDisruptionListener(logger *zap.SugaredLogger) *ChaosDisruptionListener {
	d := ChaosDisruptionListener{}

	d.Logger = logger
	d.Configuration = grpc_calcapi.DisruptionConfiguration{}

	return &d
}

// SendDisruption receives a disruption specification and configures the interceptor to spoof responses to specified endpoints.
func (d *ChaosDisruptionListener) SendDisruption(ctx context.Context, ds *pb.DisruptionSpec) (*emptypb.Empty, error) {
	if ds == nil {
		d.Logger.Error("cannot execute SendDisruption when DisruptionSpec is nil")
		return nil, status.Error(codes.InvalidArgument, "Cannot execute SendDisruption when DisruptionSpec is nil")
	}

	config := grpc_calcapi.DisruptionConfiguration{}

	for _, endpointSpec := range ds.Endpoints {
		if endpointSpec.TargetEndpoint == "" {
			d.Logger.Error("DisruptionSpec does not specify TargetEndpoint for at least one endpointAlteration")
			return nil, status.Error(codes.InvalidArgument, "Cannot execute SendDisruption without specifying TargetEndpoint for all endpointAlterations")
		}

		alterationMap, err := grpc_calcapi.FlattenAlterationSpec(endpointSpec.Alterations)
		if err != nil {
			return nil, err
		}

		// add endpoint to main configuration
		targetEndpoint := grpc_calcapi.TargetEndpoint(endpointSpec.TargetEndpoint)

		config[targetEndpoint] = grpc_calcapi.EndpointConfiguration{
			TargetEndpoint: targetEndpoint,
			AlterationMap:  alterationMap,
		}
	}

	mutex.Lock()

	if len(d.Configuration) > 0 {
		d.Logger.Error("cannot apply new DisruptionSpec when DisruptionListener is already configured")
		return nil, status.Error(codes.AlreadyExists, "Cannot apply new DisruptionSpec when DisruptionListener is already configured")
	}

	select {
	case <-ctx.Done():
		d.Logger.Error("cannot apply new DisruptionSpec, gRPC request was canceled")
	default:
		d.Configuration = config
	}

	mutex.Unlock()

	return &emptypb.Empty{}, nil
}

// CleanDisruption removes all configured endpoint alterations for DisruptionListener.
func (d *ChaosDisruptionListener) CleanDisruption(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	mutex.Lock()
	d.Configuration = make(map[grpc_calcapi.TargetEndpoint]grpc_calcapi.EndpointConfiguration)
	mutex.Unlock()

	return &emptypb.Empty{}, nil
}

// ChaosServerInterceptor is a function which can be registered on instantiation of a gRPC server
// to intercept all traffic to the server and crosscheck their endpoints to disrupt them.
func (d *ChaosDisruptionListener) ChaosServerInterceptor(ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (response interface{}, err error) {
	d.Logger.Debug("comparing with %s with %d endpoints", info.FullMethod, len(d.Configuration))

	// FullMethod is the full RPC method string, i.e., /package.service/method.
	targetEndpoint := grpc_calcapi.TargetEndpoint(info.FullMethod)

	if endptConfig, ok := d.Configuration[targetEndpoint]; ok {
		randomPercent := rand.Intn(100)

		if len(endptConfig.AlterationMap) > randomPercent {
			altConfig := endptConfig.AlterationMap[randomPercent]

			if altConfig.ErrorToReturn != "" {
				d.Logger.Debug("error code to return: %s", v1beta1.ErrorMap[altConfig.ErrorToReturn])

				return nil, status.Error(
					v1beta1.ErrorMap[altConfig.ErrorToReturn],
					// Future Work: interview users about this message //nolint:golint
					fmt.Sprintf("Chaos Controller injected this error: %s", altConfig.ErrorToReturn),
				)
			} else if altConfig.OverrideToReturn != "" {
				d.Logger.Debug("override to return: %s", altConfig.OverrideToReturn)

				return &emptypb.Empty{}, nil
			}

			d.Logger.Error("endpoint %s should define either an ErrorToReturn or OverrideToReturn but does not", endptConfig.TargetEndpoint)
		}
	}

	return handler(ctx, req)
}
