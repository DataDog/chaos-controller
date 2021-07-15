// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package ddmark_test

import (
	"math/rand"
	. "reflect"

	. "github.com/DataDog/chaos-controller/ddmark/validation"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Validation Rules Cases", func() {

	Context("Maximum test", func() {
		maxInt := rand.Intn(1000)
		max := Maximum(maxInt)

		It("rejects superior values", func() {
			Expect(max.ApplyRule(ValueOf(maxInt + 1))).ToNot(BeNil())
		})
		It("accepts exact value", func() {
			Expect(max.ApplyRule(ValueOf(maxInt))).To(BeNil())
		})
		It("accepts inferior value", func() {
			Expect(max.ApplyRule(ValueOf(maxInt - 1))).To(BeNil())
		})
	})

	Context("Minimum test", func() {
		minInt := rand.Intn(1000)
		min := Minimum(minInt)
		It("accepts superior value", func() {
			Expect(min.ApplyRule(ValueOf(minInt + 1))).To(BeNil())
		})
		It("accepts exact value", func() {
			Expect(min.ApplyRule(ValueOf(minInt))).To(BeNil())
		})
		It("rejects inferior value", func() {
			Expect(min.ApplyRule(ValueOf(minInt - 1))).ToNot(BeNil())
		})
	})

	Context("Enum test", func() {
		arr := []string{"a", "b", "c"}
		validEnum := Enum(arr)
		emptyEnum := Enum(nil)

		It("accepts a valid value", func() {
			Expect(validEnum.ApplyRule(ValueOf(arr[0]))).To(BeNil())
		})
		It("rejects a valid value", func() {
			Expect(validEnum.ApplyRule(ValueOf("notavalue"))).ToNot(BeNil())
		})
		It("errors out if enum is empty", func() {
			Expect(emptyEnum.ApplyRule(ValueOf("any"))).ToNot(BeNil())
		})
	})

	Context("Required test", func() {
		trueRequired := Required(true)
		falseRequired := Required(false)

		It("true errors given nil", func() {
			Expect(trueRequired.ApplyRule(ValueOf(nil))).ToNot(BeNil())
		})
		It("true errors given empty string", func() {
			Expect(trueRequired.ApplyRule(ValueOf(""))).ToNot(BeNil())
		})
		It("true errors out given 0", func() {
			Expect(trueRequired.ApplyRule(ValueOf(0))).ToNot(BeNil())
		})
		It("true accepts regular values", func() {
			Expect(trueRequired.ApplyRule(ValueOf("a"))).To(BeNil())
			Expect(trueRequired.ApplyRule(ValueOf(1))).To(BeNil())
		})
		It("false doesn't error given nil", func() {
			Expect(falseRequired.ApplyRule(ValueOf(nil))).To(BeNil())
		})
	})

	Context("ExclusiveFields test", func() {
		arr := []string{"Field1", "Field2", "Field3"}

		excl := ExclusiveFields(arr)

		fakeObj := struct {
			Field1 string
			Field2 int
			Field3 int
		}{
			Field1: "a",
			Field2: 2,
			Field3: 3,
		}

		It("rejects object with 3+ fields", func() {
			Expect(excl.ApplyRule(ValueOf(fakeObj))).ToNot(BeNil())
		})

		It("rejects object with 2 fields", func() {
			fakeObj.Field2 = 0
			Expect(excl.ApplyRule(ValueOf(fakeObj))).ToNot(BeNil())
		})

		It("validates object with 1 field", func() {
			fakeObj.Field1 = ""
			fakeObj.Field2 = 0
			Expect(excl.ApplyRule(ValueOf(fakeObj))).To(BeNil())
		})
	})
})
