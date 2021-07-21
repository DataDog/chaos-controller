package grpc

import (
	"context"
	"fmt"
	"log"

	pb "github.com/DataDog/chaos-controller/grpc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type DisruptionListener struct {
	pb.UnimplementedDisruptionListenerServer
	Spec GrpcDisruptionSpec
}

type GrpcDisruptionSpec []EndpointAlteration

type EndpointAlteration struct {
	TargetEndpoint   string
	ErrorToReturn    string
	OverrideToReturn string
}

func (d *DisruptionListener) SendDisruption(ctx context.Context, ds *pb.DisruptionSpec) (*emptypb.Empty, error) {
	if ds == nil {
		//TODO: return error
	}

	for _, alt := range ds.EndpointAlterations {
		if alt.ErrorToReturn == "" || (alt.TargetEndpoint == "" && alt.OverrideToReturn == "") {
			//TODO: return error
		}

		alteration := EndpointAlteration{
			TargetEndpoint:   alt.TargetEndpoint,
			ErrorToReturn:    alt.ErrorToReturn,
			OverrideToReturn: alt.OverrideToReturn,
		}
		d.Spec = append(d.Spec, alteration)
	}

	return &emptypb.Empty{}, nil
}

func (d *DisruptionListener) DisruptionStatus(context.Context, *emptypb.Empty) (*pb.DisruptionSpec, error) {
	ds := pb.DisruptionSpec{EndpointAlterations: []*pb.EndpointAlteration{}}

	for _, alt := range d.Spec {
		alteration := pb.EndpointAlteration{
			TargetEndpoint:   alt.TargetEndpoint,
			ErrorToReturn:    alt.ErrorToReturn,
			OverrideToReturn: alt.OverrideToReturn,
		}
		ds.EndpointAlterations = append(ds.EndpointAlterations, &alteration)
	}

	return &ds, nil
}

func (d *DisruptionListener) CleanDisruption(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	d.Spec = []EndpointAlteration{}
	return &emptypb.Empty{}, nil
}

func (d *DisruptionListener) ChaosServerInterceptor(ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (response interface{}, err error) {
	// Find out what api endpoint the request is for
	// FullMethod is the full RPC method string, i.e., /package.service/method.
	endpoint := info.FullMethod

	log.Printf("interceptor called on endpoint: %s; getting checked against %d endpoints", endpoint, len(d.Spec))

	for _, alteration := range d.Spec {
		if endpoint == alteration.TargetEndpoint {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("this was injected: %s", alteration.ErrorToReturn))
		} else {
			// TODO:  nothing?
		}
	}

	// Check our disruption to see how we should return

	// Return the specified error or empty struct (other responses tbd)

	return handler(ctx, req)
}
