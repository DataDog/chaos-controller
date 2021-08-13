// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package grpc

import (
	"context"
	"log"
	"time"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	pb "github.com/DataDog/chaos-controller/grpc/disruption_listener"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ExecuteSendDisruption takes in a CRD specification for GRPC disruptions and
// executes a SendDisruption call on the provided DisruptionListenerClient
func ExecuteSendDisruption(client pb.DisruptionListenerClient, spec chaosv1beta1.GRPCDisruptionSpec) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	endpointSpecs := GenerateEndpointSpecs(spec.Endpoints)

	_, err := client.SendDisruption(ctx, &pb.DisruptionSpec{Endpoints: endpointSpecs})
	if err != nil {
		log.Printf("Received an error: %v", err)
	}
}

// ExecuteCleanDisruption executes a CleanDisruption call on the provided DisruptionListenerClient
func ExecuteCleanDisruption(client pb.DisruptionListenerClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.CleanDisruption(ctx, &emptypb.Empty{})
	if err != nil {
		log.Printf("Received an error: %v", err)
	}
}

// GenerateEndpointSpecs converts an EndpointAlteration into a list of EndpointSpecs which
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
				TargetEndpoint: string(endptAlt.TargetEndpoint),
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

	endpointSpecs := make([]*pb.EndpointSpec, 0)
	for _, endptSpec := range targetToEndpointSpec {
		endpointSpecs = append(endpointSpecs, endptSpec)
	}

	return endpointSpecs
}
