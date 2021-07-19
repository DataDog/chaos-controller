// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package validation_test

import (
	"github.com/DataDog/chaos-controller/ddmark"
	ddvalidation "github.com/DataDog/chaos-controller/ddmark/validation"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8syaml "sigs.k8s.io/yaml"
)

var _ = Describe("Validation Integration Tests", func() {
	Context("Minimum/Maximum Markers", func() {
		It("checks out valid values", func() {
			var minmaxValidYaml string = `
minmaxtest:
  intfield: 6
  pintfield: 6
`
			errorList := validateString(minmaxValidYaml)
			Expect(errorList).To(HaveLen(0))
		})
		It("checks out valid values", func() {
			var minmaxValidYaml string = `
minmaxtest:
  intfield: 5
  pintfield: 10
`
			errorList := validateString(minmaxValidYaml)
			Expect(errorList).To(HaveLen(0))
		})
		It("rejects invalid values", func() {
			var minmaxInvalidYaml string = `
minmaxtest:
  intfield: 4
  pintfield: 11
`
			errorList := validateString(minmaxInvalidYaml)
			Expect(errorList).To(HaveLen(2))
		})
	})

	Context("Required Marker", func() {
		It("rejects on all but one missing fields", func() {
			var requiredOneFieldYaml string = `
requiredtest:
  intfield: 1
`
			errorList := validateString(requiredOneFieldYaml)
			Expect(errorList).To(HaveLen(5))
		})
		It("rejects all missing fields", func() {
			var requiredNoFieldYaml string = `
requiredtest:
  intfield: 1
  pintfield: 0
`
			errorList := validateString(requiredNoFieldYaml)
			Expect(errorList).To(HaveLen(5))
		})
		It("rejects on all missing fields", func() {
			var requiredNoField2Yaml string = `
requiredtest:
  pintfield: 1
  pstructfield:
    a:
`
			errorList := validateString(requiredNoField2Yaml)
			Expect(errorList).To(HaveLen(5))
		})
		It("checks out on valid file", func() {
			var requiredValidYaml string = `
requiredtest:
  intfield: 1
  pintfield: 1
  strfield: a
  pstrfield: a
  structfield:
    a: 1
  pstructfield:
    a: 1
`
			errorList := validateString(requiredValidYaml)
			Expect(errorList).To(HaveLen(0))
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
			errorList := validateString(enumCorrectYaml)
			Expect(errorList).To(HaveLen(0))
		})
		It("rejects invalid values", func() {
			var enumCorrectYaml string = `
enumtest:
  strfield: notinenum
  pstrfield: notinenum
  intfield: 4
  pintfield: 4
`
			errorList := validateString(enumCorrectYaml)
			Expect(errorList).To(HaveLen(4))
		})
	})

	Context("ExclusiveFields Marker", func() {
		It("rejects invalid values", func() {
			var exclusivefieldsInvalidYaml = `
exclusivefieldstest:
  intfield: 1
  pintfield: 1
  strfield: aa
  pstrfield: aa
`
			errorList := validateString(exclusivefieldsInvalidYaml)
			Expect(errorList).To(HaveLen(2))
		})
		It("checks out valid values", func() {
			var exclusivefieldsValidYaml = `
exclusivefieldstest:
  intfield: 
  pintfield: 1
  strfield: aa
  pstrfield: 
`
			errorList := validateString(exclusivefieldsValidYaml)
			Expect(errorList).To(HaveLen(0))
		})
	})
})

// unmarshall a file into a TestStruct
func testStructFromYaml(yamlBytes []byte) (ddvalidation.Teststruct, error) {
	parsedSpec := ddvalidation.Teststruct{}
	err := k8syaml.UnmarshalStrict(yamlBytes, &parsedSpec)

	if err != nil {
		return ddvalidation.Teststruct{}, err
	}

	return parsedSpec, nil
}

func validateString(yamlStr string) []error {
	marshalledStruct, err := testStructFromYaml([]byte(yamlStr))
	errorList := ddmark.ValidateStruct(marshalledStruct, "test_suite",
		"github.com/DataDog/chaos-controller/ddmark/validation",
	)
	if err != nil {
		errorList = append(errorList, err)
	}
	return errorList
}
