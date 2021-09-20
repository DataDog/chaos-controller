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
		It("should create 15 (out of 100 potential) slots with alteration configurations", func() {
			mapping[altCfgs[0]] = PercentAffected(15)

			percentToAlteration = GetPercentToAlteration(mapping, logger)

			Expect(len(percentToAlteration)).To(Equal(15))

			altCfg := percentToAlteration[0]
			Expect(altCfg.ErrorToReturn).To(Equal("CANCELED"))
			Expect(altCfg.OverrideToReturn).To(Equal(""))

			altCfg = percentToAlteration[14]
			Expect(altCfg.ErrorToReturn).To(Equal("CANCELED"))
			Expect(altCfg.OverrideToReturn).To(Equal(""))
		})
	})
	Context("with three alterations that add up to less than 100 percent", func() {
		It("should create 70 (out of 100 potential) slots with alteration configurations", func() {
			mapping[altCfgs[0]] = PercentAffected(15)
			mapping[altCfgs[1]] = PercentAffected(20)
			mapping[altCfgs[2]] = PercentAffected(35)

			percentToAlteration = GetPercentToAlteration(mapping, logger)

			Expect(len(percentToAlteration)).To(Equal(70))

			position := 0
			altCfg := percentToAlteration[position]

			for i := 0; i < 3; i++ {
				if altCfg.ErrorToReturn == "CANCELED" {
					Expect(altCfg.OverrideToReturn).To(Equal(""))

					position = position + 14
					altCfg = percentToAlteration[position]
					Expect(altCfg.ErrorToReturn).To(Equal("CANCELED"))
					Expect(altCfg.OverrideToReturn).To(Equal(""))

					position = position + 1

					if position < 70 {
						altCfg = percentToAlteration[position]
					}
				} else if altCfg.ErrorToReturn == "ALREADY_EXISTS" {
					Expect(altCfg.OverrideToReturn).To(Equal(""))

					position = position + 19
					altCfg = percentToAlteration[position]
					Expect(altCfg.ErrorToReturn).To(Equal("ALREADY_EXISTS"))
					Expect(altCfg.OverrideToReturn).To(Equal(""))

					position = position + 1

					if position < 70 {
						altCfg = percentToAlteration[position]
					}
				} else {
					Expect(altCfg.ErrorToReturn).To(Equal(""))
					Expect(altCfg.OverrideToReturn).To(Equal("{}"))

					position = position + 34
					altCfg = percentToAlteration[position]
					Expect(altCfg.ErrorToReturn).To(Equal(""))
					Expect(altCfg.OverrideToReturn).To(Equal("{}"))

					position = position + 1

					if position < 70 {
						altCfg = percentToAlteration[position]
					}
				}
			}
		})
	})
	Context("with three alterations that add to 100", func() {
		It("should create 100 (out of 100 potential) slots with alteration configurations", func() {
			mapping[altCfgs[0]] = PercentAffected(40)
			mapping[altCfgs[1]] = PercentAffected(40)
			mapping[altCfgs[2]] = PercentAffected(20)

			percentToAlteration = GetPercentToAlteration(mapping, logger)

			Expect(len(percentToAlteration)).To(Equal(100))

			position := 0
			altCfg := percentToAlteration[position]

			for i := 0; i < 3; i++ {
				if altCfg.ErrorToReturn == "CANCELED" {
					Expect(altCfg.OverrideToReturn).To(Equal(""))

					position = position + 39
					altCfg = percentToAlteration[position]
					Expect(altCfg.ErrorToReturn).To(Equal("CANCELED"))
					Expect(altCfg.OverrideToReturn).To(Equal(""))

					position = position + 1

					if position < 100 {
						altCfg = percentToAlteration[position]
					}
				} else if altCfg.ErrorToReturn == "ALREADY_EXISTS" {
					Expect(altCfg.OverrideToReturn).To(Equal(""))

					position = position + 39
					altCfg = percentToAlteration[position]
					Expect(altCfg.ErrorToReturn).To(Equal("ALREADY_EXISTS"))
					Expect(altCfg.OverrideToReturn).To(Equal(""))

					position = position + 1

					if position < 100 {
						altCfg = percentToAlteration[position]
					}
				} else {
					Expect(altCfg.ErrorToReturn).To(Equal(""))
					Expect(altCfg.OverrideToReturn).To(Equal("{}"))

					position = position + 19
					altCfg = percentToAlteration[position]
					Expect(altCfg.ErrorToReturn).To(Equal(""))
					Expect(altCfg.OverrideToReturn).To(Equal("{}"))

					position = position + 1

					if position < 100 {
						altCfg = percentToAlteration[position]
					}
				}
			}
		})
	})
})
