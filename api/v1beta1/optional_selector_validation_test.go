// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package v1beta1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("Optional selector validation", func() {
	var spec DisruptionSpec

	BeforeEach(func() {
		count := intstr.FromInt(1)
		spec = DisruptionSpec{
			Count: &count, Level: "pod", Duration: "1m",
			Network: &NetworkDisruptionSpec{Drop: 100},
		}
	})

	DescribeTable("rejects malformed selectors even when selectors are optional", func(selector labels.Set, advanced []metav1.LabelSelectorRequirement) {
		spec.Selector = selector
		spec.AdvancedSelector = advanced
		Expect(spec.ValidateSelectorsOptional(false)).NotTo(Succeed())
		cron := &DisruptionCron{Spec: DisruptionCronSpec{Schedule: "@every 1h", DisruptionTemplate: spec}}
		Expect(cron.validateDisruptionCronSpec()).NotTo(Succeed())
	},
		Entry("invalid simple-selector key", labels.Set{"app/key/extra": "value"}, nil),
		Entry("invalid advanced-selector operator", nil, []metav1.LabelSelectorRequirement{{Key: "app", Operator: "Bogus", Values: []string{"example"}}}),
		Entry("invalid Exists values", nil, []metav1.LabelSelectorRequirement{{Key: "app", Operator: metav1.LabelSelectorOpExists, Values: []string{"example"}}}),
	)

	It("still permits omitted selectors only when they are optional", func() {
		Expect(spec.ValidateSelectorsOptional(false)).To(Succeed())
		Expect(spec.Validate()).NotTo(Succeed())
	})

	It("accepts valid simple and advanced selectors together", func() {
		spec.Selector = labels.Set{"app": "example"}
		spec.AdvancedSelector = []metav1.LabelSelectorRequirement{{Key: "tier", Operator: metav1.LabelSelectorOpIn, Values: []string{"frontend"}}}
		Expect(spec.ValidateSelectorsOptional(false)).To(Succeed())
		Expect(spec.Validate()).To(Succeed())
	})
})
