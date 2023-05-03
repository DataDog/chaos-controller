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
			Expect(max.ApplyRule(ValueOf(-1001))).To(Succeed())
		})
		It("rejects small string values", func() {
			Expect(max.ApplyRule(ValueOf("0"))).ToNot(Succeed())
		})
		It("rejects large string values", func() {
			Expect(max.ApplyRule(ValueOf("1001"))).ToNot(Succeed())
		})
		It("rejects superior values", func() {
			Expect(max.ApplyRule(ValueOf(maxInt + 1))).ToNot(Succeed())
		})
		It("accepts exact value", func() {
			Expect(max.ApplyRule(ValueOf(maxInt))).To(Succeed())
		})
		It("accepts inferior value", func() {
			Expect(max.ApplyRule(ValueOf(maxInt - 1))).To(Succeed())
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
			Expect(min.ApplyRule(ValueOf(-1001))).ToNot(Succeed())
		})
		It("rejects small string values", func() {
			Expect(min.ApplyRule(ValueOf("0"))).ToNot(Succeed())
		})
		It("rejects large string values", func() {
			Expect(min.ApplyRule(ValueOf("1001"))).ToNot(Succeed())
		})
		It("accepts superior value", func() {
			Expect(min.ApplyRule(ValueOf(minInt + 1))).To(Succeed())
		})
		It("accepts exact value", func() {
			Expect(min.ApplyRule(ValueOf(minInt))).To(Succeed())
		})
		It("rejects inferior value", func() {
			Expect(min.ApplyRule(ValueOf(minInt - 1))).ToNot(Succeed())
		})
	})

	Context("Enum test", func() {
		arrStr := []interface{}{"a", "b", "c", "4"}
		arrInt := []interface{}{1, 2, 3}
		validStrEnum := Enum(arrStr)
		validIntEnum := Enum(arrInt)
		emptyEnum := Enum(nil)

		It("accepts a valid string value", func() {
			Expect(validStrEnum.ApplyRule(ValueOf(arrStr[0]))).To(Succeed())
		})
		It("rejects an invalid string value", func() {
			Expect(validStrEnum.ApplyRule(ValueOf("notavalue"))).ToNot(Succeed())
		})
		It("rejects an invalid int value", func() {
			Expect(validStrEnum.ApplyRule(ValueOf(4))).ToNot(Succeed())
		})
		It("rejects a combined str value", func() {
			Expect(validStrEnum.ApplyRule(ValueOf("ab"))).ToNot(Succeed())
		})
		It("accepts a valid int value", func() {
			Expect(validIntEnum.ApplyRule(ValueOf(arrInt[0]))).To(Succeed())
		})
		It("rejects an invalid int value", func() {
			Expect(validIntEnum.ApplyRule(ValueOf(4))).ToNot(Succeed())
		})
		It("int enum rejects a fitting string value", func() {
			Expect(validIntEnum.ApplyRule(ValueOf("1"))).ToNot(Succeed())
		})
		It("errors out if enum is empty", func() {
			Expect(emptyEnum.ApplyRule(ValueOf("any"))).ToNot(Succeed())
		})
	})

	Context("Required test", func() {
		const trueRequired Required = Required(true)
		const falseRequired Required = Required(false)

		It("true errors given nil", func() {
			Expect(trueRequired.ApplyRule(ValueOf(nil))).ToNot(Succeed())
		})
		It("true errors given empty string", func() {
			Expect(trueRequired.ApplyRule(ValueOf(""))).ToNot(Succeed())
		})
		It("true errors out given 0", func() {
			Expect(trueRequired.ApplyRule(ValueOf(0))).ToNot(Succeed())
		})
		It("true accepts regular values", func() {
			Expect(trueRequired.ApplyRule(ValueOf("a"))).To(Succeed())
			Expect(trueRequired.ApplyRule(ValueOf(1))).To(Succeed())
		})
		It("false doesn't error given nil", func() {
			Expect(falseRequired.ApplyRule(ValueOf(nil))).To(Succeed())
		})
		It("false accepts regular values", func() {
			Expect(trueRequired.ApplyRule(ValueOf("a"))).To(Succeed())
			Expect(trueRequired.ApplyRule(ValueOf(1))).To(Succeed())
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
			Expect(excl.ApplyRule(ValueOf(fakeObj))).ToNot(Succeed())
		})

		It("rejects object with 2 fields", func() {
			fakeObj.Field2 = 0
			Expect(excl.ApplyRule(ValueOf(fakeObj))).ToNot(Succeed())
		})

		It("validates object with 1 field", func() {
			fakeObj.Field1 = ""
			fakeObj.Field2 = 0
			Expect(excl.ApplyRule(ValueOf(fakeObj))).To(Succeed())
		})

		It("accepts object with 0 fields", func() {
			fakeObj.Field1 = ""
			fakeObj.Field2 = 0
			fakeObj.Field3 = 0
			Expect(excl.ApplyRule(ValueOf(fakeObj))).To(Succeed())
		})

		It("accepts object with all fields but first set", func() {
			fakeObj.Field1 = ""
			Expect(excl.ApplyRule(ValueOf(fakeObj))).To(Succeed())
		})
	})

	Context("LinkedFieldsValue test", func() {
		type dummyStruct struct {
			Field1 string
			Field2 int
			Field3 *int
		}

		// fakeObj is a unit test object that will be filled with non-empty values by default
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
				Expect(linked.ApplyRule(ValueOf(fakeObj))).To(Succeed())
			})

			It("is valid if given all nil fields", func() {
				fakeObj.Field1 = ""
				fakeObj.Field2 = 0
				fakeObj.Field3 = nil
				Expect(linked.ApplyRule(ValueOf(fakeObj))).To(Succeed())
			})

			It("is valid if given all non-nil fields (0-value pointer int is not-nil)", func() {
				i := 0
				var pi *int = &i

				fakeObj.Field3 = pi
				Expect(linked.ApplyRule(ValueOf(fakeObj))).To(Succeed())
			})

			It("is invalid if given an empty string value (empty-value string is nil)", func() {
				fakeObj.Field1 = ""
				Expect(linked.ApplyRule(ValueOf(fakeObj))).To(MatchError(linked.GenValueCheckError()))
			})

			It("is invalid if given one missing field (zero-int is nil)", func() {
				fakeObj.Field2 = 0
				Expect(linked.ApplyRule(ValueOf(fakeObj))).To(MatchError(linked.GenValueCheckError()))
			})

			It("is invalid if given nil pointer (nil pointer is nil)", func() {
				fakeObj.Field3 = nil
				Expect(linked.ApplyRule(ValueOf(fakeObj))).To(MatchError(linked.GenValueCheckError()))
			})
		})

		Context("given values/mixed requirements", func() {
			arr := []string{"Field1=a", "Field2=2", "Field3"}
			linked := LinkedFieldsValue(arr)

			It("is valid with all valid/non-nil fields", func() {
				Expect(linked.ApplyRule(ValueOf(fakeObj))).To(Succeed())
			})

			It("is valid if given all nil/invalid fields", func() {
				fakeObj.Field1 = "b"
				fakeObj.Field2 = 1
				fakeObj.Field3 = nil
				Expect(linked.ApplyRule(ValueOf(fakeObj))).To(Succeed())
			})

			It("is valid if given all non-nil fields (0-value pointer int is not-nil)", func() {
				i := 0
				var pi *int = &i

				fakeObj.Field3 = pi
				Expect(linked.ApplyRule(ValueOf(fakeObj))).To(Succeed())
			})

			It("is invalid if given empty string value (empty-value string is nil)", func() {
				fakeObj.Field1 = ""
				Expect(linked.ApplyRule(ValueOf(fakeObj))).To(MatchError(linked.GenValueCheckError()))
			})

			It("is invalid if given one missing field (Field2 is 0/nil, expected value was 2)", func() {
				fakeObj.Field2 = 0
				Expect(linked.ApplyRule(ValueOf(fakeObj))).To(MatchError(linked.GenValueCheckError()))
			})

			It("is invalid if given one incorrect field (Field2 is 3, expected value was 2)", func() {
				fakeObj.Field2 = 3
				Expect(linked.ApplyRule(ValueOf(fakeObj))).To(MatchError(linked.GenValueCheckError()))
			})

			It("is invalid if given nil pointer (empty value for Field3)", func() {
				fakeObj.Field3 = nil
				Expect(linked.ApplyRule(ValueOf(fakeObj))).To(MatchError(linked.GenValueCheckError()))
			})
		})

		Context("given empty value requirements", func() {
			arr := []string{"Field1=", "Field2="}
			linked := LinkedFieldsValue(arr)

			It("accepts non-empty values for both Field1 and Field2", func() {
				Expect(linked.ApplyRule(ValueOf(fakeObj))).To(Succeed())
			})

			Context("with empty string as a valid empty-required value for Field1", func() {
				BeforeEach(func() {
					fakeObj.Field1 = ""
				})
				It("is valid if Field2 is 0", func() {
					fakeObj.Field2 = 0
					Expect(linked.ApplyRule(ValueOf(fakeObj))).To(Succeed())
				})
				It("is invalid if Field2 is not 0", func() {
					fakeObj.Field2 = 1
					Expect(linked.ApplyRule(ValueOf(fakeObj))).To(MatchError(linked.GenValueCheckError()))
				})
			})

			Context("with non-empty string as an invalid empty-required value for Field1", func() {
				BeforeEach(func() {
					fakeObj.Field1 = "notempty"
				})
				It("is valid if Field2 is not 0", func() {
					fakeObj.Field2 = 1
					Expect(linked.ApplyRule(ValueOf(fakeObj))).To(Succeed())
				})
				It("is invalid if Field2 is 0", func() {
					fakeObj.Field2 = 0
					Expect(linked.ApplyRule(ValueOf(fakeObj))).To(MatchError(linked.GenValueCheckError()))
				})
			})
		})

		Context("given empty value requirements on pointer field", func() {
			// we expect that; if Field2 value is 0 then Field3 value is 0
			arr := []string{"Field2=", "Field3="}
			linked := LinkedFieldsValue(arr)

			It("accepts non-empty values for both Field2 and Field3", func() {
				Expect(linked.ApplyRule(ValueOf(fakeObj))).To(Succeed())
			})

			Context("with empty pointer as a valid empty-required value for Field3", func() {
				BeforeEach(func() {
					fakeObj.Field3 = nil
				})

				It("is valid if Field2 is 0", func() {
					fakeObj.Field2 = 0
					Expect(linked.ApplyRule(ValueOf(fakeObj))).To(Succeed())
				})

				It("is invalid if Field2 is not 0", func() {
					fakeObj.Field2 = 1
					Expect(linked.ApplyRule(ValueOf(fakeObj))).To(MatchError(linked.GenValueCheckError()))
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
					Expect(linked.ApplyRule(ValueOf(fakeObj))).To(Succeed())
				})

				It("is invalid if Field2 is 0", func() {
					fakeObj.Field2 = 0
					Expect(linked.ApplyRule(ValueOf(fakeObj))).To(MatchError(linked.GenValueCheckError()))
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

		// fakeObj is a unit test object that will be filled with valid/non-empty values by default
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

		Context("with non-pointer trigger value", func() {
			// we expect that; if Field1 value is "aaa" then Field 2 value is 12 and Field3 is not nil
			arr := []string{"Field1=aaa", "Field2=12", "Field3"}
			linked := LinkedFieldsValueWithTrigger(arr)

			Context("with valid trigger value ('Field1=aaa')", func() {
				It("is valid if all other fields are correct", func() {
					Expect(linked.ApplyRule(ValueOf(fakeObj))).To(Succeed())
				})

				It("is invalid if one value is nil / missing (nil-value pointer int is nil)", func() {
					fakeObj.Field3 = nil
					Expect(linked.ApplyRule(ValueOf(fakeObj))).To(MatchError(linked.GenValueCheckError()))
				})

				It("is invalid if one value is nil / missing (expected value for Field2 is value 12)", func() {
					fakeObj.Field2 = 0
					Expect(linked.ApplyRule(ValueOf(fakeObj))).To(MatchError(linked.GenValueCheckError()))
				})

				It("is invalid if one value is incorrect (expected value for Field2 is value 12)", func() {
					fakeObj.Field2 = 1
					Expect(linked.ApplyRule(ValueOf(fakeObj))).To(MatchError(linked.GenValueCheckError()))
				})

				It("is valid if other fields are correct (incl. 0-value pointer-int -- it's not-nil)", func() {
					i := 0
					pi := &i

					fakeObj.Field3 = pi
					Expect(linked.ApplyRule(ValueOf(fakeObj))).To(Succeed())
				})
			})

			Context("with invalid trigger value ('Field1 = bbb')", func() {
				BeforeEach(func() {
					fakeObj.Field1 = "bbb"
				})

				It("is valid if all other fields (except the trigger) are correct", func() {
					Expect(linked.ApplyRule(ValueOf(fakeObj))).To(Succeed())
				})

				It("is valid if all other fields are incorrect", func() {
					fakeObj.Field2 = 11
					fakeObj.Field3 = nil
					Expect(linked.ApplyRule(ValueOf(fakeObj))).To(Succeed())
				})

				It("is valid if all other fields are nil", func() {
					fakeObj.Field2 = 0
					fakeObj.Field3 = nil
					Expect(linked.ApplyRule(ValueOf(fakeObj))).To(Succeed())
				})
			})

			Context("with nil trigger value ('Field1 = \"\"')", func() {
				BeforeEach(func() {
					fakeObj.Field1 = ""
				})

				It("is valid if all other fields (except the trigger) are correct", func() {
					Expect(linked.ApplyRule(ValueOf(fakeObj))).To(Succeed())
				})

				It("is valid if all other fields are incorrect", func() {
					fakeObj.Field2 = 11
					fakeObj.Field3 = nil
					Expect(linked.ApplyRule(ValueOf(fakeObj))).To(Succeed())
				})

				It("is valid if all other fields are nil", func() {
					fakeObj.Field2 = 0
					fakeObj.Field3 = nil
					Expect(linked.ApplyRule(ValueOf(fakeObj))).To(Succeed())
				})
			})
		})

		Context("with pointer trigger value (Field3 is the trigger)", func() {
			// we expect that; if Field3 pointer value is 3 then Field1 is expected to be 'aaa' and Field 2 value is expected to be 12
			arr := []string{"Field3=3", "Field1=aaa", "Field2=12"}
			linked := LinkedFieldsValueWithTrigger(arr)

			Context("with valid trigger value ('Field3=3')", func() {
				It("is valid if all other fields are correct", func() {
					Expect(linked.ApplyRule(ValueOf(fakeObj))).To(Succeed())
				})

				It("is invalid if one value is nil / missing(expected value is value 12)", func() {
					fakeObj.Field2 = 0
					Expect(linked.ApplyRule(ValueOf(fakeObj))).To(MatchError(linked.GenValueCheckError()))
				})

				It("is invalid if one value is incorrect (expected value is value 12)", func() {
					fakeObj.Field2 = 1
					Expect(linked.ApplyRule(ValueOf(fakeObj))).To(MatchError(linked.GenValueCheckError()))
				})
			})

			Context("with nil trigger value ('Field3=nil') -- should work as invalid", func() {
				BeforeEach(func() {
					fakeObj.Field3 = nil
				})

				It("is valid if all other fields (except the trigger) are correct", func() {
					fakeObj.Field2 = 0
					fakeObj.Field3 = nil
					Expect(linked.ApplyRule(ValueOf(fakeObj))).To(Succeed())
				})

				It("is valid if all other fields are incorrect", func() {
					fakeObj.Field2 = 11
					fakeObj.Field3 = nil
					Expect(linked.ApplyRule(ValueOf(fakeObj))).To(Succeed())
				})

				It("is valid if all other fields are nil", func() {
					fakeObj.Field2 = 0
					fakeObj.Field3 = nil
					Expect(linked.ApplyRule(ValueOf(fakeObj))).To(Succeed())
				})
			})
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
			Expect(linked.ApplyRule(ValueOf(fakeObj))).To(Succeed())
		})

		It("validates object with all non-nil fields (0-value pointer int is not-nil)", func() {
			i := 0
			var pi *int = &i

			fakeObj.Field3 = pi
			Expect(linked.ApplyRule(ValueOf(fakeObj))).To(Succeed())
		})

		It("rejects object with all nil fields", func() {
			fakeObj.Field1 = ""
			fakeObj.Field2 = 0
			fakeObj.Field3 = nil
			Expect(linked.ApplyRule(ValueOf(fakeObj))).ToNot(Succeed())
		})

		It("validates object with only 1 value (0-value int is nil, nil-value pointer int is nil)", func() {
			fakeObj.Field2 = 0
			fakeObj.Field3 = nil
			Expect(linked.ApplyRule(ValueOf(fakeObj))).To(Succeed())
		})

		It("validates object with only 1 value (empty-value string is nil, 0-value int is nil)", func() {
			fakeObj.Field1 = ""
			fakeObj.Field2 = 0
			Expect(linked.ApplyRule(ValueOf(fakeObj))).To(Succeed())
		})

		It("validates object with only 1 value (empty-value string is nil, nil-value pointer is nil)", func() {
			fakeObj.Field1 = ""
			fakeObj.Field3 = nil
			Expect(linked.ApplyRule(ValueOf(fakeObj))).To(Succeed())
		})
	})
})
