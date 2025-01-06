// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package v1beta1_test

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("ValidateCount", func() {
	Describe("Success", func() {
		Context("when count is a percentage", func() {
			It("should return nil", func() {
				err := v1beta1.ValidateCount(&intstr.IntOrString{Type: intstr.String, StrVal: "50%"})
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when count is an integer", func() {
			It("should return nil", func() {
				err := v1beta1.ValidateCount(&intstr.IntOrString{Type: intstr.String, StrVal: "2"})
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("Failure", func() {
		Context("when count is not a percentage or an integer", func() {
			It("should return an error", func() {
				err := v1beta1.ValidateCount(&intstr.IntOrString{Type: intstr.String, StrVal: "foo"})
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when count is a negative integer", func() {
			It("should return an error", func() {
				err := v1beta1.ValidateCount(&intstr.IntOrString{Type: intstr.String, StrVal: "-1"})
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when count is a negative percentage", func() {
			It("should return an error", func() {
				err := v1beta1.ValidateCount(&intstr.IntOrString{Type: intstr.String, StrVal: "-1%"})
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when count is a percentage greater than 100", func() {
			It("should return an error", func() {
				err := v1beta1.ValidateCount(&intstr.IntOrString{Type: intstr.String, StrVal: "101%"})
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when count is nil", func() {
			It("should return an error", func() {
				err := v1beta1.ValidateCount(nil)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
