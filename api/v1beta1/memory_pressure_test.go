// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package v1beta1_test

import (
	. "github.com/DataDog/chaos-controller/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MemoryPressureSpec", func() {
	Describe("Validate", func() {
		It("succeeds with valid percentage", func() {
			spec := &MemoryPressureSpec{
				TargetPercent: "76%",
			}
			Expect(spec.Validate()).To(Succeed())
		})

		It("succeeds with percentage without suffix", func() {
			spec := &MemoryPressureSpec{
				TargetPercent: "50",
			}
			Expect(spec.Validate()).To(Succeed())
		})

		It("succeeds with ramp duration", func() {
			spec := &MemoryPressureSpec{
				TargetPercent: "76%",
				RampDuration:  DisruptionDuration("10m"),
			}
			Expect(spec.Validate()).To(Succeed())
		})

		It("fails with zero percent", func() {
			spec := &MemoryPressureSpec{
				TargetPercent: "0%",
			}
			Expect(spec.Validate()).To(HaveOccurred())
		})

		It("fails with percent over 100", func() {
			spec := &MemoryPressureSpec{
				TargetPercent: "101%",
			}
			Expect(spec.Validate()).To(HaveOccurred())
		})

		It("fails with invalid percent string", func() {
			spec := &MemoryPressureSpec{
				TargetPercent: "abc",
			}
			Expect(spec.Validate()).To(HaveOccurred())
		})
	})

	Describe("GenerateArgs", func() {
		It("generates args without ramp duration", func() {
			spec := &MemoryPressureSpec{
				TargetPercent: "76%",
			}
			args := spec.GenerateArgs()
			Expect(args).To(Equal([]string{"memory-pressure", "--target-percent", "76%"}))
		})

		It("generates args with ramp duration", func() {
			spec := &MemoryPressureSpec{
				TargetPercent: "50%",
				RampDuration:  DisruptionDuration("10m0s"),
			}
			args := spec.GenerateArgs()
			Expect(args).To(Equal([]string{"memory-pressure", "--target-percent", "50%", "--ramp-duration", "10m0s"}))
		})
	})

	Describe("Explain", func() {
		It("explains without ramp", func() {
			spec := &MemoryPressureSpec{
				TargetPercent: "76%",
			}
			explanation := spec.Explain()
			Expect(explanation).To(HaveLen(2))
			Expect(explanation[1]).To(ContainSubstring("76%"))
			Expect(explanation[1]).To(ContainSubstring("immediately"))
		})

		It("explains with ramp", func() {
			spec := &MemoryPressureSpec{
				TargetPercent: "50%",
				RampDuration:  DisruptionDuration("10m0s"),
			}
			explanation := spec.Explain()
			Expect(explanation).To(HaveLen(2))
			Expect(explanation[1]).To(ContainSubstring("ramping up"))
		})
	})
})
