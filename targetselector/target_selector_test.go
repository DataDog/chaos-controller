// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package targetselector_test

import (
	"github.com/DataDog/chaos-controller/targetselector"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/labels"
)

var _ = ginkgo.Describe("Label Selector Validation", func() {
	ginkgo.Context("validating an empty label selector", func() {
		ginkgo.It("should succeed", func() {
			selector := labels.Set{}
			gomega.Expect(targetselector.ValidateLabelSelector(selector.AsSelector())).ToNot(gomega.Succeed())
		})
	})
	ginkgo.Context("validating a good label selector", func() {
		ginkgo.It("should succeed", func() {
			selector := labels.Set{"foo": "bar"}
			gomega.Expect(targetselector.ValidateLabelSelector(selector.AsSelector())).To(gomega.Succeed())
		})
	})
	ginkgo.Context("validating special characters in label selector", func() {
		ginkgo.It("should succeed", func() {
			selector := labels.Set{"foo": "”bar”"}
			//.AsSelector() should strip invalid characters
			gomega.Expect(targetselector.ValidateLabelSelector(selector.AsSelector())).To(gomega.Succeed())
		})
	})
	ginkgo.Context("validating too many quotes in label selector", func() {
		ginkgo.It("should succeed", func() {
			selector := labels.Set{"foo": "\"bar\""}
			//.AsSelector() should strip invalid characters
			gomega.Expect(targetselector.ValidateLabelSelector(selector.AsSelector())).To(gomega.Succeed())
		})
	})
})
