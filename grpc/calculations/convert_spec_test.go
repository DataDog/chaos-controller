// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package calculations_test

import (
	. "github.com/DataDog/chaos-controller/grpc/calculations"
	pb "github.com/DataDog/chaos-controller/grpc/disruptionlistener"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ConvertSpecifications", func() {
	It("converts valid alteration specs into weighted slice", func() {
		specs := []*pb.AlterationSpec{
			{ErrorToReturn: "NOT_FOUND", QueryPercent: 50},
			{OverrideToReturn: "{}", QueryPercent: 30},
		}
		result, err := ConvertSpecifications(specs)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(HaveLen(80))
	})

	It("returns error for invalid spec (both fields empty)", func() {
		specs := []*pb.AlterationSpec{
			{ErrorToReturn: "", OverrideToReturn: ""},
		}
		_, err := ConvertSpecifications(specs)
		Expect(err).To(HaveOccurred())
	})

	It("returns empty slice for empty input", func() {
		result, err := ConvertSpecifications([]*pb.AlterationSpec{})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(BeEmpty())
	})
})
