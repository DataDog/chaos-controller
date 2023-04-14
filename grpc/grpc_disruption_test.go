// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package grpc_test

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/grpc"
	pb "github.com/DataDog/chaos-controller/grpc/disruptionlistener"

	. "github.com/onsi/ginkgo"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ = Describe("Test send and clean disruption", func() {
	Context("Basic GRPCDisruptionSpec", func() {
		// define parameters of NewGRPCDisruptionInjector
		spec := v1beta1.GRPCDisruptionSpec{
			Port: 2000,
			Endpoints: []v1beta1.EndpointAlteration{
				{
					TargetEndpoint:   "/chaosdogfood.ChaosDogfood/order",
					ErrorToReturn:    "",
					OverrideToReturn: "{}",
					QueryPercent:     50,
				},
				{
					TargetEndpoint:   "/chaosdogfood.ChaosDogfood/getCatalog",
					ErrorToReturn:    "NOT_FOUND",
					OverrideToReturn: "",
					QueryPercent:     25,
				},
				{
					TargetEndpoint:   "/chaosdogfood.ChaosDogfood/getCatalog",
					ErrorToReturn:    "ALREADY_EXISTS",
					OverrideToReturn: "",
					QueryPercent:     50,
				},
				{
					TargetEndpoint:   "/chaosdogfood.ChaosDogfood/getCatalog",
					ErrorToReturn:    "",
					OverrideToReturn: "{}",
					QueryPercent:     0,
				},
			},
		}

		disruptionListenerClient := pb.NewMockDisruptionListenerClient(GinkgoT())

		Specify("calls Disrupt and ResetDisruptions with expected parameters", func() {
			// define expectations
			disruptionListenerClient.EXPECT().Disrupt(
				mock.Anything,
				mock.MatchedBy(func(spec *pb.DisruptionSpec) bool {
					endpts := spec.Endpoints
					if len(endpts) != 2 {
						return false
					}

					altSpecForOrder := []*pb.AlterationSpec{
						{
							ErrorToReturn:    "",
							OverrideToReturn: "{}",
							QueryPercent:     int32(50),
						},
					}

					altSpecForGetCatalog := []*pb.AlterationSpec{
						{
							ErrorToReturn:    "NOT_FOUND",
							OverrideToReturn: "",
							QueryPercent:     int32(25),
						},
						{
							ErrorToReturn:    "ALREADY_EXISTS",
							OverrideToReturn: "",
							QueryPercent:     int32(50),
						},
						{
							ErrorToReturn:    "",
							OverrideToReturn: "{}",
						},
					}

					// handling that the results of `endpts` is indeterminate
					if endpts[0].TargetEndpoint == "/chaosdogfood.ChaosDogfood/order" {
						return specsAreEqual(endpts[0].Alterations[0], altSpecForOrder[0]) &&
							specsAreEqual(endpts[1].Alterations[0], altSpecForGetCatalog[0]) &&
							specsAreEqual(endpts[1].Alterations[1], altSpecForGetCatalog[1]) &&
							specsAreEqual(endpts[1].Alterations[2], altSpecForGetCatalog[2])
					}
					return specsAreEqual(endpts[1].Alterations[0], altSpecForOrder[0]) &&
						specsAreEqual(endpts[0].Alterations[0], altSpecForGetCatalog[0]) &&
						specsAreEqual(endpts[0].Alterations[1], altSpecForGetCatalog[1]) &&
						specsAreEqual(endpts[0].Alterations[2], altSpecForGetCatalog[2])
				}),
			).Return(&emptypb.Empty{}, nil)

			disruptionListenerClient.EXPECT().ResetDisruptions(
				mock.Anything,
				&emptypb.Empty{},
			).Return(&emptypb.Empty{}, nil)

			grpc.SendGrpcDisruption(disruptionListenerClient, spec)
			grpc.ClearGrpcDisruptions(disruptionListenerClient)

			// run test
			disruptionListenerClient.AssertExpectations(GinkgoT())
		})
	})
})

func specsAreEqual(actual *pb.AlterationSpec, expected *pb.AlterationSpec) bool {
	return actual.ErrorToReturn == expected.ErrorToReturn &&
		actual.OverrideToReturn == expected.OverrideToReturn &&
		actual.QueryPercent == expected.QueryPercent
}
