// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package ddmark_test

import (
	"math/rand"
	. "reflect"

	. "github.com/DataDog/chaos-controller/ddmark"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Validation Rules Cases", func() {
	Context("Maximum test", func() {
		var maxInt int
		var max Maximum

		BeforeEach(func() {
			maxInt = rand.Intn(1000)
			max = Maximum(maxInt)
		})

		It("accept large negative values", func() {
			Expect(max.ApplyRule(ValueOf(-1001))).To(BeNil())
		})
		It("rejects small string values", func() {
			Expect(max.ApplyRule(ValueOf("0"))).ToNot(BeNil())
		})
		It("rejects large string values", func() {
			Expect(max.ApplyRule(ValueOf("1001"))).ToNot(BeNil())
		})
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
		var minInt int
		var min Minimum

		BeforeEach(func() {
			minInt = rand.Intn(1000)
			min = Minimum(minInt)
		})

		It("rejects large negative values", func() {
			Expect(min.ApplyRule(ValueOf(-1001))).ToNot(BeNil())
		})
		It("rejects small string values", func() {
			Expect(min.ApplyRule(ValueOf("0"))).ToNot(BeNil())
		})
		It("rejects large string values", func() {
			Expect(min.ApplyRule(ValueOf("1001"))).ToNot(BeNil())
		})
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
		arrStr := []interface{}{"a", "b", "c", "4"}
		arrInt := []interface{}{1, 2, 3}
		validStrEnum := Enum(arrStr)
		validIntEnum := Enum(arrInt)
		emptyEnum := Enum(nil)

		It("accepts a valid string value", func() {
			Expect(validStrEnum.ApplyRule(ValueOf(arrStr[0]))).To(BeNil())
		})
		It("rejects an invalid string value", func() {
			Expect(validStrEnum.ApplyRule(ValueOf("notavalue"))).ToNot(BeNil())
		})
		It("rejects an invalid int value", func() {
			Expect(validStrEnum.ApplyRule(ValueOf(4))).ToNot(BeNil())
		})
		It("rejects a combined str value", func() {
			Expect(validStrEnum.ApplyRule(ValueOf("ab"))).ToNot(BeNil())
		})
		It("accepts a valid int value", func() {
			Expect(validIntEnum.ApplyRule(ValueOf(arrInt[0]))).To(BeNil())
		})
		It("rejects an invalid int value", func() {
			Expect(validIntEnum.ApplyRule(ValueOf(4))).ToNot(BeNil())
		})
		It("int enum rejects a fitting string value", func() {
			Expect(validIntEnum.ApplyRule(ValueOf("1"))).ToNot(BeNil())
		})
		It("errors out if enum is empty", func() {
			Expect(emptyEnum.ApplyRule(ValueOf("any"))).ToNot(BeNil())
		})
	})

	Context("Required test", func() {
		const trueRequired Required = Required(true)
		const falseRequired Required = Required(false)

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
		It("false accepts regular values", func() {
			Expect(trueRequired.ApplyRule(ValueOf("a"))).To(BeNil())
			Expect(trueRequired.ApplyRule(ValueOf(1))).To(BeNil())
		})
	})

	Context("ExclusiveFields test", func() {
		type dummyStruct struct {
			Field1 string
			Field2 int
			Field3 int
		}

		arr := []string{"Field1", "Field2", "Field3"}
		excl := ExclusiveFields(arr)
		var fakeObj dummyStruct

		BeforeEach(func() {
			fakeObj = dummyStruct{
				Field1: "a",
				Field2: 2,
				Field3: 3,
			}
		})

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

		It("accepts object with 0 fields", func() {
			fakeObj.Field1 = ""
			fakeObj.Field2 = 0
			fakeObj.Field3 = 0
			Expect(excl.ApplyRule(ValueOf(fakeObj))).To(BeNil())
		})

		It("accepts object with all fields but first set", func() {
			fakeObj.Field1 = ""
			Expect(excl.ApplyRule(ValueOf(fakeObj))).To(BeNil())
		})
	})

	Context("LinkedFieldsValue test", func() {
		type dummyStruct struct {
			Field1 string
			Field2 int
			Field3 *int
		}

		// fakeObj is a unit test object that will be filled with non-empty values
		var fakeObj dummyStruct

		BeforeEach(func() {
			i := 3
			var pi *int = &i

			fakeObj = dummyStruct{
				Field1: "a",
				Field2: 2,
				Field3: pi,
			}
		})

		Context("given standard requirements", func() {
			arr := []string{"Field1", "Field2", "Field3"}
			linked := LinkedFieldsValue(arr)

			It("is valid with all non-nil fields", func() {
				Expect(linked.ApplyRule(ValueOf(fakeObj))).To(BeNil())
			})

			It("is valid if given all nil fields", func() {
				fakeObj.Field1 = ""
				fakeObj.Field2 = 0
				fakeObj.Field3 = nil
				Expect(linked.ApplyRule(ValueOf(fakeObj))).To(BeNil())
			})

			It("is valid if given all non-nil fields (0-value pointer int is not-nil)", func() {
				i := 0
				var pi *int = &i

				fakeObj.Field3 = pi
				Expect(linked.ApplyRule(ValueOf(fakeObj))).To(BeNil())
			})

			It("is invalid if given an empty string value (empty-value string is nil)", func() {
				fakeObj.Field1 = ""
				Expect(linked.ApplyRule(ValueOf(fakeObj))).ToNot(BeNil())
			})

			It("is invalid if given one missing field (zero-int is nil)", func() {
				fakeObj.Field2 = 0
				Expect(linked.ApplyRule(ValueOf(fakeObj))).ToNot(BeNil())
			})

			It("is invalid if given nil pointer (nil pointer is nil)", func() {
				fakeObj.Field3 = nil
				Expect(linked.ApplyRule(ValueOf(fakeObj))).ToNot(BeNil())
			})
		})

		Context("given values/mixed requirements", func() {
			arr := []string{"Field1=a", "Field2=2", "Field3"}
			linked := LinkedFieldsValue(arr)

			It("is valid with all valid/non-nil fields", func() {
				Expect(linked.ApplyRule(ValueOf(fakeObj))).To(BeNil())
			})

			It("is valid if given all nil/invalid fields", func() {
				fakeObj.Field1 = "b"
				fakeObj.Field2 = 1
				fakeObj.Field3 = nil
				Expect(linked.ApplyRule(ValueOf(fakeObj))).To(BeNil())
			})

			It("is valid if given all non-nil fields (0-value pointer int is not-nil)", func() {
				i := 0
				var pi *int = &i

				fakeObj.Field3 = pi
				Expect(linked.ApplyRule(ValueOf(fakeObj))).To(BeNil())
			})

			It("is invalid if given empty string value (empty-value string is nil)", func() {
				fakeObj.Field1 = ""
				Expect(linked.ApplyRule(ValueOf(fakeObj))).ToNot(BeNil())
			})

			It("is invalid if given one missing field (Field2 is 0/nil, expected value was 2)", func() {
				fakeObj.Field2 = 0
				Expect(linked.ApplyRule(ValueOf(fakeObj))).ToNot(BeNil())
			})

			It("is invalid if given one incorrect field (Field2 is 3, expected value was 2)", func() {
				fakeObj.Field2 = 3
				Expect(linked.ApplyRule(ValueOf(fakeObj))).ToNot(BeNil())
			})

			It("is invalid if given nil pointer (empty value for Field3)", func() {
				fakeObj.Field3 = nil
				Expect(linked.ApplyRule(ValueOf(fakeObj))).ToNot(BeNil())
			})
		})

		Context("given empty value requirements", func() {
			arr := []string{"Field1=", "Field2="}
			emptyLinked := LinkedFieldsValue(arr)

			It("accepts non-empty values for both Field1 and Field2", func() {
				Expect(emptyLinked.ApplyRule(ValueOf(fakeObj))).To(BeNil())
			})

			Context("with empty string as a valid empty-required value for Field1", func() {
				BeforeEach(func() {
					fakeObj.Field1 = ""
				})
				It("is valid if Field2 is 0", func() {
					fakeObj.Field2 = 0
					Expect(emptyLinked.ApplyRule(ValueOf(fakeObj))).To(BeNil())
				})
				It("is invalid if Field2 is not 0", func() {
					fakeObj.Field2 = 1
					Expect(emptyLinked.ApplyRule(ValueOf(fakeObj))).ToNot(BeNil())
				})
			})
			Context("with non-empty string as an invalid empty-required value for Field1", func() {
				BeforeEach(func() {
					fakeObj.Field1 = "notempty"
				})
				It("is valid if Field2 is not zero", func() {
					fakeObj.Field2 = 1
					Expect(emptyLinked.ApplyRule(ValueOf(fakeObj))).To(BeNil())
				})
				It("is invalid if Field2 is zero", func() {
					fakeObj.Field2 = 0
					Expect(emptyLinked.ApplyRule(ValueOf(fakeObj))).ToNot(BeNil())
				})

			})
		})

		Context("given empty value requirements on pointer field", func() {
			arr := []string{"Field2=", "Field3="}
			emptyPointerLinked := LinkedFieldsValue(arr)

			It("accepts non-empty values for both Field2 and Field3", func() {
				Expect(emptyPointerLinked.ApplyRule(ValueOf(fakeObj))).To(BeNil())
			})

			Context("with empty pointer as a valid empty-required value for Field3", func() {
				BeforeEach(func() {
					fakeObj.Field3 = nil
				})

				It("is valid if Field2 is 0", func() {
					fakeObj.Field2 = 0
					Expect(emptyPointerLinked.ApplyRule(ValueOf(fakeObj))).To(BeNil())
				})

				It("is invalid if Field2 is not 0", func() {
					fakeObj.Field2 = 1
					Expect(emptyPointerLinked.ApplyRule(ValueOf(fakeObj))).ToNot(BeNil())
				})
			})

			Context("with valid pointer to a zero int (or any other int) as a not-empty value for Field3", func() {
				BeforeEach(func() {
					i := 0
					pi := &i

					fakeObj.Field3 = pi
				})

				It("is valid if Field2 is not 0", func() {
					fakeObj.Field2 = 1
					Expect(emptyPointerLinked.ApplyRule(ValueOf(fakeObj))).To(BeNil())
				})

				It("is invalid if Field2 is 0", func() {
					fakeObj.Field2 = 0
					Expect(emptyPointerLinked.ApplyRule(ValueOf(fakeObj))).ToNot(BeNil())
				})
			})
		})
	})

	Context("LinkedFieldsValueWithTrigger test", func() {
		type dummyStruct struct {
			Field1 string
			Field2 int
			Field3 *int
		}

		// we expect that; if Field1 value is "aaa" then Field 2 value is 12 and Field3 is not nil
		arr := []string{"Field1=aaa", "Field2=12", "Field3"}
		linked := LinkedFieldsValueWithTrigger(arr)
		var fakeObj dummyStruct

		BeforeEach(func() {
			i := 3
			var pi *int = &i

			fakeObj = dummyStruct{
				Field1: "aaa",
				Field2: 12,
				Field3: pi,
			}
		})
		It("validates object with all correct fields", func() {
			Expect(linked.ApplyRule(ValueOf(fakeObj))).To(BeNil())
		})

		It("validates object with all nil fields", func() {
			fakeObj.Field1 = ""
			fakeObj.Field2 = 0
			fakeObj.Field3 = nil
			Expect(linked.ApplyRule(ValueOf(fakeObj))).To(BeNil())
		})

		It("validates object with correct trigger, correct fields (incl. 0-value pointer-int -- it's not-nil)", func() {
			i := 0
			var pi *int = &i

			fakeObj.Field3 = pi
			Expect(linked.ApplyRule(ValueOf(fakeObj))).To(BeNil())
		})

		It("rejects object with nil pointer value (nil-value pointer int is nil)", func() {
			fakeObj.Field3 = nil
			Expect(linked.ApplyRule(ValueOf(fakeObj))).ToNot(BeNil())
		})

		It("rejects object with one missing value (expected value is value 12)", func() {
			fakeObj.Field2 = 0
			Expect(linked.ApplyRule(ValueOf(fakeObj))).ToNot(BeNil())
		})

		It("rejects object with one incorrect value (expected value is value 12)", func() {
			fakeObj.Field2 = 1
			Expect(linked.ApplyRule(ValueOf(fakeObj))).ToNot(BeNil())
		})
	})

	Context("AtLeastOneOf test", func() {
		type dummyStruct struct {
			Field1 string
			Field2 int
			Field3 *int
		}

		arr := []string{"Field1", "Field2", "Field3"}
		linked := AtLeastOneOf(arr)
		var fakeObj dummyStruct

		BeforeEach(func() {
			i := 3
			var pi *int = &i

			fakeObj = dummyStruct{
				Field1: "a",
				Field2: 2,
				Field3: pi,
			}
		})
		It("validates object with all non-nil fields", func() {
			Expect(linked.ApplyRule(ValueOf(fakeObj))).To(BeNil())
		})

		It("validates object with all non-nil fields (0-value pointer int is not-nil)", func() {
			i := 0
			var pi *int = &i

			fakeObj.Field3 = pi
			Expect(linked.ApplyRule(ValueOf(fakeObj))).To(BeNil())
		})

		It("rejects object with all nil fields", func() {
			fakeObj.Field1 = ""
			fakeObj.Field2 = 0
			fakeObj.Field3 = nil
			Expect(linked.ApplyRule(ValueOf(fakeObj))).ToNot(BeNil())
		})

		It("validates object with only 1 value (0-value int is nil, nil-value pointer int is nil)", func() {
			fakeObj.Field2 = 0
			fakeObj.Field3 = nil
			Expect(linked.ApplyRule(ValueOf(fakeObj))).To(BeNil())
		})

		It("validates object with only 1 value (empty-value string is nil, 0-value int is nil)", func() {
			fakeObj.Field1 = ""
			fakeObj.Field2 = 0
			Expect(linked.ApplyRule(ValueOf(fakeObj))).To(BeNil())
		})

		It("validates object with only 1 value (empty-value string is nil, nil-value pointer is nil)", func() {
			fakeObj.Field1 = ""
			fakeObj.Field3 = nil
			Expect(linked.ApplyRule(ValueOf(fakeObj))).To(BeNil())
		})
	})
})
