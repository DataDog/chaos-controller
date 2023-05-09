// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package ddmark_test

import (
	"github.com/DataDog/chaos-controller/ddmark"
	"github.com/hashicorp/go-multierror"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8syaml "sigs.k8s.io/yaml"
)

const (
	MinMaxTestErr0 = "test_suite>MinMaxTest>IntField - ddmark:validation:Minimum: field has value 4, min is 5 (included)"
	MinMaxTestErr1 = "test_suite>MinMaxTest>PIntField - ddmark:validation:Maximum: field has value 11, max is 10 (included)"

	RequiredTestErr0 = "test_suite>RequiredTest>PIntField is required"
	RequiredTestErr1 = "test_suite>RequiredTest>StrField - ddmark:validation:Required: field is required: currently missing"
	RequiredTestErr2 = "test_suite>RequiredTest>PStrField is required"
	RequiredTestErr3 = "test_suite>RequiredTest>StructField - ddmark:validation:Required: field is required: currently missing"
	RequiredTestErr4 = "test_suite>RequiredTest>PStructField is required"
	RequiredTestErr5 = "test_suite>RequiredTest>IntField - ddmark:validation:Required: field is required: currently missing"

	EnumTestErr0 = "test_suite>EnumTest>StrField - ddmark:validation:Enum: field needs to be one of [aa bb 11], currently \"notinenum\""
	EnumTestErr1 = "test_suite>EnumTest>PStrField - ddmark:validation:Enum: field needs to be one of [aa bb 11], currently \"notinenum\""
	EnumTestErr2 = "test_suite>EnumTest>IntField - ddmark:validation:Enum: field needs to be one of [1 2 3], currently \"4\""
	EnumTestErr3 = "test_suite>EnumTest>PIntField - ddmark:validation:Enum: field needs to be one of [1 2 3], currently \"4\""

	AtLeastOneOfTestErr0 = "test_suite>AtLeastOneOfTest - ddmark:validation:AtLeastOneOf: at least one of the following fields need to be non-nil (currently all nil): [StrField IntField]"
	AtLeastOneOfTestErr1 = "test_suite>AtLeastOneOfTest - ddmark:validation:AtLeastOneOf: at least one of the following fields need to be non-nil (currently all nil): [PStrField PIntField AIntField]"

	ExclusiveFieldsTestErr0 = "test_suite>ExclusiveFieldsTest - ddmark:validation:ExclusiveFields: some fields are incompatible, PIntField can't be set alongside any of [PStrField]"
	ExclusiveFieldsTestErr1 = "test_suite>ExclusiveFieldsTest - ddmark:validation:ExclusiveFields: some fields are incompatible, IntField can't be set alongside any of [StrField]"

	LinkedFieldsValueTestError0 = "test_suite>LinkedFieldsValueTest - ddmark:validation:LinkedFieldsValue: all of the following fields need to be either nil/at the indicated value or non-nil/not at the indicated value; currently unmatched: [StrField=aaa IntField]"
	LinkedFieldsValueTestError1 = "test_suite>LinkedFieldsValueTest - ddmark:validation:LinkedFieldsValue: all of the following fields need to be either nil/at the indicated value or non-nil/not at the indicated value; currently unmatched: [PStrField PIntField AIntField]"

	LinkedFieldsValueWithTriggerTestError0 = "test_suite>LinkedFieldsValueWithTriggerTest - ddmark:validation:LinkedFieldsValueWithTrigger: all of the following fields need to be aligned; if StrField=aaa is set, all the following need to either exist or have the indicated value: [IntField=2]"
	LinkedFieldsValueWithTriggerTestError1 = "test_suite>LinkedFieldsValueWithTriggerTest - ddmark:validation:LinkedFieldsValueWithTrigger: all of the following fields need to be aligned; if PStrField=bbb is set, all the following need to either exist or have the indicated value: [PIntField=12 AIntField]"
)

