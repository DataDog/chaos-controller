// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package v1beta1_test

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/grpc/calculations"
	pb "github.com/DataDog/chaos-controller/grpc/disruptionlistener"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ = Describe("gRPC implicit percentage admission", func() {
	DescribeTable("agrees with listener allocation regardless of alteration order", func(percentages []int, valid bool) {
		spec := v1beta1.GRPCDisruptionSpec{Port: 50051}
		var alterations []*pb.AlterationSpec
		errors := []string{"NOT_FOUND", "UNAVAILABLE", "ABORTED"}
		for i, percentage := range percentages {
			spec.Endpoints = append(spec.Endpoints, v1beta1.EndpointAlteration{
				TargetEndpoint: "/svc/Method", ErrorToReturn: errors[i], QueryPercent: percentage,
			})
			alterations = append(alterations, &pb.AlterationSpec{
				ErrorToReturn: errors[i], QueryPercent: int32(percentage),
			})
		}

		_, listenerErr := calculations.ConvertSpecifications(alterations)
		if valid {
			Expect(listenerErr).NotTo(HaveOccurred())
			Expect(spec.Validate()).To(Succeed())
		} else {
			Expect(status.Code(listenerErr)).To(Equal(codes.InvalidArgument))
			Expect(spec.Validate()).NotTo(Succeed(), "admission accepted an allocation the listener rejects")
		}
	},
		Entry("one implicit alteration after 100 percent", []int{100, 0}, false),
		Entry("one implicit alteration before 100 percent", []int{0, 100}, false),
		Entry("two implicit alterations before 99 percent", []int{0, 0, 99}, false),
		Entry("two implicit alterations after 99 percent", []int{99, 0, 0}, false),
		Entry("one percent remains for one alteration", []int{99, 0}, true),
		Entry("two percent remains for two alterations", []int{0, 0, 98}, true),
	)
})
