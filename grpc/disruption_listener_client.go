package grpc

import (
	"context"
	"log"
	"time"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	pb "github.com/DataDog/chaos-controller/grpc/disruption_listener"
	"google.golang.org/protobuf/types/known/emptypb"
)

/*

message DisruptionSpec {
  repeated EndpointSpec endpoints = 1;
}

message EndpointSpec {
  string targetEndpoint = 1;
  repeated AlterationSpec alterations = 2;
}

message AlterationSpec {
  string errorToReturn = 1;
  string overrideToReturn = 2;
  int32 queryPercent = 3;
}

*/

/*
	endptSpec := &pb.EndpointSpec{
		TargetEndpoint: string(endptConfig.TargetEndpoint),
		Alterations: make([]*pb.AlterationSpec, 0),
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
*/
// ExecuteSendDisruption takes in a CRD specification for GRPC disruptions and
// executes a SendDisruption call on the provided DisruptionListenerClient
func ExecuteSendDisruption(client pb.DisruptionListenerClient, spec chaosv1beta1.GRPCDisruptionSpec) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	targetToEndpointSpec := make(map[string]*pb.EndpointSpec)

	for _, endptAlt := range spec.Endpoints {
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

	_, err := client.SendDisruption(
		ctx,
		&pb.DisruptionSpec{Endpoints: endpointSpecs},
	)

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
