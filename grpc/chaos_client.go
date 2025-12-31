// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package grpc

import (
	"context"
	"time"

	"google.golang.org/protobuf/types/known/emptypb"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	pb "github.com/DataDog/chaos-controller/grpc/disruptionlistener"
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
	endpointAlterationMap := make(map[string]*pb.EndpointSpec, len(endpoints))

	// a user can set one endpoint to multiple alterations. We transform the list of alterations into a map of endpoints to alterations
	for _, endpoint := range endpoints {
		targetedEndpoint := endpoint.TargetEndpoint
		alteredSpec := &pb.AlterationSpec{
			ErrorToReturn:    endpoint.ErrorToReturn,
			OverrideToReturn: endpoint.OverrideToReturn,
			QueryPercent:     int32(endpoint.QueryPercent),
		}

		// if endpoint already exists in the list, we append to the alterations
		if foundEndpoint, ok := endpointAlterationMap[targetedEndpoint]; ok {
			foundEndpoint.Alterations = append(foundEndpoint.Alterations, alteredSpec)
		} else {
			endpointAlterationMap[targetedEndpoint] = &pb.EndpointSpec{
				TargetEndpoint: targetedEndpoint,
				Alterations: []*pb.AlterationSpec{
					alteredSpec,
				},
			}
		}
	}

	var endpointSpecs []*pb.EndpointSpec
	for _, endpointSpec := range endpointAlterationMap {
		endpointSpecs = append(endpointSpecs, endpointSpec)
	}

	return endpointSpecs
}
