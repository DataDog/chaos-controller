// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package ddmark_test

import (
	"math/rand"
	"reflect"

	. "github.com/DataDog/chaos-controller/ddmark/validation"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Validation Rules Cases", func() {

	Context("Maximum test", func() {
		maxInt := rand.Intn(1000)
		max := Maximum(maxInt)

		It("checks superior values", func() {
			Expect(max.ApplyRule(reflect.ValueOf(maxInt + 1))).ToNot(BeNil())
		})
		It("checks exact value", func() {
			Expect(max.ApplyRule(reflect.ValueOf(maxInt))).To(BeNil())
		})
		It("checks inferior value", func() {
			Expect(max.ApplyRule(reflect.ValueOf(maxInt - 1))).To(BeNil())
		})
	})

	Context("Minimum test", func() {
		minInt := rand.Intn(1000)
		min := Minimum(minInt)
		It("checks superior value", func() {
			Expect(min.ApplyRule(reflect.ValueOf(minInt + 1))).To(BeNil())
		})
		It("checks exact value", func() {
			Expect(min.ApplyRule(reflect.ValueOf(minInt))).To(BeNil())
		})
		It("checks inferior value", func() {
			Expect(min.ApplyRule(reflect.ValueOf(minInt - 1))).ToNot(BeNil())
		})
	})

	// TODO: add unit tests to all validation markers
})
