// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package grpc_test

import (
	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	pb "github.com/DataDog/chaos-controller/grpc/disruption_listener"

	. "github.com/DataDog/chaos-controller/grpc"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("construct DisruptionListener query for configuring disruptions from api spec", func() {
	var (
		endpointAlterations []chaosv1beta1.EndpointAlteration
		endpointSpec        []*pb.EndpointSpec
	)

	Context("with five alterations which add up to less than 100", func() {
		BeforeEach(func() {
			endpointAlterations = []chaosv1beta1.EndpointAlteration{
				{
					TargetEndpoint:   "service/api_1",
					ErrorToReturn:    "CANCELLED",
					OverrideToReturn: "",
					QueryPercent:     25,
				},
				{
					TargetEndpoint:   "service/api_2",
					ErrorToReturn:    "PERMISSION_DENIED",
					OverrideToReturn: "",
					QueryPercent:     50,
				},
				{
					TargetEndpoint:   "service/api_1",
					ErrorToReturn:    "ALREADY_EXISTS",
					OverrideToReturn: "",
					QueryPercent:     20,
				},
				{
					TargetEndpoint:   "service/api_2",
					ErrorToReturn:    "NOT_FOUND",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					TargetEndpoint:   "service/api_1",
					ErrorToReturn:    "{}}",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
			}

			var err error
			endpointSpec = GenerateEndpointSpecs(endpointAlterations)
			Expect(err).To(BeNil())
		})

		It("should create a list of endpointSpecs with 2 elements", func() {
			Expect(len(endpointSpec)).To(Equal(2))
		})

		It("should create and endpointSpec for api_1 with 3 elements", func() {
			Expect(len(endpointSpec[0].TargetEndpoint)).To(Equal("service/api_1"))
			Expect(len(endpointSpec[0].Alterations)).To(Equal(3))

			Expect(endpointSpec[0].Alterations[0].ErrorToReturn).To(Equal("CANCELLED"))
			Expect(endpointSpec[0].Alterations[0].OverrideToReturn).To(Equal(""))
			Expect(endpointSpec[0].Alterations[0].QueryPercent).To(Equal(25))

			Expect(endpointSpec[0].Alterations[1].ErrorToReturn).To(Equal("ALREADY_EXISTS"))
			Expect(endpointSpec[0].Alterations[1].OverrideToReturn).To(Equal(""))
			Expect(endpointSpec[0].Alterations[1].QueryPercent).To(Equal(20))

			Expect(endpointSpec[0].Alterations[2].ErrorToReturn).To(Equal(""))
			Expect(endpointSpec[0].Alterations[2].OverrideToReturn).To(Equal("{}"))
			Expect(endpointSpec[0].Alterations[2].QueryPercent).To(Equal(0))
		})

		It("should create and endpointSpec for api_2 with 2 elements", func() {
			Expect(len(endpointSpec[0].TargetEndpoint)).To(Equal("service/api_2"))
			Expect(len(endpointSpec[1].Alterations)).To(Equal(2))

			Expect(endpointSpec[1].Alterations[0].ErrorToReturn).To(Equal("PERMISSION_DENIED"))
			Expect(endpointSpec[1].Alterations[0].OverrideToReturn).To(Equal(""))
			Expect(endpointSpec[1].Alterations[0].QueryPercent).To(Equal(50))

			Expect(endpointSpec[1].Alterations[1].ErrorToReturn).To(Equal("NOT_FOUND"))
			Expect(endpointSpec[1].Alterations[1].OverrideToReturn).To(Equal(""))
			Expect(endpointSpec[1].Alterations[1].QueryPercent).To(Equal(0))
		})
	})
})
