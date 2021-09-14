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

package injector_test

import (
	"testing"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/grpc"
	. "github.com/DataDog/chaos-controller/injector"

	pb "github.com/DataDog/chaos-controller/grpc/disruption_listener"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestSendAndCleaDisruption(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "gRPC Disruption Suite - Send Disruption")

	// define parameters of NewGRPCDisruptionInjector
	spec := v1beta1.GRPCDisruptionSpec{
		Port: 2000,
		Endpoints: []v1beta1.EndpointAlteration{
			{
				TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
				ErrorToReturn:    "",
				OverrideToReturn: "{}",
				QueryPercent:     50,
			},
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
		},
	}

	disruptionListenerClient := &DisruptionListenerClientMock{}

	// define expectations
	disruptionListenerClient.On("SendDisruption",
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
		}),
	).Return(&emptypb.Empty{}, nil)

	disruptionListenerClient.On("CleanDisruption",
		mock.Anything,
		&emptypb.Empty{},
	).Return(&emptypb.Empty{}, nil)

	grpc.ExecuteSendDisruption(disruptionListenerClient, spec)
	grpc.ExecuteCleanDisruption(disruptionListenerClient)

	// run test
	disruptionListenerClient.AssertExpectations(t)
}

func specsAreEqual(actual *pb.AlterationSpec, expected *pb.AlterationSpec) bool {
	return actual.ErrorToReturn == expected.ErrorToReturn &&
		actual.OverrideToReturn == expected.OverrideToReturn &&
		actual.QueryPercent == expected.QueryPercent
}
