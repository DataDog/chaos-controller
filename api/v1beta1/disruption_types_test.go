// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package v1beta1_test

import (
	"sort"

	. "github.com/DataDog/chaos-controller/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TargetInjections", func() {
	Describe("GetTargetNames", func() {
		var targetInjections TargetInjections

		Context("with three targets", func() {
			BeforeEach(func() {
				targetInjections = TargetInjections{"target-1": {}, "target-2": {}, "target-3": {}}
			})

			It("should return the list of targets name", func() {
				targetNames := targetInjections.GetTargetNames()
				sort.Strings(targetNames)
				Expect(targetNames).Should(BeEquivalentTo([]string{"target-1", "target-2", "target-3"}))
			})
		})

		Context("without targets", func() {
			BeforeEach(func() {
				targetInjections = TargetInjections{}
			})

			It("should return the list of targets name", func() {
				Expect(targetInjections.GetTargetNames()).Should(BeEquivalentTo([]string{}))
			})
		})
	})
})

var _ = Describe("Check if a target exist into DisruptionStatus targets list", func() {
	var disruptionStatus DisruptionStatus

	BeforeEach(func() {
		disruptionStatus = DisruptionStatus{
			TargetInjections: TargetInjections{"test-1": {}},
		}
	})

	Context("with an empty target", func() {
		It("should return false", func() {
			Expect(disruptionStatus.HasTarget("")).Should(BeFalse())
		})
	})

	Context("with an existing target", func() {
		It("should return true", func() {
			Expect(disruptionStatus.HasTarget("test-1")).Should(BeTrue())
		})
	})

	Context("with an non existing target", func() {
		It("should return false", func() {
			Expect(disruptionStatus.HasTarget("test-2")).Should(BeFalse())
		})
	})
})
