package grpc

import (
	"context"
	"log"
	"time"

	pb "github.com/DataDog/chaos-controller/grpc/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

func ExecuteSendDisruption(client pb.DisruptionListenerClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	endpointAlteration := pb.EndpointAlteration{TargetEndpoint: "/chaos_dogfood.ChaosDogfood/order", ErrorToReturn: "InvalidRequest"}
	endpointAlterationList := []*pb.EndpointAlteration{&endpointAlteration}

	_, err := client.SendDisruption(ctx, &pb.DisruptionSpec{EndpointAlterations: endpointAlterationList})

	if err != nil {
		log.Printf("Received an error: %v", err)
	}
}

func ExecuteCleanDisruption(client pb.DisruptionListenerClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := client.CleanDisruption(ctx, &emptypb.Empty{})
	if err != nil {
		log.Printf("Received an error: %v", err)
	}
}
