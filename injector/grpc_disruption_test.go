// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package injector_test

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	pb "github.com/DataDog/chaos-controller/grpc/disruption_listener"
	. "github.com/DataDog/chaos-controller/injector"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ = Describe("Failure", func() {
	var (
		inj                      Injector
		config                   GRPCDisruptionInjectorConfig
		spec                     v1beta1.GRPCDisruptionSpec
		disruptionListenerClient *DisruptionListenerClientMock
	)

	BeforeEach(func() {
		disruptionListenerClient = &DisruptionListenerClientMock{}
		disruptionListenerClient.On("SendDisruption", mock.Anything, mock.Anything).Return(&emptypb.Empty{}, nil)
		disruptionListenerClient.On("CleanDisruption", mock.Anything, mock.Anything).Return(&emptypb.Empty{}, nil)

		// config
		config = GRPCDisruptionInjectorConfig{
			Config: Config{
				Log: log,
			},
		}

		spec = v1beta1.GRPCDisruptionSpec{
			Port: 2000,
			Endpoints: []v1beta1.EndpointAlteration{
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/getCatalog",
					ErrorToReturn:    "NOT_FOUND",
					OverrideToReturn: "",
					QueryPercent:     25,
				},
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/getCatalog",
					ErrorToReturn:    "ALREADY_EXISTS",
					OverrideToReturn: "",
					QueryPercent:     50,
				},
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/getCatalog",
					ErrorToReturn:    "",
					OverrideToReturn: "{}",
					QueryPercent:     0,
				},
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "",
					OverrideToReturn: "{}",
					QueryPercent:     50,
				},
			},
		}
	})

	JustBeforeEach(func() {
		var err error
		inj, err = NewGRPCDisruptionInjector(spec, config, disruptionListenerClient)
		Expect(err).To(BeNil())
	})
	Describe("inj.Inject", func() {
		JustBeforeEach(func() {
			Expect(inj.Inject()).To(BeNil())
		})

		It("should make a sendDisruption call to the disruption_listener on target service", func() {
			disruptionListenerSpecMatcher := mock.MatchedBy(func(spec *pb.DisruptionSpec) bool {
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

				if endpts[0].TargetEndpoint == "/chaos_dogfood.ChaosDogfood/order" {
					return specsAreEqual(endpts[0].Alterations[0], altSpecForOrder[0]) &&
						specsAreEqual(endpts[1].Alterations[0], altSpecForGetCatalog[0]) &&
						specsAreEqual(endpts[1].Alterations[1], altSpecForGetCatalog[1]) &&
						specsAreEqual(endpts[1].Alterations[2], altSpecForGetCatalog[2])
				}
				return specsAreEqual(endpts[1].Alterations[0], altSpecForOrder[0]) &&
					specsAreEqual(endpts[0].Alterations[0], altSpecForGetCatalog[0]) &&
					specsAreEqual(endpts[0].Alterations[1], altSpecForGetCatalog[1]) &&
					specsAreEqual(endpts[0].Alterations[2], altSpecForGetCatalog[2])
			})

			disruptionListenerClient.AssertCalled(
				GinkgoT(),
				"SendDisruption",
				disruptionListenerSpecMatcher,
			)
		})
	})

	Describe("inj.Clean", func() {
		JustBeforeEach(func() {
			Expect(inj.Clean()).To(BeNil())
		})

		It("should make a cleanDisruption call to the disruption_listener on target service", func() {
			disruptionListenerClient.AssertCalled(
				GinkgoT(),
				"CleanDisruption",
				&emptypb.Empty{},
			)
		})
	})
})

func specsAreEqual(actual *pb.AlterationSpec, expected *pb.AlterationSpec) bool {
	return actual.ErrorToReturn == expected.ErrorToReturn &&
		actual.OverrideToReturn == expected.OverrideToReturn &&
		actual.QueryPercent == expected.QueryPercent
}
