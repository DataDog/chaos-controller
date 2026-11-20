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

var _ = Describe("DiskPressureSpec", func() {
	intPtr := func(i int) *int { return &i }

	When("Call the 'GenerateArgs' method", func() {
		DescribeTable("argument generation",
			func(spec DiskPressureSpec, expected []string) {
				Expect(spec.GenerateArgs()).To(Equal(expected))
			},
			Entry("with only bandwidth throttling (back-compat)",
				DiskPressureSpec{
					Path: "/mnt/data",
					Throttling: DiskPressureThrottlingSpec{
						ReadBytesPerSec:  intPtr(1024),
						WriteBytesPerSec: intPtr(4096),
					},
				},
				[]string{
					"disk-pressure", "--path", "/mnt/data",
					"--read-bytes-per-sec", "1024",
					"--write-bytes-per-sec", "4096",
				},
			),
			Entry("with only read iops throttling",
				DiskPressureSpec{
					Path: "/mnt/data",
					Throttling: DiskPressureThrottlingSpec{
						ReadIOPSPerSec: intPtr(50),
					},
				},
				[]string{
					"disk-pressure", "--path", "/mnt/data",
					"--read-iops-per-sec", "50",
				},
			),
			Entry("with only write iops throttling",
				DiskPressureSpec{
					Path: "/mnt/data",
					Throttling: DiskPressureThrottlingSpec{
						WriteIOPSPerSec: intPtr(75),
					},
				},
				[]string{
					"disk-pressure", "--path", "/mnt/data",
					"--write-iops-per-sec", "75",
				},
			),
			Entry("with bandwidth and iops throttling combined",
				DiskPressureSpec{
					Path: "/mnt/data",
					Throttling: DiskPressureThrottlingSpec{
						ReadBytesPerSec:  intPtr(1024),
						WriteBytesPerSec: intPtr(4096),
						ReadIOPSPerSec:   intPtr(50),
						WriteIOPSPerSec:  intPtr(75),
					},
				},
				[]string{
					"disk-pressure", "--path", "/mnt/data",
					"--read-bytes-per-sec", "1024",
					"--write-bytes-per-sec", "4096",
					"--read-iops-per-sec", "50",
					"--write-iops-per-sec", "75",
				},
			),
			Entry("with no throttling set",
				DiskPressureSpec{Path: "/mnt/data"},
				[]string{"disk-pressure", "--path", "/mnt/data"},
			),
		)
	})

	When("Call the 'Validate' method", func() {
		It("accepts a spec with no throttling set", func() {
			spec := DiskPressureSpec{Path: "/mnt/data"}
			Expect(spec.Validate()).To(Succeed())
		})

		It("accepts positive throttle values", func() {
			spec := DiskPressureSpec{
				Path: "/mnt/data",
				Throttling: DiskPressureThrottlingSpec{
					ReadBytesPerSec: intPtr(1024),
					ReadIOPSPerSec:  intPtr(50),
				},
			}
			Expect(spec.Validate()).To(Succeed())
		})

		It("accepts a zero throttle value (no-op, removes the limit)", func() {
			spec := DiskPressureSpec{
				Path: "/mnt/data",
				Throttling: DiskPressureThrottlingSpec{
					ReadBytesPerSec: intPtr(0),
					WriteIOPSPerSec: intPtr(0),
				},
			}
			Expect(spec.Validate()).To(Succeed())
		})

		DescribeTable("rejects negative throttle values",
			func(spec DiskPressureSpec, field string) {
				err := spec.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(field))
			},
			Entry("negative read bytes",
				DiskPressureSpec{Path: "/mnt/data", Throttling: DiskPressureThrottlingSpec{ReadBytesPerSec: intPtr(-1)}},
				"readBytesPerSec",
			),
			Entry("negative write bytes",
				DiskPressureSpec{Path: "/mnt/data", Throttling: DiskPressureThrottlingSpec{WriteBytesPerSec: intPtr(-1)}},
				"writeBytesPerSec",
			),
			Entry("negative read iops",
				DiskPressureSpec{Path: "/mnt/data", Throttling: DiskPressureThrottlingSpec{ReadIOPSPerSec: intPtr(-1)}},
				"readIOPSPerSec",
			),
			Entry("negative write iops",
				DiskPressureSpec{Path: "/mnt/data", Throttling: DiskPressureThrottlingSpec{WriteIOPSPerSec: intPtr(-5)}},
				"writeIOPSPerSec",
			),
		)
	})

	When("Call the 'Explain' method", func() {
		It("mentions iops throttling when set", func() {
			spec := DiskPressureSpec{
				Path: "/mnt/data",
				Throttling: DiskPressureThrottlingSpec{
					ReadIOPSPerSec:  intPtr(50),
					WriteIOPSPerSec: intPtr(75),
				},
			}

			explanation := spec.Explain()

			Expect(explanation).To(ContainElement(ContainSubstring("50 read io per second")))
			Expect(explanation).To(ContainElement(ContainSubstring("75 write io per second")))
		})
	})
})
