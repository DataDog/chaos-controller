// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package grpc_test

import (
	"context"

	chaos "github.com/DataDog/chaos-controller/grpc"
	pb "github.com/DataDog/chaos-controller/grpc/disruptionlistener"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ = Describe("gRPC listener percentage lower bound", func() {
	DescribeTable("rejects negative percentages instead of interpreting them as unspecified", func(percentage int32) {
		listener := chaos.NewDisruptionListener(zap.NewNop().Sugar())
		ctx := context.Background()
		_, err := listener.Disrupt(ctx, &pb.DisruptionSpec{
			Endpoints: []*pb.EndpointSpec{{
				TargetEndpoint: "/svc/Method",
				Alterations:    []*pb.AlterationSpec{{ErrorToReturn: "NOT_FOUND", QueryPercent: percentage}},
			}},
		})
		Expect(status.Code(err)).To(Equal(codes.InvalidArgument))

		called := false
		_, err = listener.ChaosServerInterceptor(ctx, nil, &googlegrpc.UnaryServerInfo{FullMethod: "/svc/Method"},
			func(context.Context, interface{}) (interface{}, error) {
				called = true
				return nil, nil
			})
		Expect(err).NotTo(HaveOccurred())
		Expect(called).To(BeTrue(), "a rejected specification must leave the endpoint undisrupted")
	},
		Entry("minus one", int32(-1)),
		Entry("minimum int32", int32(-2147483648)),
	)
})
