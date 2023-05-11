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
			markerSuccess(max, -1001)
		})
		It("rejects small string values", func() {
			expectTypeError(max, "0")
		})
		It("rejects large string values", func() {
			expectTypeError(max, "1001")
		})
		It("rejects superior values", func() {
			expectValueError(max, maxInt+1)
		})
		It("accepts exact value", func() {
			markerSuccess(max, maxInt)
		})
		It("accepts inferior value", func() {
			markerSuccess(max, maxInt-1)
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
			expectValueError(min, -1001)
		})
		It("rejects small string values", func() {
			expectTypeError(min, "0")
		})
		It("rejects large string values", func() {
			expectTypeError(min, "1001")
		})
		It("accepts superior value", func() {
			markerSuccess(min, minInt+1)
		})
		It("accepts exact value", func() {
			markerSuccess(min, minInt)
		})
		It("rejects inferior value", func() {
			expectValueError(min, minInt-1)
		})
	})

	Context("Enum test", func() {
		arrStr := []interface{}{"a", "b", "c", "4"}
		arrInt := []interface{}{1, 2, 3}
		validStrEnum := Enum(arrStr)
		validIntEnum := Enum(arrInt)
		emptyEnum := Enum(nil)

		It("accepts a valid string value", func() {
			markerSuccess(validStrEnum, "a")
		})
		It("rejects an invalid string value", func() {
			expectValueError(validStrEnum, "notavalue")
		})
		It("rejects an invalid int value", func() {
			expectTypeError(validStrEnum, 4)
		})
		It("rejects a combined str value", func() {
			expectValueError(validStrEnum, "ab")
		})
		It("accepts a valid int value", func() {
			markerSuccess(validIntEnum, arrInt[0])
		})
		It("rejects an invalid int value", func() {
			expectValueError(validIntEnum, 4)
		})
		It("int enum rejects a fitting string value", func() {
			expectValueError(validIntEnum, "1")
		})
		It("errors out if enum is empty", func() {
			expectValueError(emptyEnum, "any")
		})
	})

	Context("Required test", func() {
		const trueRequired Required = Required(true)
		const falseRequired Required = Required(false)

		It("true errors given nil", func() {
			expectValueError(trueRequired, nil)
		})
		It("true errors given empty string", func() {
			expectValueError(trueRequired, "")
		})
		It("true errors out given 0", func() {
			expectValueError(trueRequired, 0)
		})
		It("true accepts regular values", func() {
			markerSuccess(trueRequired, "a")
			markerSuccess(trueRequired, 1)
		})
		It("false doesn't error given nil", func() {
			markerSuccess(falseRequired, nil)
		})
		It("false accepts regular values", func() {
			markerSuccess(falseRequired, "a")
			markerSuccess(falseRequired, 1)
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
			expectValueError(excl, fakeObj)
		})

		It("rejects object with 2 fields", func() {
			fakeObj.Field2 = 0
			expectValueError(excl, fakeObj)
		})

		It("validates object with 1 field", func() {
			fakeObj.Field1 = ""
			fakeObj.Field2 = 0
			markerSuccess(excl, fakeObj)
		})

		It("accepts object with 0 fields", func() {
			fakeObj.Field1 = ""
			fakeObj.Field2 = 0
			fakeObj.Field3 = 0
			markerSuccess(excl, fakeObj)
		})

		It("accepts object with all fields but first set", func() {
			fakeObj.Field1 = ""
			markerSuccess(excl, fakeObj)
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
				markerSuccess(linked, fakeObj)
			})

			It("is valid if given all nil fields", func() {
				fakeObj.Field1 = ""
				fakeObj.Field2 = 0
				fakeObj.Field3 = nil
				markerSuccess(linked, fakeObj)
			})

			It("is valid if given all non-nil fields (0-value pointer int is not-nil)", func() {
				i := 0
				var pi *int = &i

				fakeObj.Field3 = pi
				markerSuccess(linked, fakeObj)
			})

			It("is invalid if given an empty string value (empty-value string is nil)", func() {
				fakeObj.Field1 = ""
				expectValueError(linked, fakeObj)
			})

			It("is invalid if given one missing field (zero-int is nil)", func() {
				fakeObj.Field2 = 0
				expectValueError(linked, fakeObj)
			})

			It("is invalid if given nil pointer (nil pointer is nil)", func() {
				fakeObj.Field3 = nil
				expectValueError(linked, fakeObj)
			})
		})

		Context("given values/mixed requirements", func() {
			arr := []string{"Field1=a", "Field2=2", "Field3"}
			linked := LinkedFieldsValue(arr)

			It("is valid with all valid/non-nil fields", func() {
				markerSuccess(linked, fakeObj)
			})

			It("is valid if given all nil/invalid fields", func() {
				fakeObj.Field1 = "b"
				fakeObj.Field2 = 1
				fakeObj.Field3 = nil
				markerSuccess(linked, fakeObj)
			})

			It("is valid if given all non-nil fields (0-value pointer int is not-nil)", func() {
				i := 0
				var pi *int = &i

				fakeObj.Field3 = pi
				markerSuccess(linked, fakeObj)
			})

			It("is invalid if given empty string value (empty-value string is nil)", func() {
				fakeObj.Field1 = ""
				expectValueError(linked, fakeObj)
			})

			It("is invalid if given one missing field (Field2 is 0/nil, expected value was 2)", func() {
				fakeObj.Field2 = 0
				expectValueError(linked, fakeObj)
			})

			It("is invalid if given one incorrect field (Field2 is 3, expected value was 2)", func() {
				fakeObj.Field2 = 3
				expectValueError(linked, fakeObj)
			})

			It("is invalid if given nil pointer (empty value for Field3)", func() {
				fakeObj.Field3 = nil
				expectValueError(linked, fakeObj)
			})
		})

		Context("given empty value requirements", func() {
			arr := []string{"Field1=", "Field2="}
			linked := LinkedFieldsValue(arr)

			It("accepts non-empty values for both Field1 and Field2", func() {
				markerSuccess(linked, fakeObj)
			})

			Context("with empty string as a valid empty-required value for Field1", func() {
				BeforeEach(func() {
					fakeObj.Field1 = ""
				})
				It("is valid if Field2 is 0", func() {
					fakeObj.Field2 = 0
					markerSuccess(linked, fakeObj)
				})
				It("is invalid if Field2 is not 0", func() {
					fakeObj.Field2 = 1
					expectValueError(linked, fakeObj)
				})
			})

			Context("with non-empty string as an invalid empty-required value for Field1", func() {
				BeforeEach(func() {
					fakeObj.Field1 = "notempty"
				})
				It("is valid if Field2 is not 0", func() {
					fakeObj.Field2 = 1
					markerSuccess(linked, fakeObj)
				})
				It("is invalid if Field2 is 0", func() {
					fakeObj.Field2 = 0
					expectValueError(linked, fakeObj)
				})
			})
		})

		Context("given empty value requirements on pointer field", func() {
			// we expect that; if Field2 value is 0 then Field3 value is 0
			arr := []string{"Field2=", "Field3="}
			linked := LinkedFieldsValue(arr)

			It("accepts non-empty values for both Field2 and Field3", func() {
				markerSuccess(linked, fakeObj)
			})

			Context("with empty pointer as a valid empty-required value for Field3", func() {
				BeforeEach(func() {
					fakeObj.Field3 = nil
				})

				It("is valid if Field2 is 0", func() {
					fakeObj.Field2 = 0
					markerSuccess(linked, fakeObj)
				})

				It("is invalid if Field2 is not 0", func() {
					fakeObj.Field2 = 1
					expectValueError(linked, fakeObj)
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
					markerSuccess(linked, fakeObj)
				})

				It("is invalid if Field2 is 0", func() {
					fakeObj.Field2 = 0
					expectValueError(linked, fakeObj)
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
					markerSuccess(linked, fakeObj)
				})

				It("is invalid if one value is nil / missing (nil-value pointer int is nil)", func() {
					fakeObj.Field3 = nil
					expectValueError(linked, fakeObj)
				})

				It("is invalid if one value is nil / missing (expected value for Field2 is value 12)", func() {
					fakeObj.Field2 = 0
					expectValueError(linked, fakeObj)
				})

				It("is invalid if one value is incorrect (expected value for Field2 is value 12)", func() {
					fakeObj.Field2 = 1
					expectValueError(linked, fakeObj)
				})

				It("is valid if other fields are correct (incl. 0-value pointer-int -- it's not-nil)", func() {
					i := 0
					pi := &i

					fakeObj.Field3 = pi
					markerSuccess(linked, fakeObj)
				})
			})

			Context("with invalid trigger value ('Field1 = bbb')", func() {
				BeforeEach(func() {
					fakeObj.Field1 = "bbb"
				})

				It("is valid if all other fields (except the trigger) are correct", func() {
					markerSuccess(linked, fakeObj)
				})

				It("is valid if all other fields are incorrect", func() {
					fakeObj.Field2 = 11
					fakeObj.Field3 = nil
					markerSuccess(linked, fakeObj)
				})

				It("is valid if all other fields are nil", func() {
					fakeObj.Field2 = 0
					fakeObj.Field3 = nil
					markerSuccess(linked, fakeObj)
				})
			})

			Context("with nil trigger value ('Field1 = \"\"')", func() {
				BeforeEach(func() {
					fakeObj.Field1 = ""
				})

				It("is valid if all other fields (except the trigger) are correct", func() {
					markerSuccess(linked, fakeObj)
				})

				It("is valid if all other fields are incorrect", func() {
					fakeObj.Field2 = 11
					fakeObj.Field3 = nil
					markerSuccess(linked, fakeObj)
				})

				It("is valid if all other fields are nil", func() {
					fakeObj.Field2 = 0
					fakeObj.Field3 = nil
					markerSuccess(linked, fakeObj)
				})
			})
		})

		Context("with pointer trigger value (Field3 is the trigger)", func() {
			// we expect that; if Field3 pointer value is 3 then Field1 is expected to be 'aaa' and Field 2 value is expected to be 12
			arr := []string{"Field3=3", "Field1=aaa", "Field2=12"}
			linked := LinkedFieldsValueWithTrigger(arr)

			Context("with valid trigger value ('Field3=3')", func() {
				It("is valid if all other fields are correct", func() {
					markerSuccess(linked, fakeObj)
				})

				It("is invalid if one value is nil / missing(expected value is value 12)", func() {
					fakeObj.Field2 = 0
					expectValueError(linked, fakeObj)
				})

				It("is invalid if one value is incorrect (expected value is value 12)", func() {
					fakeObj.Field2 = 1
					expectValueError(linked, fakeObj)
				})
			})

			Context("with nil trigger value ('Field3=nil') -- should work as invalid", func() {
				BeforeEach(func() {
					fakeObj.Field3 = nil
				})

				It("is valid if all other fields (except the trigger) are correct", func() {
					fakeObj.Field2 = 0
					fakeObj.Field3 = nil
					markerSuccess(linked, fakeObj)
				})

				It("is valid if all other fields are incorrect", func() {
					fakeObj.Field2 = 11
					fakeObj.Field3 = nil
					markerSuccess(linked, fakeObj)
				})

				It("is valid if all other fields are nil", func() {
					fakeObj.Field2 = 0
					fakeObj.Field3 = nil
					markerSuccess(linked, fakeObj)
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
		atLeast := AtLeastOneOf(arr)
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
			markerSuccess(atLeast, fakeObj)
		})

		It("validates object with all non-nil fields (0-value pointer int is not-nil)", func() {
			i := 0
			var pi *int = &i

			fakeObj.Field3 = pi
			markerSuccess(atLeast, fakeObj)
		})

		It("rejects object with all nil fields", func() {
			fakeObj.Field1 = ""
			fakeObj.Field2 = 0
			fakeObj.Field3 = nil
			expectValueError(atLeast, fakeObj)
		})

		It("validates object with only 1 value (0-value int is nil, nil-value pointer int is nil)", func() {
			fakeObj.Field2 = 0
			fakeObj.Field3 = nil
			markerSuccess(atLeast, fakeObj)
		})

		It("validates object with only 1 value (empty-value string is nil, 0-value int is nil)", func() {
			fakeObj.Field1 = ""
			fakeObj.Field2 = 0
			markerSuccess(atLeast, fakeObj)
		})

		It("validates object with only 1 value (empty-value string is nil, nil-value pointer is nil)", func() {
			fakeObj.Field1 = ""
			fakeObj.Field3 = nil
			markerSuccess(atLeast, fakeObj)
		})
	})
})

// HELPERS -- reusable logic

// markerSuccess runs given value against the marker and expects no error
func markerSuccess(marker DDValidationMarker, i any) bool {
	return Expect(marker.ApplyRule(ValueOf(i))).To(Succeed())
}

// expectTypeError runs given value against the marker and expects a type check error
func expectTypeError(marker DDValidationMarker, i any) {
	Expect(marker.ApplyRule(ValueOf(i))).To(MatchError(marker.TypeCheckError(ValueOf(i))))
}

// expectTypeError runs given value against the marker and expects a value check error
func expectValueError(marker DDValidationMarker, i any) {
	Expect(marker.ApplyRule(ValueOf(i))).To(MatchError(marker.ValueCheckError()))
}
