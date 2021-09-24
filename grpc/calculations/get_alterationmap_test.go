// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package calculations_test

import (
	. "github.com/DataDog/chaos-controller/grpc/calculations"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("get mapping from randomly generated Percent to Alteration based on GetMappingFromIndividualalterationMap", func() {
	var (
		altCfgs                        []AlterationConfiguration
		alterationConfigToQueryPercent map[AlterationConfiguration]QueryPercent
		alterationMap                  []AlterationConfiguration
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
		alterationConfigToQueryPercent = make(map[AlterationConfiguration]QueryPercent)
	})

	Context("with one alteration", func() {
		It("should create 15 (out of 100 potential) slots with alteration configurations", func() {
			alterationConfigToQueryPercent[altCfgs[0]] = QueryPercent(15)

			alterationMap = ConvertQueryPercentByAltConfigToAlterationMap(alterationConfigToQueryPercent)

			Expect(len(alterationMap)).To(Equal(15))

			altCfg := alterationMap[0]
			Expect(altCfg.ErrorToReturn).To(Equal("CANCELED"))
			Expect(altCfg.OverrideToReturn).To(Equal(""))

			altCfg = alterationMap[14]
			Expect(altCfg.ErrorToReturn).To(Equal("CANCELED"))
			Expect(altCfg.OverrideToReturn).To(Equal(""))
		})
	})
	Context("with three alterations that add up to less than 100 percent", func() {
		It("should create 70 (out of 100 potential) slots with alteration configurations", func() {
			alterationConfigToQueryPercent[altCfgs[0]] = QueryPercent(15)
			alterationConfigToQueryPercent[altCfgs[1]] = QueryPercent(20)
			alterationConfigToQueryPercent[altCfgs[2]] = QueryPercent(35)

			alterationMap = ConvertQueryPercentByAltConfigToAlterationMap(alterationConfigToQueryPercent)

			Expect(len(alterationMap)).To(Equal(70))

			position := 0
			altCfg := alterationMap[position]

			for i := 0; i < 3; i++ {
				if altCfg.ErrorToReturn == "CANCELED" {
					Expect(altCfg.OverrideToReturn).To(Equal(""))

					position = position + 14
					altCfg = alterationMap[position]
					Expect(altCfg.ErrorToReturn).To(Equal("CANCELED"))
					Expect(altCfg.OverrideToReturn).To(Equal(""))

					position = position + 1

					if position < 70 {
						altCfg = alterationMap[position]
					}
				} else if altCfg.ErrorToReturn == "ALREADY_EXISTS" {
					Expect(altCfg.OverrideToReturn).To(Equal(""))

					position = position + 19
					altCfg = alterationMap[position]
					Expect(altCfg.ErrorToReturn).To(Equal("ALREADY_EXISTS"))
					Expect(altCfg.OverrideToReturn).To(Equal(""))

					position = position + 1

					if position < 70 {
						altCfg = alterationMap[position]
					}
				} else {
					Expect(altCfg.ErrorToReturn).To(Equal(""))
					Expect(altCfg.OverrideToReturn).To(Equal("{}"))

					position = position + 34
					altCfg = alterationMap[position]
					Expect(altCfg.ErrorToReturn).To(Equal(""))
					Expect(altCfg.OverrideToReturn).To(Equal("{}"))

					position = position + 1

					if position < 70 {
						altCfg = alterationMap[position]
					}
				}
			}
		})
	})
	Context("with three alterations that add to 100", func() {
		It("should create 100 (out of 100 potential) slots with alteration configurations", func() {
			alterationConfigToQueryPercent[altCfgs[0]] = QueryPercent(40)
			alterationConfigToQueryPercent[altCfgs[1]] = QueryPercent(40)
			alterationConfigToQueryPercent[altCfgs[2]] = QueryPercent(20)

			alterationMap = ConvertQueryPercentByAltConfigToAlterationMap(alterationConfigToQueryPercent)

			Expect(len(alterationMap)).To(Equal(100))

			position := 0
			altCfg := alterationMap[position]

			for i := 0; i < 3; i++ {
				if altCfg.ErrorToReturn == "CANCELED" {
					Expect(altCfg.OverrideToReturn).To(Equal(""))

					position = position + 39
					altCfg = alterationMap[position]
					Expect(altCfg.ErrorToReturn).To(Equal("CANCELED"))
					Expect(altCfg.OverrideToReturn).To(Equal(""))

					position = position + 1

					if position < 100 {
						altCfg = alterationMap[position]
					}
				} else if altCfg.ErrorToReturn == "ALREADY_EXISTS" {
					Expect(altCfg.OverrideToReturn).To(Equal(""))

					position = position + 39
					altCfg = alterationMap[position]
					Expect(altCfg.ErrorToReturn).To(Equal("ALREADY_EXISTS"))
					Expect(altCfg.OverrideToReturn).To(Equal(""))

					position = position + 1

					if position < 100 {
						altCfg = alterationMap[position]
					}
				} else {
					Expect(altCfg.ErrorToReturn).To(Equal(""))
					Expect(altCfg.OverrideToReturn).To(Equal("{}"))

					position = position + 19
					altCfg = alterationMap[position]
					Expect(altCfg.ErrorToReturn).To(Equal(""))
					Expect(altCfg.OverrideToReturn).To(Equal("{}"))

					position = position + 1

					if position < 100 {
						altCfg = alterationMap[position]
					}
				}
			}
		})
	})
})
