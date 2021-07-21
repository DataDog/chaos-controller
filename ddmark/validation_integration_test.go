// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package ddmark_test

import (
	"github.com/DataDog/chaos-controller/ddmark"
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
		It("rejects empty pointer value", func() {
			var minmaxValidYaml string = `
minmaxtest:
  intfield: 6
  pintfield: 
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
		It("rejects on all but one missing fields", func() {
			var requiredOneFieldYaml string = `
requiredtest:
  intfield: 1
  pintfield: 
`
			errorList := validateString(requiredOneFieldYaml)
			Expect(errorList).To(HaveLen(5))
		})
		It("rejects and counts all 4 missing fields", func() {
			var requiredNoFieldYaml string = `
requiredtest:
  intfield: 1
  pintfield: 0
`
			errorList := validateString(requiredNoFieldYaml)
			Expect(errorList).To(HaveLen(4))
		})
		It("rejects and counts all 5 missing fields", func() {
			var requiredNoField2Yaml string = `
requiredtest:
  pintfield: 1
  pstructfield:
    a:
`
			errorList := validateString(requiredNoField2Yaml)
			Expect(errorList).To(HaveLen(4))
		})
		It("checks out on valid file", func() {
			var requiredValidYaml string = `
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
		It("allows latter fields to be set freely", func() {
			var exclusivefieldsValidYaml = `
exclusivefieldstest:
  bfield: 1
  cfield: 1
`
			errorList := validateString(exclusivefieldsValidYaml)
			Expect(errorList).To(HaveLen(0))
		})
	})

	Context("LinkedFields Marker", func() {
		It("checks out valid values", func() {
			var linkedfieldsValidYaml = `
linkedfieldstest:
  strfield: aa
  pstrfield: bb  
  intfield: 1
  pintfield: 1
  aintfield: [1,2]
`
			errorList := validateString(linkedfieldsValidYaml)
			Expect(errorList).To(HaveLen(0))
		})
		It("rejects both errors", func() {
			var linkedfieldsInvalidYaml = `
linkedfieldstest:
  strfield: aa
  pstrfield: aa
  intfield: 
  pintfield:
  aintfield:
`
			errorList := validateString(linkedfieldsInvalidYaml)
			Expect(errorList).To(HaveLen(2))
		})
		It("rejects one error, 0 value is nil on pointer", func() {
			var linkedfieldsInvalidYaml = `
linkedfieldstest:
  strfield: aa
  pstrfield: aa
  intfield: 0
  pintfield: 0
  aintfield: [1,2]
`
			errorList := validateString(linkedfieldsInvalidYaml)
			Expect(errorList).To(HaveLen(1))
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

func validateString(yamlStr string) []error {
	// Teststruct is a test-dedicated struct built strictly for these integration tests
	var marshalledStruct ddmark.Teststruct

	marshalledStruct, err := testStructFromYaml([]byte(yamlStr))
	errorList := ddmark.ValidateStruct(marshalledStruct, "test_suite",
		"github.com/DataDog/chaos-controller/ddmark",
	)
	if err != nil {
		errorList = append(errorList, err)
	}
	return errorList
}
