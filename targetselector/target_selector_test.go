// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package targetselector_test

import (
	"github.com/DataDog/chaos-controller/targetselector"
	"k8s.io/apimachinery/pkg/labels"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Label Selector Validation", func() {
	Context("validating an empty label selector", func() {
		It("should succeed", func() {
			selector := labels.Set{}
			Expect(targetselector.ValidateLabelSelector(selector.AsSelector())).ToNot(Succeed())
		})
	})
	Context("validating a good label selector", func() {
		It("should succeed", func() {
			selector := labels.Set{"foo": "bar"}
			Expect(targetselector.ValidateLabelSelector(selector.AsSelector())).To(Succeed())
		})
	})
	Context("validating special characters in label selector", func() {
		It("should succeed", func() {
			selector := labels.Set{"foo": "”bar”"}
			//.AsSelector() should strip invalid characters
			Expect(targetselector.ValidateLabelSelector(selector.AsSelector())).To(Succeed())
		})
	})
	Context("validating too many quotes in label selector", func() {
		It("should succeed", func() {
			selector := labels.Set{"foo": "\"bar\""}
			//.AsSelector() should strip invalid characters
			Expect(targetselector.ValidateLabelSelector(selector.AsSelector())).To(Succeed())
		})
	})
})
