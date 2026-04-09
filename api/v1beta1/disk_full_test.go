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

var _ = Describe("DiskFullSpec", func() {
	When("Call the 'Validate' method", func() {
		DescribeTable("success cases",
			func(spec DiskFullSpec) {
				Expect(spec.Validate()).Should(Succeed())
			},
			Entry("with capacity percentage",
				DiskFullSpec{
					Path:     "/data",
					Capacity: "95%",
				},
			),
			Entry("with capacity at 1%",
				DiskFullSpec{
					Path:     "/data",
					Capacity: "1%",
				},
			),
			Entry("with capacity at 100%",
				DiskFullSpec{
					Path:     "/data",
					Capacity: "100%",
				},
			),
			Entry("with remaining in Mi",
				DiskFullSpec{
					Path:      "/data",
					Remaining: "50Mi",
				},
			),
			Entry("with remaining in Gi",
				DiskFullSpec{
					Path:      "/var/log",
					Remaining: "1Gi",
				},
			),
			Entry("with remaining at 0",
				DiskFullSpec{
					Path:      "/data",
					Remaining: "0",
				},
			),
		)

		DescribeTable("error cases",
			func(spec DiskFullSpec, expectedErrors []string) {
				err := spec.Validate()
				Expect(err).To(HaveOccurred())
				for _, expected := range expectedErrors {
					Expect(err.Error()).To(ContainSubstring(expected))
				}
			},
			Entry("with empty path",
				DiskFullSpec{
					Path:     "",
					Capacity: "95%",
				},
				[]string{"the path of the disk full disruption must not be empty"},
			),
			Entry("with blank path",
				DiskFullSpec{
					Path:     "   ",
					Capacity: "95%",
				},
				[]string{"the path of the disk full disruption must not be empty"},
			),
			Entry("with both capacity and remaining set",
				DiskFullSpec{
					Path:      "/data",
					Capacity:  "95%",
					Remaining: "50Mi",
				},
				[]string{"capacity and remaining are mutually exclusive"},
			),
			Entry("with neither capacity nor remaining set",
				DiskFullSpec{
					Path: "/data",
				},
				[]string{"one of capacity or remaining must be set"},
			),
			Entry("with capacity missing percent suffix",
				DiskFullSpec{
					Path:     "/data",
					Capacity: "95",
				},
				[]string{"capacity must be a percentage suffixed with %"},
			),
			Entry("with capacity at 0%",
				DiskFullSpec{
					Path:     "/data",
					Capacity: "0%",
				},
				[]string{"capacity percentage must be between 1 and 100"},
			),
			Entry("with capacity at 101%",
				DiskFullSpec{
					Path:     "/data",
					Capacity: "101%",
				},
				[]string{"capacity percentage must be between 1 and 100"},
			),
			Entry("with non-numeric capacity",
				DiskFullSpec{
					Path:     "/data",
					Capacity: "abc%",
				},
				[]string{"capacity percentage must be an integer"},
			),
			Entry("with invalid remaining quantity",
				DiskFullSpec{
					Path:      "/data",
					Remaining: "not-a-quantity",
				},
				[]string{"remaining must be a valid Kubernetes resource quantity"},
			),
			Entry("with negative remaining",
				DiskFullSpec{
					Path:      "/data",
					Remaining: "-1Mi",
				},
				[]string{"remaining must not be negative"},
			),
			Entry("with empty path and no capacity/remaining",
				DiskFullSpec{
					Path: "",
				},
				[]string{
					"the path of the disk full disruption must not be empty",
					"one of capacity or remaining must be set",
				},
			),
		)
	})

	When("Call the 'GenerateArgs' method", func() {
		DescribeTable("success cases",
			func(spec DiskFullSpec, expectedArgs []string) {
				expectedArgs = append([]string{"disk-full"}, expectedArgs...)
				args := spec.GenerateArgs()
				Expect(args).Should(Equal(expectedArgs))
			},
			Entry("with capacity",
				DiskFullSpec{
					Path:     "/data",
					Capacity: "95%",
				},
				[]string{"--path", "/data", "--capacity", "95%"},
			),
			Entry("with remaining",
				DiskFullSpec{
					Path:      "/data",
					Remaining: "50Mi",
				},
				[]string{"--path", "/data", "--remaining", "50Mi"},
			),
		)
	})

	When("Call the 'Explain' method", func() {
		It("explains capacity mode", func() {
			spec := DiskFullSpec{
				Path:     "/data",
				Capacity: "95%",
			}
			explanation := spec.Explain()
			Expect(explanation).To(HaveLen(2))
			Expect(explanation[1]).To(ContainSubstring("/data"))
			Expect(explanation[1]).To(ContainSubstring("95%"))
			Expect(explanation[1]).To(ContainSubstring("ENOSPC"))
		})

		It("explains remaining mode", func() {
			spec := DiskFullSpec{
				Path:      "/var/log",
				Remaining: "50Mi",
			}
			explanation := spec.Explain()
			Expect(explanation).To(HaveLen(2))
			Expect(explanation[1]).To(ContainSubstring("/var/log"))
			Expect(explanation[1]).To(ContainSubstring("50Mi"))
			Expect(explanation[1]).To(ContainSubstring("ENOSPC"))
		})

	})
})
