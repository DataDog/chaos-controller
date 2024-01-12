// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package grpc

import (
	"context"
	"time"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	pb "github.com/DataDog/chaos-controller/grpc/disruptionlistener"
	"google.golang.org/protobuf/types/known/emptypb"
)

// SendGrpcDisruption takes in a CRD specification for GRPC disruptions and
// executes a Disrupt call on the provided DisruptionListenerClient
func SendGrpcDisruption(client pb.DisruptionListenerClient, spec chaosv1beta1.GRPCDisruptionSpec) error {
	endpointSpecs := GenerateEndpointSpecs(spec.Endpoints)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Disrupt(ctx, &pb.DisruptionSpec{Endpoints: endpointSpecs})

	return err
}

// ClearGrpcDisruptions executes a ResetDisruptions call on the provided DisruptionListenerClient
func ClearGrpcDisruptions(client pb.DisruptionListenerClient) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.ResetDisruptions(ctx, &emptypb.Empty{})

	return err
}

// GenerateEndpointSpecs converts a slice of EndpointAlterations into a slice of EndpointSpecs which
// can be sent through gRPC call to disruptionListener
func GenerateEndpointSpecs(endpoints []chaosv1beta1.EndpointAlteration) []*pb.EndpointSpec {
	targetToEndpointSpec := make(map[string]*pb.EndpointSpec)

	for _, endptAlt := range endpoints {
		targeted := endptAlt.TargetEndpoint

		if existingEndptSpec, ok := targetToEndpointSpec[targeted]; ok {
			altSpec := &pb.AlterationSpec{
				ErrorToReturn:    endptAlt.ErrorToReturn,
				OverrideToReturn: endptAlt.OverrideToReturn,
				QueryPercent:     int32(endptAlt.QueryPercent),
			}
			existingEndptSpec.Alterations = append(existingEndptSpec.Alterations, altSpec)
		} else {
			targetToEndpointSpec[targeted] = &pb.EndpointSpec{
				TargetEndpoint: targeted,
				Alterations: []*pb.AlterationSpec{
					{
						ErrorToReturn:    endptAlt.ErrorToReturn,
						OverrideToReturn: endptAlt.OverrideToReturn,
						QueryPercent:     int32(endptAlt.QueryPercent),
					},
				},
			}
		}
	}

	endpointSpecs := []*pb.EndpointSpec{}
	for _, endptSpec := range targetToEndpointSpec {
		endpointSpecs = append(endpointSpecs, endptSpec)
	}

	return endpointSpecs
}
