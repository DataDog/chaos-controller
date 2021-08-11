// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package grpc_test

import (
	. "github.com/DataDog/chaos-controller/grpc"
	pb "github.com/DataDog/chaos-controller/grpc/disruption_listener"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("get mapping from PercentSlot to Alteration based on AlterationToPercentAffected", func() {
	var (
		alterationSpecs []*pb.AlterationSpec
		mapping         map[AlterationConfiguration]PercentAffected
	)

	Context("with three alterations which add up to less than 100", func() {
		BeforeEach(func() {
			alterationSpecs = []*pb.AlterationSpec{
				{
					ErrorToReturn:    "CANCELED",
					OverrideToReturn: "",
					QueryPercent:     int32(20),
				},
				{
					ErrorToReturn:    "ALREADY_EXISTS",
					OverrideToReturn: "",
					QueryPercent:     int32(30),
				},
				{
					ErrorToReturn:    "",
					OverrideToReturn: "{}",
					QueryPercent:     int32(40),
				},
			}

			var err error
			mapping, err = GetAlterationToPercentAffected(alterationSpecs, "endpointname")
			Expect(err).To(BeNil())
		})

		It("should create a mapping with 3 elements", func() {
			Expect(len(mapping)).To(Equal(3))
		})

		It("should create a mapping with correct configs", func() {
			var altCfg AlterationConfiguration

			altCfg = AlterationConfiguration{
				ErrorToReturn:    "CANCELED",
				OverrideToReturn: "",
			}
			pct_cancelled, ok_cancelled := mapping[altCfg]
			Expect(ok_cancelled).To(BeTrue())
			Expect(pct_cancelled).To(Equal(PercentSlot(20)))

			altCfg = AlterationConfiguration{
				ErrorToReturn:    "ALREADY_EXISTS",
				OverrideToReturn: "",
			}
			pct_exists, ok_exists := mapping[altCfg]
			Expect(ok_exists).To(BeTrue())
			Expect(pct_exists).To(Equal(PercentSlot(30)))

			altCfg = AlterationConfiguration{
				ErrorToReturn:    "",
				OverrideToReturn: "{}",
			}
			pct_emptyret, ok_emptyret := mapping[altCfg]
			Expect(ok_emptyret).To(BeTrue())
			Expect(pct_emptyret).To(Equal(PercentSlot(40)))
		})
	})

	Context("with one alterations with too many fields specified", func() {
		var err error

		BeforeEach(func() {
			alterationSpecs = []*pb.AlterationSpec{
				{
					ErrorToReturn:    "CANCELED",
					OverrideToReturn: "{}",
				},
			}

			_, err = GetAlterationToPercentAffected(alterationSpecs, "endpointname")
		})

		It("should fail", func() {
			Expect(err.Error()).To(Equal("Cannot execute SendDisruption without specifying either ErrorToReturn or OverrideToReturn for all target endpoints"))
		})
	})

	Context("with one alterations with too few fields specified", func() {
		var err error

		BeforeEach(func() {
			alterationSpecs = []*pb.AlterationSpec{
				{
					ErrorToReturn:    "",
					OverrideToReturn: "",
				},
			}

			_, err = GetAlterationToPercentAffected(alterationSpecs, "endpointname")
		})

		It("should fail", func() {
			Expect(err.Error()).To(Equal("Cannot execute SendDisruption where ErrorToReturn or OverrideToReturn are both specified for a target endpoints"))
		})
	})

	Context("with three alterations which are more than 100", func() {
		BeforeEach(func() {
			alterationSpecs = []*pb.AlterationSpec{
				{
					ErrorToReturn:    "CANCELED",
					OverrideToReturn: "",
					QueryPercent:     int32(50),
				},
				{
					ErrorToReturn:    "ALREADY_EXISTS",
					OverrideToReturn: "",
					QueryPercent:     int32(50),
				},
				{
					ErrorToReturn:    "",
					OverrideToReturn: "{}",
					QueryPercent:     int32(50),
				},
			}

			var err error
			mapping, err = GetAlterationToPercentAffected(alterationSpecs, "endpointname")
			Expect(err).To(BeNil())
		})

		It("should create a mapping with 3 elements", func() {
			Expect(len(mapping)).To(Equal(3))
		})

		It("should create a mapping with correct configs", func() {
			var altCfg AlterationConfiguration

			altCfg = AlterationConfiguration{
				ErrorToReturn:    "CANCELED",
				OverrideToReturn: "",
			}
			pct_cancelled, ok_cancelled := mapping[altCfg]
			Expect(ok_cancelled).To(BeTrue())
			Expect(pct_cancelled).To(Equal(PercentSlot(50)))

			altCfg = AlterationConfiguration{
				ErrorToReturn:    "ALREADY_EXISTS",
				OverrideToReturn: "",
			}
			pct_exists, ok_exists := mapping[altCfg]
			Expect(ok_exists).To(BeTrue())
			Expect(pct_exists).To(Equal(PercentSlot(50)))

			altCfg = AlterationConfiguration{
				ErrorToReturn:    "",
				OverrideToReturn: "{}",
			}

			// intuition here as that the service will never pick a number larger than 100 randomly
			// so this return value never gets triggered, but the function does not error out
			pct_emptyret, ok_emptyret := mapping[altCfg]
			Expect(ok_emptyret).To(BeTrue())
			Expect(pct_emptyret).To(Equal(PercentSlot(50)))
		})
	})

	Context("with one alteration less than 100", func() {
		BeforeEach(func() {
			alterationSpecs = []*pb.AlterationSpec{
				{
					ErrorToReturn:    "CANCELED",
					OverrideToReturn: "",
					QueryPercent:     int32(40),
				},
			}

			var err error
			mapping, err = GetAlterationToPercentAffected(alterationSpecs, "endpointname")
			Expect(err).To(BeNil())
		})
		It("should create a mapping with 1 element", func() {
			Expect(len(mapping)).To(Equal(1))
		})

		It("should create a mapping with correct configs", func() {
			var altCfg AlterationConfiguration

			altCfg = AlterationConfiguration{
				ErrorToReturn:    "CANCELED",
				OverrideToReturn: "",
			}
			pct_cancelled, ok_cancelled := mapping[altCfg]
			Expect(ok_cancelled).To(BeTrue())
			Expect(pct_cancelled).To(Equal(PercentSlot(40)))
		})
	})

	Context("with one alteration lacking query percent", func() {
		BeforeEach(func() {
			alterationSpecs = []*pb.AlterationSpec{
				{
					ErrorToReturn:    "CANCELED",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
			}

			var err error
			mapping, err = GetAlterationToPercentAffected(alterationSpecs, "endpointname")
			Expect(err).To(BeNil())
		})
		It("should create a mapping with 1 element", func() {
			Expect(len(mapping)).To(Equal(1))
		})

		It("should create a mapping with correct configs", func() {
			var altCfg AlterationConfiguration

			altCfg = AlterationConfiguration{
				ErrorToReturn:    "CANCELED",
				OverrideToReturn: "",
			}
			pct_cancelled, ok_cancelled := mapping[altCfg]
			Expect(ok_cancelled).To(BeTrue())
			Expect(pct_cancelled).To(Equal(PercentSlot(100)))
		})
	})

	Context("with three alterations, two of which lack a queryPercent", func() {
		BeforeEach(func() {
			alterationSpecs = []*pb.AlterationSpec{
				{
					ErrorToReturn:    "CANCELED",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					ErrorToReturn:    "ALREADY_EXISTS",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					ErrorToReturn:    "",
					OverrideToReturn: "{}",
					QueryPercent:     int32(50),
				},
			}

			var err error
			mapping, err = GetAlterationToPercentAffected(alterationSpecs, "endpointname")
			Expect(err).To(BeNil())
		})
		It("should create a mapping with 3 elements", func() {
			Expect(len(mapping)).To(Equal(3))
		})

		It("should create a mapping with correct configs", func() {
			var altCfg AlterationConfiguration

			altCfg = AlterationConfiguration{
				ErrorToReturn:    "CANCELED",
				OverrideToReturn: "",
			}
			pct_cancelled, ok_cancelled := mapping[altCfg]
			Expect(ok_cancelled).To(BeTrue())
			Expect(pct_cancelled).To(Equal(PercentSlot(25)))

			altCfg = AlterationConfiguration{
				ErrorToReturn:    "ALREADY_EXISTS",
				OverrideToReturn: "",
			}
			pct_exists, ok_exists := mapping[altCfg]
			Expect(ok_exists).To(BeTrue())
			Expect(pct_exists).To(Equal(PercentSlot(25)))

			altCfg = AlterationConfiguration{
				ErrorToReturn:    "",
				OverrideToReturn: "{}",
			}
			pct_emptyret, ok_emptyret := mapping[altCfg]
			Expect(ok_emptyret).To(BeTrue())
			Expect(pct_emptyret).To(Equal(PercentSlot(50)))
		})
	})
	Context("with three alterations, two of which lack a queryPercent", func() {
		BeforeEach(func() {
			alterationSpecs = []*pb.AlterationSpec{
				{
					ErrorToReturn:    "CANCELED",
					OverrideToReturn: "",
					QueryPercent:     int32(50),
				},
				{
					ErrorToReturn:    "ALREADY_EXISTS",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					ErrorToReturn:    "UNKNOWN",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					ErrorToReturn:    "INVALID_ARGUMENT",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					ErrorToReturn:    "DEADLINE_EXCEEDED",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					ErrorToReturn:    "NOT_FOUND",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					ErrorToReturn:    "PERMISSION_DENIED",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
			}

			var err error
			mapping, err = GetAlterationToPercentAffected(alterationSpecs, "endpointname")
			Expect(err).To(BeNil())
		})
		It("should create a mapping with 7 elements", func() {
			Expect(len(mapping)).To(Equal(7))
		})

		It("should create a mapping with correct configs", func() {
			var (
				altCfg AlterationConfiguration
				pct    PercentAffected
				ok     bool
			)

			altCfg = AlterationConfiguration{
				ErrorToReturn:    "CANCELED",
				OverrideToReturn: "",
			}
			pct, ok = mapping[altCfg]
			Expect(ok).To(BeTrue())
			Expect(pct).To(Equal(PercentSlot(90)))

			altCfg = AlterationConfiguration{
				ErrorToReturn:    "ALREADY_EXISTS",
				OverrideToReturn: "",
			}
			pct, ok = mapping[altCfg]
			Expect(ok).To(BeTrue())
			Expect(pct).To(Equal(PercentSlot(1)))

			altCfg = AlterationConfiguration{
				ErrorToReturn:    "UNKNOWN",
				OverrideToReturn: "",
			}
			pct, ok = mapping[altCfg]
			Expect(ok).To(BeTrue())
			Expect(pct).To(Equal(PercentSlot(1)))

			altCfg = AlterationConfiguration{
				ErrorToReturn:    "INVALID_ARGUMENT",
				OverrideToReturn: "",
			}
			pct, ok = mapping[altCfg]
			Expect(ok).To(BeTrue())
			Expect(pct).To(Equal(PercentSlot(1)))

			altCfg = AlterationConfiguration{
				ErrorToReturn:    "DEADLINE_EXCEEDED",
				OverrideToReturn: "",
			}
			pct, ok = mapping[altCfg]
			Expect(ok).To(BeTrue())
			Expect(pct).To(Equal(PercentSlot(1)))

			altCfg = AlterationConfiguration{
				ErrorToReturn:    "NOT_FOUND",
				OverrideToReturn: "",
			}
			pct, ok = mapping[altCfg]
			Expect(ok).To(BeTrue())
			Expect(pct).To(Equal(PercentSlot(1)))

			altCfg = AlterationConfiguration{
				ErrorToReturn:    "PERMISSION_DENIED",
				OverrideToReturn: "",
			}
			pct, ok = mapping[altCfg]
			Expect(ok).To(BeTrue())
			Expect(pct).To(Equal(PercentSlot(5)))
		})
	})

	Context("with three alterations, two of which lack a queryPercent", func() {
		BeforeEach(func() {
			alterationSpecs = []*pb.AlterationSpec{
				{
					ErrorToReturn:    "CANCELED",
					OverrideToReturn: "",
					QueryPercent:     int32(50),
				},
				{
					ErrorToReturn:    "ALREADY_EXISTS",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					ErrorToReturn:    "UNKNOWN",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					ErrorToReturn:    "INVALID_ARGUMENT",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					ErrorToReturn:    "DEADLINE_EXCEEDED",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					ErrorToReturn:    "NOT_FOUND",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					ErrorToReturn:    "PERMISSION_DENIED",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					ErrorToReturn:    "RESOURCE_EXHAUSTED",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					ErrorToReturn:    "FAILED_PRECONDITION",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					ErrorToReturn:    "ABORTED",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					ErrorToReturn:    "OUT_OF_RANGE",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
				{
					ErrorToReturn:    "UNIMPLEMENTED",
					OverrideToReturn: "",
					QueryPercent:     0,
				},
			}

			var err error
			mapping, err = GetAlterationToPercentAffected(alterationSpecs, "endpointname")
			Expect(err).To(BeNil())
		})

		It("should create a mapping with 12 elements", func() {
			Expect(len(mapping)).To(Equal(12))
		})

		It("should create a mapping with correct configs", func() {
			var (
				altCfg AlterationConfiguration
				pct    PercentAffected
				ok     bool
			)

			altCfg = AlterationConfiguration{
				ErrorToReturn:    "CANCELED",
				OverrideToReturn: "",
			}
			pct, ok = mapping[altCfg]
			Expect(ok).To(BeTrue())
			Expect(pct).To(Equal(PercentSlot(90)))

			altCfg = AlterationConfiguration{
				ErrorToReturn:    "ALREADY_EXISTS",
				OverrideToReturn: "",
			}
			pct, ok = mapping[altCfg]
			Expect(ok).To(BeTrue())
			Expect(pct).To(Equal(PercentSlot(0)))

			altCfg = AlterationConfiguration{
				ErrorToReturn:    "UNKNOWN",
				OverrideToReturn: "",
			}
			pct, ok = mapping[altCfg]
			Expect(ok).To(BeTrue())
			Expect(pct).To(Equal(PercentSlot(0)))

			altCfg = AlterationConfiguration{
				ErrorToReturn:    "INVALID_ARGUMENT",
				OverrideToReturn: "",
			}
			pct, ok = mapping[altCfg]
			Expect(ok).To(BeTrue())
			Expect(pct).To(Equal(PercentSlot(0)))

			altCfg = AlterationConfiguration{
				ErrorToReturn:    "DEADLINE_EXCEEDED",
				OverrideToReturn: "",
			}
			pct, ok = mapping[altCfg]
			Expect(ok).To(BeTrue())
			Expect(pct).To(Equal(PercentSlot(0)))

			altCfg = AlterationConfiguration{
				ErrorToReturn:    "NOT_FOUND",
				OverrideToReturn: "",
			}
			pct, ok = mapping[altCfg]
			Expect(ok).To(BeTrue())
			Expect(pct).To(Equal(PercentSlot(0)))

			altCfg = AlterationConfiguration{
				ErrorToReturn:    "PERMISSION_DENIED",
				OverrideToReturn: "",
			}
			pct, ok = mapping[altCfg]
			Expect(ok).To(BeTrue())
			Expect(pct).To(Equal(PercentSlot(0)))

			altCfg = AlterationConfiguration{
				ErrorToReturn:    "RESOURCE_EXHAUSTED",
				OverrideToReturn: "",
			}
			pct, ok = mapping[altCfg]
			Expect(ok).To(BeTrue())
			Expect(pct).To(Equal(PercentSlot(0)))

			altCfg = AlterationConfiguration{
				ErrorToReturn:    "FAILED_PRECONDITION",
				OverrideToReturn: "",
			}
			pct, ok = mapping[altCfg]
			Expect(ok).To(BeTrue())
			Expect(pct).To(Equal(PercentSlot(0)))

			altCfg = AlterationConfiguration{
				ErrorToReturn:    "ABORTED",
				OverrideToReturn: "",
			}
			pct, ok = mapping[altCfg]
			Expect(ok).To(BeTrue())
			Expect(pct).To(Equal(PercentSlot(0)))

			altCfg = AlterationConfiguration{
				ErrorToReturn:    "OUT_OF_RANGE",
				OverrideToReturn: "",
			}
			pct, ok = mapping[altCfg]
			Expect(ok).To(BeTrue())
			Expect(pct).To(Equal(PercentSlot(0)))

			altCfg = AlterationConfiguration{
				ErrorToReturn:    "UNIMPLEMENTED",
				OverrideToReturn: "",
			}
			pct, ok = mapping[altCfg]
			Expect(ok).To(BeTrue())
			Expect(pct).To(Equal(PercentSlot(10)))
		})
	})
})
