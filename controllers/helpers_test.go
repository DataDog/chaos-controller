// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package controllers

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/labels"
)

var _ = Describe("Label Selector Validation", func() {
	Context("validating an empty label selector", func() {
		It("", func() {
			selector := labels.Set{}
			Expect(validateLabelSelector(selector.AsSelector())).ToNot(BeNil())
		})
	})
	Context("validating a good label selector", func() {
		It("", func() {
			selector := labels.Set{"foo": "bar"}
			Expect(validateLabelSelector(selector.AsSelector())).To(BeNil())
		})
	})
	Context("validating special characters in label selector", func() {
		It("", func() {
			selector := labels.Set{"foo": "”bar”"}
			//.AsSelector() should strip invalid characters
			Expect(validateLabelSelector(selector.AsSelector())).To(BeNil())
		})
	})
	Context("validating too many quotes in label selector", func() {
		It("", func() {
			selector := labels.Set{"foo": "\"bar\""}
			//.AsSelector() should strip invalid characters
			Expect(validateLabelSelector(selector.AsSelector())).To(BeNil())
		})
	})
})
