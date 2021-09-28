// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package api_test

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("GRPCDisruption Validation", func() {
	It("Error and override cannot both be defined", func() {
		spec := v1beta1.GRPCDisruptionSpec{
			Port: 50051,
			Endpoints: []v1beta1.EndpointAlteration{
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "CANCELED",
					OverrideToReturn: "{}",
					QueryPercent:     100,
				},
			},
		}

		err := spec.Validate()
		Expect(err.Error()).To(Equal("the gRPC disruption has ErrorToReturn and OverrideToReturn specified for endpoint /chaos_dogfood.ChaosDogfood/order, but it can only have one"))
	})

	It("Error and override cannot both be undefined", func() {
		spec := v1beta1.GRPCDisruptionSpec{
			Port: 50051,
			Endpoints: []v1beta1.EndpointAlteration{
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "",
					OverrideToReturn: "",
					QueryPercent:     100,
				},
			},
		}

		err := spec.Validate()
		Expect(err.Error()).To(Equal("the gRPC disruption must have either ErrorToReturn or OverrideToReturn specified for endpoint /chaos_dogfood.ChaosDogfood/order"))
	})

	It("Query percent of 100 does validate", func() {
		spec := v1beta1.GRPCDisruptionSpec{
			Port: 50051,
			Endpoints: []v1beta1.EndpointAlteration{
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "CANCELED",
					OverrideToReturn: "",
					QueryPercent:     100,
				},
			},
		}

		Expect(spec.Validate()).To(BeNil())
	})

	It("Query percent of 99 does validate", func() {
		spec := v1beta1.GRPCDisruptionSpec{
			Port: 50051,
			Endpoints: []v1beta1.EndpointAlteration{
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "CANCELED",
					OverrideToReturn: "",
					QueryPercent:     99,
				},
			},
		}

		Expect(spec.Validate()).To(BeNil())
	})

	It("Query percent of 100 on two separate endpoints does validate", func() {
		spec := v1beta1.GRPCDisruptionSpec{
			Port: 50051,
			Endpoints: []v1beta1.EndpointAlteration{
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "CANCELED",
					OverrideToReturn: "",
					QueryPercent:     100,
				},
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/getCatalog",
					ErrorToReturn:    "CANCELED",
					OverrideToReturn: "",
					QueryPercent:     100,
				},
			}}

		Expect(spec.Validate()).To(BeNil())
	})

	It("Query percents which sum to exactly 100 on one endpoint does validate", func() {
		spec := v1beta1.GRPCDisruptionSpec{
			Port: 50051,
			Endpoints: []v1beta1.EndpointAlteration{
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "CANCELED",
					OverrideToReturn: "",
					QueryPercent:     60,
				},
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "ALREADY_EXISTS",
					OverrideToReturn: "",
					QueryPercent:     40,
				},
			}}

		Expect(spec.Validate()).To(BeNil())
	})

	It("Query percents which sum to less than 100 on one endpoint does validate", func() {
		spec := v1beta1.GRPCDisruptionSpec{
			Port: 50051,
			Endpoints: []v1beta1.EndpointAlteration{
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "CANCELED",
					OverrideToReturn: "",
					QueryPercent:     49,
				},
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "ALREADY_EXISTS",
					OverrideToReturn: "",
					QueryPercent:     49,
				},
			}}

		Expect(spec.Validate()).To(BeNil())
	})

	It("Query percents which sum to more than 100 on one endpoint does not validate", func() {
		spec := v1beta1.GRPCDisruptionSpec{
			Port: 50051,
			Endpoints: []v1beta1.EndpointAlteration{
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "CANCELED",
					OverrideToReturn: "",
					QueryPercent:     50,
				},
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "ALREADY_EXISTS",
					OverrideToReturn: "",
					QueryPercent:     51,
				},
			}}

		Expect(spec.Validate()).ToNot(BeNil())
	})

	It("Errors not in the standard grpc errors do not validate", func() {
		spec := v1beta1.GRPCDisruptionSpec{
			Port: 50051,
			Endpoints: []v1beta1.EndpointAlteration{
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "MEOW",
					OverrideToReturn: "",
					QueryPercent:     50,
				},
			},
		}

		Expect(spec.Validate()).ToNot(BeNil())
	})

	It("All standard grpc errors validate", func() {
		spec := v1beta1.GRPCDisruptionSpec{
			Port: 50051,
			Endpoints: []v1beta1.EndpointAlteration{
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "CANCELED",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "ALREADY_EXISTS",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "UNKNOWN",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "INVALID_ARGUMENT",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "DEADLINE_EXCEEDED",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "NOT_FOUND",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "PERMISSION_DENIED",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "RESOURCE_EXHAUSTED",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "FAILED_PRECONDITION",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "ABORTED",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "OUT_OF_RANGE",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "UNIMPLEMENTED",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
			},
		}

		Expect(spec.Validate()).To(BeNil())
	})

	It("Good errors but 1 too many unquantified errors returns error because some errors will never get returned", func() {
		spec := v1beta1.GRPCDisruptionSpec{
			Port: 50051,
			Endpoints: []v1beta1.EndpointAlteration{
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "CANCELED",
					OverrideToReturn: "",
					QueryPercent:     90,
				},
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "ALREADY_EXISTS",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "UNKNOWN",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "INVALID_ARGUMENT",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "DEADLINE_EXCEEDED",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "NOT_FOUND",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "PERMISSION_DENIED",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "RESOURCE_EXHAUSTED",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "FAILED_PRECONDITION",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "ABORTED",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "OUT_OF_RANGE",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "UNIMPLEMENTED",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
			},
		}

		Expect(spec.Validate()).ToNot(BeNil())
	})
})
