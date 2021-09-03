// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package grpc_test

import (
	. "github.com/DataDog/chaos-controller/grpc"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("get mapping from randomly generated Percent to Alteration based on AlterationToPercentAffected", func() {
	var (
		altCfgs             []AlterationConfiguration
		mapping             map[AlterationConfiguration]PercentAffected
		percentToAlteration []AlterationConfiguration
	)
	BeforeEach(func() {
		altCfgs = []AlterationConfiguration{
			{
				ErrorToReturn:    "CANCELED",
				OverrideToReturn: "",
			},
			{
				ErrorToReturn:    "ALREADY_EXISTS",
				OverrideToReturn: "",
			},
			{
				ErrorToReturn:    "",
				OverrideToReturn: "{}",
			},
		}
		mapping = make(map[AlterationConfiguration]PercentAffected)
	})

	Context("with one alteration", func() {
		BeforeEach(func() {
			mapping[altCfgs[0]] = PercentAffected(15)

			percentToAlteration = GetPercentToAlteration(mapping)
		})

		It("should create 15 SlotPercents", func() {
			Expect(len(percentToAlteration)).To(Equal(15))
		})

		It("should create 15 Slot Percents which are configured to the single alteration", func() {
			altCfg_0 := percentToAlteration[0]
			Expect(altCfg_0.ErrorToReturn).To(Equal("CANCELED"))
			Expect(altCfg_0.OverrideToReturn).To(Equal(""))

			altCfg_14 := percentToAlteration[14]
			Expect(altCfg_14.ErrorToReturn).To(Equal("CANCELED"))
			Expect(altCfg_14.OverrideToReturn).To(Equal(""))
		})
	})

	Context("with three alterations", func() {
		BeforeEach(func() {
			mapping[altCfgs[0]] = PercentAffected(15)
			mapping[altCfgs[1]] = PercentAffected(20)
			mapping[altCfgs[2]] = PercentAffected(35)

			percentToAlteration = GetPercentToAlteration(mapping)
		})

		It("should create 70 SlotPercents", func() {
			Expect(len(percentToAlteration)).To(Equal(70))
		})

		It("should create 15 Slot Percents which are configured to the first alteration", func() {
			altCfg_0 := percentToAlteration[0]
			Expect(altCfg_0.ErrorToReturn).To(Equal("CANCELED"))
			Expect(altCfg_0.OverrideToReturn).To(Equal(""))

			altCfg_14 := percentToAlteration[0]
			Expect(altCfg_14.ErrorToReturn).To(Equal("CANCELED"))
			Expect(altCfg_14.OverrideToReturn).To(Equal(""))
		})

		It("should create 20 Slot Percents which are configured to the second alteration", func() {
			altCfg_15 := percentToAlteration[15]
			Expect(altCfg_15.ErrorToReturn).To(Equal("ALREADY_EXISTS"))
			Expect(altCfg_15.OverrideToReturn).To(Equal(""))

			altCfg_34 := percentToAlteration[34]
			Expect(altCfg_34.ErrorToReturn).To(Equal("ALREADY_EXISTS"))
			Expect(altCfg_34.OverrideToReturn).To(Equal(""))
		})

		It("should create 35 Slot Percents which are configured to the third alteration", func() {
			altCfg_35 := percentToAlteration[35]
			Expect(altCfg_35.ErrorToReturn).To(Equal(""))
			Expect(altCfg_35.OverrideToReturn).To(Equal("{}"))

			altCfg_69 := percentToAlteration[60]
			Expect(altCfg_69.ErrorToReturn).To(Equal(""))
			Expect(altCfg_69.OverrideToReturn).To(Equal("{}"))
		})
	})
	Context("with three alterations that add to 100", func() {
		BeforeEach(func() {
			mapping[altCfgs[0]] = PercentAffected(40)
			mapping[altCfgs[1]] = PercentAffected(40)
			mapping[altCfgs[2]] = PercentAffected(20)

			percentToAlteration = GetPercentToAlteration(mapping)
		})

		It("should create 100 SlotPercents", func() {
			Expect(len(percentToAlteration)).To(Equal(100))
		})

		It("should create Slot Percents which are configured to the right alterations", func() {
			altCfg_35 := percentToAlteration[35]
			Expect(altCfg_35.ErrorToReturn).To(Equal("CANCELED"))
			Expect(altCfg_35.OverrideToReturn).To(Equal(""))

			altCfg_60 := percentToAlteration[60]
			Expect(altCfg_60.ErrorToReturn).To(Equal("ALREADY_EXISTS"))
			Expect(altCfg_60.OverrideToReturn).To(Equal(""))

			altCfg_99 := percentToAlteration[60]
			Expect(altCfg_99.ErrorToReturn).To(Equal(""))
			Expect(altCfg_99.OverrideToReturn).To(Equal("{}"))
		})
	})
})