var _ = Describe("Validation Integration Tests", func() {
	Context("Minimum/Maximum Markers", func() {
		It("checks out valid values", func() {
			var minmaxYaml string = `
minmaxtest:
  intfield: 6
  pintfield: 6
`
			err := validateString(minmaxYaml)
			Expect(err.Errors).To(BeEmpty())
		})
		It("rejects empty pointer value", func() {
			var minmaxYaml string = `
minmaxtest:
  intfield: 6
  pintfield:
`
			err := validateString(minmaxYaml)
			Expect(err.Errors).To(BeEmpty())
		})
		It("checks out valid values", func() {
			var minmaxYaml string = `
minmaxtest:
  intfield: 5
  pintfield: 10
`
			err := validateString(minmaxYaml)
			Expect(err.Errors).To(BeEmpty())
		})
		It("rejects invalid values", func() {
			var minmaxYaml string = `
minmaxtest:
  intfield: 4
  pintfield: 11
`
			err := validateString(minmaxYaml)
			Expect(err.Errors).To(HaveLen(2))
			Expect(err.Errors[0]).To(MatchError(MinMaxTestErr0))
			Expect(err.Errors[1]).To(MatchError(MinMaxTestErr1))
		})
	})

	Context("Required Marker", func() {
		It("rejects on all but one missing fields", func() {
			var requiredYaml string = `
requiredtest:
  intfield: 1
`
			err := validateString(requiredYaml)
			Expect(err.Errors).To(HaveLen(5))
			Expect(err.Errors[0]).To(MatchError(RequiredTestErr0))
			Expect(err.Errors[1]).To(MatchError(RequiredTestErr1))
			Expect(err.Errors[2]).To(MatchError(RequiredTestErr2))
			Expect(err.Errors[3]).To(MatchError(RequiredTestErr3))
			Expect(err.Errors[4]).To(MatchError(RequiredTestErr4))
		})
		It("rejects on all but one missing fields", func() {
			var requiredYaml string = `
requiredtest:
  intfield: 1
  pintfield:
`
			err := validateString(requiredYaml)
			Expect(err.Errors).To(HaveLen(5))
			Expect(err.Errors[0]).To(MatchError(RequiredTestErr0))
			Expect(err.Errors[1]).To(MatchError(RequiredTestErr1))
			Expect(err.Errors[2]).To(MatchError(RequiredTestErr2))
			Expect(err.Errors[3]).To(MatchError(RequiredTestErr3))
			Expect(err.Errors[4]).To(MatchError(RequiredTestErr4))
		})
		It("rejects and counts all 4 missing fields", func() {
			var requiredYaml string = `
requiredtest:
  intfield: 1
  pintfield: 0
`
			err := validateString(requiredYaml)
			Expect(err.Errors).To(HaveLen(4))
			Expect(err.Errors[0]).To(MatchError(RequiredTestErr1))
			Expect(err.Errors[1]).To(MatchError(RequiredTestErr2))
			Expect(err.Errors[2]).To(MatchError(RequiredTestErr3))
			Expect(err.Errors[3]).To(MatchError(RequiredTestErr4))
		})
		It("rejects and counts all 5 missing fields", func() {
			var requiredYaml string = `
requiredtest:
  pintfield: 1
  pstructfield:
    a:
`
			err := validateString(requiredYaml)
			Expect(err.Errors).To(HaveLen(4))
			Expect(err.Errors[0]).To(MatchError(RequiredTestErr5))
			Expect(err.Errors[1]).To(MatchError(RequiredTestErr1))
			Expect(err.Errors[2]).To(MatchError(RequiredTestErr2))
			Expect(err.Errors[3]).To(MatchError(RequiredTestErr3))
		})
		It("checks out on valid file", func() {
			var requiredYaml string = `
requiredtest:
  intfield: 1
  pintfield: 0
  strfield: a
  pstrfield: ""
  structfield:
    a: 1
  pstructfield:
    a: 1
`
			err := validateString(requiredYaml)
			Expect(err.Errors).To(BeEmpty())
		})
	})

	Context("Enum Marker", func() {
		It("checks out valid values", func() {
			var enumCorrectYaml string = `
enumtest:
  strfield: aa
  pstrfield: bb
  intfield: 1
  pintfield: 2
`
			err := validateString(enumCorrectYaml)
			Expect(err.Errors).To(BeEmpty())
		})
		It("rejects invalid values", func() {
			var enumCorrectYaml string = `
enumtest:
  strfield: notinenum
  pstrfield: notinenum
  intfield: 4
  pintfield: 4
`
			err := validateString(enumCorrectYaml)
			Expect(err.Errors).To(HaveLen(4))
			Expect(err.Errors[0]).To(MatchError(EnumTestErr0))
			Expect(err.Errors[1]).To(MatchError(EnumTestErr1))
			Expect(err.Errors[2]).To(MatchError(EnumTestErr2))
			Expect(err.Errors[3]).To(MatchError(EnumTestErr3))
		})
	})

	Context("ExclusiveFields Marker", func() {
		It("rejects invalid values", func() {
			exclusivefieldsYaml := `
exclusivefieldstest:
  intfield: 1
  pintfield: 1
  strfield: aa
  pstrfield: aa
`
			err := validateString(exclusivefieldsYaml)
			Expect(err.Errors).To(HaveLen(2))
			Expect(err.Errors[0]).To(MatchError(ExclusiveFieldsTestErr0))
			Expect(err.Errors[1]).To(MatchError(ExclusiveFieldsTestErr1))
		})
		It("checks out valid values", func() {
			exclusivefieldsYaml := `
exclusivefieldstest:
  intfield:
  pintfield: 1
  strfield: aa
  pstrfield:
`
			err := validateString(exclusivefieldsYaml)
			Expect(err.Errors).To(BeEmpty())
		})
		It("allows latter fields to be set freely", func() {
			exclusivefieldsYaml := `
exclusivefieldstest:
  bfield: 1
  cfield: 1
`
			err := validateString(exclusivefieldsYaml)
			Expect(err.Errors).To(BeEmpty())
		})
	})

	Context("LinkedFieldsValue Marker", func() {
		It("checks out valid all-non-nil values", func() {
			linkedfieldsvalueYaml := `
linkedfieldsvaluetest:
  strfield: aaa
  pstrfield: bb
  intfield: 1
  pintfield: 1
  aintfield: [1,2]
`
			err := validateString(linkedfieldsvalueYaml)
			Expect(err.Errors).To(BeEmpty())
		})
		It("rejects invalid requires value for StrField", func() {
			linkedfieldsvalueYaml := `
linkedfieldsvaluetest:
  strfield: notaaa
  pstrfield: bb
  intfield: 1
  pintfield: 1
  aintfield: [1,2]
`
			err := validateString(linkedfieldsvalueYaml)
			Expect(err.Errors).To(HaveLen(1))
			Expect(err.Errors[0]).To(MatchError(LinkedFieldsValueTestError0))
		})
		It("checks out valid all-nil values", func() {
			linkedfieldsvalueYaml := `
linkedfieldsvaluetest:
  randomintfield: 1
  strfield:
  pstrfield:
  intfield: 0 # is zero / nil
  pintfield:  # is nil
  aintfield:
`
			err := validateString(linkedfieldsvalueYaml)
			Expect(err.Errors).To(BeEmpty())
		})
		It("rejects both errors - first fields", func() {
			linkedfieldsvalueYaml := `
linkedfieldsvaluetest:
  strfield: aa
  pstrfield: aa
  intfield: 1
  pintfield:
  aintfield:
`
			err := validateString(linkedfieldsvalueYaml)
			Expect(err.Errors).To(HaveLen(2))
			Expect(err.Errors[0]).To(MatchError(LinkedFieldsValueTestError0))
			Expect(err.Errors[1]).To(MatchError(LinkedFieldsValueTestError1))
		})
		It("rejects both errors - second fields", func() {
			linkedfieldsvalueYaml := `
linkedfieldsvaluetest:
  strfield:
  pstrfield:
  intfield: 1  # is non-nil
  pintfield: 0 # is non-nil
  aintfield:
`
			err := validateString(linkedfieldsvalueYaml)
			Expect(err.Errors).To(HaveLen(2))
			Expect(err.Errors[0]).To(MatchError(LinkedFieldsValueTestError0))
			Expect(err.Errors[1]).To(MatchError(LinkedFieldsValueTestError1))
		})
		It("rejects one error - 0 value is nil on pointer", func() {
			linkedfieldsvalueYaml := `
linkedfieldsvaluetest:
  strfield: aaa
  pstrfield: aa
  intfield: 0 	# is nil
  pintfield: 0  # is non-nil
  aintfield: [1,2]
`
			err := validateString(linkedfieldsvalueYaml)
			Expect(err.Errors).To(HaveLen(1))
			Expect(err.Errors[0]).To(MatchError(LinkedFieldsValueTestError0))
		})
	})

	Context("LinkedFieldsValueWithTrigger Marker", func() {
		It("is valid with incorrect triggers and incorrect other values", func() {
			linkedfieldsvaluewithtriggerYaml := `
linkedfieldsvaluewithtriggertest:
  strfield: aa      # incorrect trigger
  pstrfield: bb     # incorrect trigger
  intfield: 12
  pintfield: 1
  aintfield: [1,2]
`
			err := validateString(linkedfieldsvaluewithtriggerYaml)
			Expect(err.Errors).To(BeEmpty())
		})
		It("checks out valid all-nil initial values", func() {
			linkedfieldsvaluewithtriggerYaml := `
linkedfieldsvaluewithtriggertest:
  randomintfield: 1
  strfield:
  pstrfield:
  intfield: 0 # is nil
  pintfield:  # is nil
  aintfield:  # is nil
`
			err := validateString(linkedfieldsvaluewithtriggerYaml)
			Expect(err.Errors).To(BeEmpty())
		})
		It("rejects both errors - nil second fields", func() {
			linkedfieldsvaluewithtriggerYaml := `
linkedfieldsvaluewithtriggertest:
  strfield: aaa
  pstrfield: bbb
  intfield:  # is nil
  pintfield: # is nil
  aintfield: # is nil
`
			err := validateString(linkedfieldsvaluewithtriggerYaml)
			Expect(err.Errors).To(HaveLen(2))
			Expect(err.Errors[0]).To(MatchError(LinkedFieldsValueWithTriggerTestError0))
			Expect(err.Errors[1]).To(MatchError(LinkedFieldsValueWithTriggerTestError1))
		})
		It("rejects both errors - one second unfit, one second field missing", func() {
			linkedfieldsvaluewithtriggerYaml := `
linkedfieldsvaluewithtriggertest:
  strfield: aaa
  pstrfield: bbb
  intfield: 12 # is non-nil
  pintfield: 0 # is non-nil
  aintfield:   # is nil
`
			err := validateString(linkedfieldsvaluewithtriggerYaml)
			Expect(err.Errors).To(HaveLen(2))
			Expect(err.Errors[0]).To(MatchError(LinkedFieldsValueWithTriggerTestError0))
			Expect(err.Errors[1]).To(MatchError(LinkedFieldsValueWithTriggerTestError1))
		})
		It("rejects one error - 0 value is nil on pointer", func() {
			linkedfieldsvaluewithtriggerYaml := `
linkedfieldsvaluewithtriggertest:
  strfield: aaa
  pstrfield: bbb
  intfield: 0 	 # is nil
  pintfield: 0     # is non-nil
  aintfield: [1,2] # is non-nil
`
			err := validateString(linkedfieldsvaluewithtriggerYaml)
			Expect(err.Errors).To(HaveLen(2))
			Expect(err.Errors[0]).To(MatchError(LinkedFieldsValueWithTriggerTestError0))
			Expect(err.Errors[1]).To(MatchError(LinkedFieldsValueWithTriggerTestError1))
		})
	})

	Context("AtLeastOneOf Marker", func() {
		It("no error on all-nil sub-fields (marker not run)", func() {
			atLeastOneOfYaml := `
atleastoneoftest:
  strfield: "" # is nil
  pstrfield:   # is nil
  intfield: 0  # is nil
  pintfield:   # is nil
  aintfield:   # is nil
`
			err := validateString(atLeastOneOfYaml)
			Expect(err.Errors).To(BeEmpty())
		})
		It("rejects out all-nil values twice", func() {
			atLeastOneOfYaml := `
atleastoneoftest:
  randomintfield: 1
  strfield: "" # is nil
  pstrfield:   # is nil
  intfield: 0  # is nil
  pintfield:   # is nil
  aintfield:   # is nil
`
			err := validateString(atLeastOneOfYaml)
			Expect(err.Errors).To(HaveLen(2))
			Expect(err.Errors[0]).To(MatchError(AtLeastOneOfTestErr0))
			Expect(err.Errors[1]).To(MatchError(AtLeastOneOfTestErr1))
		})
		It("rejects almost-all-nil values once", func() {
			atLeastOneOfYaml := `
atleastoneoftest:
  strfield:
  pstrfield:
  intfield: 1 # is not-nil
  pintfield:  # is nil
  aintfield:
`
			err := validateString(atLeastOneOfYaml)
			Expect(err.Errors).To(HaveLen(1))
			Expect(err.Errors[0]).To(MatchError(AtLeastOneOfTestErr1))
		})
		It("accepts both valid value groups", func() {
			atLeastOneOfYaml := `
atleastoneoftest:
  strfield:
  pstrfield: a
  intfield: 1 # is not-nil
  pintfield:  # is nil
  aintfield:
`
			err := validateString(atLeastOneOfYaml)
			Expect(err.Errors).To(BeEmpty())
		})
		It("accepts both valid value groups", func() {
			atLeastOneOfYaml := `
atleastoneoftest:
  strfield: a
  pstrfield:
  intfield: # is nil
  pintfield:  # is nil
  aintfield: []
`
			err := validateString(atLeastOneOfYaml)
			Expect(err.Errors).To(BeEmpty())
		})
	})
})

// unmarshall a file into a TestStruct
func testStructFromYaml(yamlBytes []byte) (ddmark.Teststruct, error) {
	parsedSpec := ddmark.Teststruct{}
	err := k8syaml.UnmarshalStrict(yamlBytes, &parsedSpec)
	if err != nil {
		return ddmark.Teststruct{}, err
	}

	return parsedSpec, nil
}

func validateString(yamlStr string) *multierror.Error {
	// Teststruct is a test-dedicated struct built strictly for these integration tests
	var marshalledStruct ddmark.Teststruct
	var retErr *multierror.Error = &multierror.Error{}

	marshalledStruct, err := testStructFromYaml([]byte(yamlStr))
	if err != nil {
		return multierror.Append(retErr, err)
	}

	retErr = multierror.Append(client.ValidateStructMultierror(marshalledStruct, "test_suite"))

	return retErr
}
