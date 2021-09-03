// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1_test

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("QueryPercent Validation", func() {
	It("Query percent of 101 does not validate", func() {
		spec := v1beta1.GRPCDisruptionSpec{
			Port: 50051,
			Endpoints: []v1beta1.EndpointAlteration{
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "CANCELED",
					OverrideToReturn: "{}",
					QueryPercent:     101,
				},
			},
		}

		Expect(spec.Validate()).ToNot(BeNil())
	})

	It("Query percent of 100 does validate", func() {
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

		Expect(spec.Validate()).To(BeNil())
	})

	It("Query percent of 99 does validate", func() {
		spec := v1beta1.GRPCDisruptionSpec{
			Port: 50051,
			Endpoints: []v1beta1.EndpointAlteration{
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "CANCELED",
					OverrideToReturn: "{}",
					QueryPercent:     99,
				},
			},
		}

		Expect(spec.Validate()).To(BeNil())
	})

	It("Query percent of 100 on two endpoints does validate", func() {
		spec := v1beta1.GRPCDisruptionSpec{
			Port: 50051,
			Endpoints: []v1beta1.EndpointAlteration{
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "CANCELED",
					OverrideToReturn: "{}",
					QueryPercent:     100,
				},
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/getCatalog",
					ErrorToReturn:    "CANCELED",
					OverrideToReturn: "{}",
					QueryPercent:     100,
				},
			}}

		Expect(spec.Validate()).To(BeNil())
	})

	It("Query percent which sums to 100 on two endpoints does validate", func() {
		spec := v1beta1.GRPCDisruptionSpec{
			Port: 50051,
			Endpoints: []v1beta1.EndpointAlteration{
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "CANCELED",
					OverrideToReturn: "{}",
					QueryPercent:     60,
				},
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "CANCELED",
					OverrideToReturn: "{}",
					QueryPercent:     40,
				},
			}}

		Expect(spec.Validate()).To(BeNil())
	})

	It("Query percent which sums to less than 100 on two endpoints does validate", func() {
		spec := v1beta1.GRPCDisruptionSpec{
			Port: 50051,
			Endpoints: []v1beta1.EndpointAlteration{
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "CANCELED",
					OverrideToReturn: "{}",
					QueryPercent:     49,
				},
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "CANCELED",
					OverrideToReturn: "{}",
					QueryPercent:     49,
				},
			}}

		Expect(spec.Validate()).To(BeNil())
	})

	It("Query percent which sums to more than 100 on two endpoints does validate", func() {
		spec := v1beta1.GRPCDisruptionSpec{
			Port: 50051,
			Endpoints: []v1beta1.EndpointAlteration{
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "CANCELED",
					OverrideToReturn: "{}",
					QueryPercent:     50,
				},
				{
					TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
					ErrorToReturn:    "CANCELED",
					OverrideToReturn: "{}",
					QueryPercent:     51,
				},
			}}

		Expect(spec.Validate()).ToNot(BeNil())
	})
})
