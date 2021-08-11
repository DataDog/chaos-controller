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

var _ = Describe("get mapping from PercentSlot to Alteration based on AlterationToPercentAffected", func() {
	var (
		altCfgs                 []AlterationConfiguration
		mapping                 map[AlterationConfiguration]PercentAffected
		percentSlotToAlteration map[PercentSlot]AlterationConfiguration
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

			var err error
			percentSlotToAlteration, err = GetPercentSlotToAlteration(mapping)
			Expect(err).To(BeNil())
		})

		It("should create 15 SlotPercents", func() {
			Expect(len(percentSlotToAlteration)).To(Equal(15))

			_, ok_15 := percentSlotToAlteration[PercentSlot(15)]
			Expect(ok_15).To(BeFalse())
		})

		It("should create 15 Slot Percents which are configured to the single alteration", func() {
			altCfg_0, ok_0 := percentSlotToAlteration[PercentSlot(0)]
			Expect(ok_0).To(BeTrue())
			Expect(altCfg_0.ErrorToReturn).To(Equal("CANCELED"))
			Expect(altCfg_0.OverrideToReturn).To(Equal(""))

			altCfg_14, ok_14 := percentSlotToAlteration[PercentSlot(14)]
			Expect(ok_14).To(BeTrue())
			Expect(altCfg_14.ErrorToReturn).To(Equal("CANCELED"))
			Expect(altCfg_14.OverrideToReturn).To(Equal(""))
		})
	})

	Context("with three alterations", func() {
		BeforeEach(func() {
			mapping[altCfgs[0]] = PercentAffected(15)
			mapping[altCfgs[1]] = PercentAffected(20)
			mapping[altCfgs[2]] = PercentAffected(35)

			var err error
			percentSlotToAlteration, err = GetPercentSlotToAlteration(mapping)
			Expect(err).To(BeNil())
		})

		It("should create 70 SlotPercents", func() {
			Expect(len(percentSlotToAlteration)).To(Equal(70))

			_, ok_70 := percentSlotToAlteration[PercentSlot(70)]
			Expect(ok_70).To(BeFalse())
		})

		It("should create 15 Slot Percents which are configured to the first alteration", func() {
			altCfg_0, ok_0 := percentSlotToAlteration[PercentSlot(0)]
			Expect(ok_0).To(BeTrue())
			Expect(altCfg_0.ErrorToReturn).To(Equal("CANCELED"))
			Expect(altCfg_0.OverrideToReturn).To(Equal(""))

			altCfg_14, ok_14 := percentSlotToAlteration[PercentSlot(0)]
			Expect(ok_14).To(BeTrue())
			Expect(altCfg_14.ErrorToReturn).To(Equal("CANCELED"))
			Expect(altCfg_14.OverrideToReturn).To(Equal(""))
		})

		It("should create 20 Slot Percents which are configured to the second alteration", func() {
			altCfg_15, ok_15 := percentSlotToAlteration[PercentSlot(15)]
			Expect(ok_15).To(BeTrue())
			Expect(altCfg_15.ErrorToReturn).To(Equal("ALREADY_EXISTS"))
			Expect(altCfg_15.OverrideToReturn).To(Equal(""))

			altCfg_34, ok_34 := percentSlotToAlteration[PercentSlot(34)]
			Expect(ok_34).To(BeTrue())
			Expect(altCfg_34.ErrorToReturn).To(Equal("ALREADY_EXISTS"))
			Expect(altCfg_34.OverrideToReturn).To(Equal(""))
		})

		It("should create 35 Slot Percents which are configured to the third alteration", func() {
			altCfg_35, ok_35 := percentSlotToAlteration[PercentSlot(35)]
			Expect(ok_35).To(BeTrue())
			Expect(altCfg_35.ErrorToReturn).To(Equal(""))
			Expect(altCfg_35.OverrideToReturn).To(Equal("{}"))

			altCfg_69, ok_69 := percentSlotToAlteration[PercentSlot(60)]
			Expect(ok_69).To(BeTrue())
			Expect(altCfg_69.ErrorToReturn).To(Equal(""))
			Expect(altCfg_69.OverrideToReturn).To(Equal("{}"))
		})
	})
	Context("with three alterations that add to 100", func() {
		BeforeEach(func() {
			mapping[altCfgs[0]] = PercentAffected(40)
			mapping[altCfgs[1]] = PercentAffected(40)
			mapping[altCfgs[2]] = PercentAffected(20)

			var err error
			percentSlotToAlteration, err = GetPercentSlotToAlteration(mapping)
			Expect(err).To(BeNil())
		})

		It("should create 100 SlotPercents", func() {
			Expect(len(percentSlotToAlteration)).To(Equal(100))

			_, ok_100 := percentSlotToAlteration[PercentSlot(100)]
			Expect(ok_100).To(BeFalse())
		})

		It("should create Slot Percents which are configured to the right alterations", func() {
			altCfg_35, ok_35 := percentSlotToAlteration[PercentSlot(35)]
			Expect(ok_35).To(BeTrue())
			Expect(altCfg_35.ErrorToReturn).To(Equal("CANCELED"))
			Expect(altCfg_35.OverrideToReturn).To(Equal(""))

			altCfg_60, ok_60 := percentSlotToAlteration[PercentSlot(60)]
			Expect(ok_60).To(BeTrue())
			Expect(altCfg_60.ErrorToReturn).To(Equal("ALREADY_EXISTS"))
			Expect(altCfg_60.OverrideToReturn).To(Equal(""))

			altCfg_99, ok_99 := percentSlotToAlteration[PercentSlot(60)]
			Expect(ok_99).To(BeTrue())
			Expect(altCfg_99.ErrorToReturn).To(Equal(""))
			Expect(altCfg_99.OverrideToReturn).To(Equal("{}"))
		})
	})
})
