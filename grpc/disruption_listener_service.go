package grpc

import (
	"context"
	"fmt"
	"log"

	pb "github.com/DataDog/chaos-controller/grpc/disruption_listener"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// DisruptionListener is a gRPC Service that can disrupt endpoints of a gRPC server
type DisruptionListener struct {
	pb.UnimplementedDisruptionListenerServer
	Spec GrpcDisruptionSpec
}

// GrpcDisruptionSpec configures the DisruptionListener to chaos test endpoints of a gRPC server
type GrpcDisruptionSpec []EndpointAlteration

// EndpointAlteration configures endpoints that the DisruptionListener chaos tests on a gRPC server
type EndpointAlteration struct {
	TargetEndpoint   string
	ErrorToReturn    string
	OverrideToReturn string
}

var (
	errorMap = map[string]codes.Code{
		"OK":                  codes.OK,
		"CANCELED":            codes.Canceled,
		"UNKNOWN":             codes.Unknown,
		"INVALID_ARGUMENT":    codes.InvalidArgument,
		"DEADLINE_EXCEEDED":   codes.DeadlineExceeded,
		"NOT_FOUND":           codes.NotFound,
		"ALREADY_EXISTS":      codes.AlreadyExists,
		"PERMISSION_DENIED":   codes.PermissionDenied,
		"RESOURCE_EXHAUSTED":  codes.ResourceExhausted,
		"FAILED_PRECONDITION": codes.FailedPrecondition,
		"ABORTED":             codes.Aborted,
		"OUT_OF_RANGE":        codes.OutOfRange,
		"UNIMPLEMENTED":       codes.Unimplemented,
		"INTERNAL":            codes.Internal,
		"UNAVAILABLE":         codes.Unavailable,
		"DATALOSS":            codes.DataLoss,
		"UNAUTHENTICATED":     codes.Unauthenticated,
	}
)

// SendDisruption makes a gRPC call to DisruptionListener
func (d *DisruptionListener) SendDisruption(ctx context.Context, ds *pb.DisruptionSpec) (*emptypb.Empty, error) {
	if ds == nil {
		log.Fatal("Cannot execute SendDisruption when DisruptionSpec is nil")
	}

	for _, alt := range ds.EndpointAlterations {
		if alt.TargetEndpoint == "" {
			log.Print("SendDisruption does not specify TargetEndpoint for all endpointAlterations")
			return nil, status.Error(codes.InvalidArgument, "Cannot execute SendDisruption without specifying TargetEndpoint for all endpointAlterations")
		}
		if alt.ErrorToReturn == "" && alt.OverrideToReturn == "" {
			log.Print("For at least one endpointAlteration, SendDisruption specifies neither ErrorToReturn nor OverrideToReturn")
			return nil, status.Error(codes.InvalidArgument, "Cannot execute SendDisruption without specifying either ErrorToReturn or OverrideToReturn for all target endpoints")
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

// DisruptionStatus lists all configured endpoint alterations for DisruptionListener
func (d *DisruptionListener) DisruptionStatus(context.Context, *emptypb.Empty) (*pb.DisruptionSpec, error) {
	ds := pb.DisruptionSpec{EndpointAlterations: []*pb.EndpointAlteration{}}

	for _, alt := range d.Spec {
		ds.EndpointAlterations = append(
			ds.EndpointAlterations,
			&pb.EndpointAlteration{
				TargetEndpoint:   alt.TargetEndpoint,
				ErrorToReturn:    alt.ErrorToReturn,
				OverrideToReturn: alt.OverrideToReturn,
			},
		)
	}
	return &ds, nil
}

// CleanDisruption removes all configured endpoint alterations for DisruptionListener
func (d *DisruptionListener) CleanDisruption(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	d.Spec = []EndpointAlteration{}
	return &emptypb.Empty{}, nil
}

// ChaosServerInterceptor is a function which can be registered on instantiation of a gRPC server
// to intercept all traffic to the server and crosscheck their endpoints to disrupt them
func (d *DisruptionListener) ChaosServerInterceptor(ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (response interface{}, err error) {
	// Find out what api endpoint the request is for
	// FullMethod is the full RPC method string, i.e., /package.service/method.
	endpoint := info.FullMethod

	for _, alteration := range d.Spec {
		if endpoint == alteration.TargetEndpoint {
			if alteration.ErrorToReturn != "" {
				log.Printf("Error Code: %s", errorMap[alteration.ErrorToReturn])
				return nil, status.Error(errorMap[alteration.ErrorToReturn], fmt.Sprintf("Chaos-Controller injected this error: %s", alteration.ErrorToReturn))
			} else if alteration.OverrideToReturn != "" {
				log.Printf("OverrideToReturn: %s", alteration.OverrideToReturn)
				return &emptypb.Empty{}, nil
			} else {
				log.Printf("Endpoint %s should define either an ErrorToReturn or OverrideToReturn but does not", alteration.TargetEndpoint)
			}
		}
	}

	return handler(ctx, req)
}
