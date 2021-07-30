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

	var endpointAlterationList []*pb.EndpointAlteration
	for _, altSpec := range spec.Endpoints {
		endpointAlteration := pb.EndpointAlteration{
			TargetEndpoint:   altSpec.TargetEndpoint,
			ErrorToReturn:    altSpec.ErrorToReturn,
			OverrideToReturn: altSpec.OverrideToReturn,
		}
		endpointAlterationList = append(endpointAlterationList, &endpointAlteration)
	}

	_, err := client.SendDisruption(
		ctx,
		&pb.DisruptionSpec{EndpointAlterations: endpointAlterationList},
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
