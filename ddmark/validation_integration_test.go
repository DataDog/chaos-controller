// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package ddmark_test

import (
	"github.com/DataDog/chaos-controller/ddmark"
	"github.com/hashicorp/go-multierror"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8syaml "sigs.k8s.io/yaml"
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
			Expect(err.Errors).To(HaveLen(0))
		})
		It("rejects empty pointer value", func() {
			var minmaxYaml string = `
minmaxtest:
  intfield: 6
  pintfield: 
`
			err := validateString(minmaxYaml)
			Expect(err.Errors).To(HaveLen(0))
		})
		It("checks out valid values", func() {
			var minmaxYaml string = `
minmaxtest:
  intfield: 5
  pintfield: 10
`
			err := validateString(minmaxYaml)
			Expect(err.Errors).To(HaveLen(0))
		})
		It("rejects invalid values", func() {
			var minmaxYaml string = `
minmaxtest:
  intfield: 4
  pintfield: 11
`
			err := validateString(minmaxYaml)
			Expect(err.Errors).To(HaveLen(2))
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
		})
		It("rejects on all but one missing fields", func() {
			var requiredYaml string = `
requiredtest:
  intfield: 1
  pintfield: 
`
			err := validateString(requiredYaml)
			Expect(err.Errors).To(HaveLen(5))
		})
		It("rejects and counts all 4 missing fields", func() {
			var requiredYaml string = `
requiredtest:
  intfield: 1
  pintfield: 0
`
			err := validateString(requiredYaml)
			Expect(err.Errors).To(HaveLen(4))
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
			Expect(err.Errors).To(HaveLen(0))
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
			Expect(err.Errors).To(HaveLen(0))
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
		})
	})

	Context("ExclusiveFields Marker", func() {
		It("rejects invalid values", func() {
			var exclusivefieldsYaml = `
exclusivefieldstest:
  intfield: 1
  pintfield: 1
  strfield: aa
  pstrfield: aa
`
			err := validateString(exclusivefieldsYaml)
			Expect(err.Errors).To(HaveLen(2))
		})
		It("checks out valid values", func() {
			var exclusivefieldsYaml = `
exclusivefieldstest:
  intfield: 
  pintfield: 1
  strfield: aa
  pstrfield: 
`
			err := validateString(exclusivefieldsYaml)
			Expect(err.Errors).To(HaveLen(0))
		})
		It("allows latter fields to be set freely", func() {
			var exclusivefieldsYaml = `
exclusivefieldstest:
  bfield: 1
  cfield: 1
`
			err := validateString(exclusivefieldsYaml)
			Expect(err.Errors).To(HaveLen(0))
		})
	})

	Context("LinkedFields Marker", func() {
		It("checks out valid all-non-nil values", func() {
			var linkedfieldsYaml = `
linkedfieldstest:
  strfield: aa
  pstrfield: bb  
  intfield: 1
  pintfield: 1
  aintfield: [1,2]
`
			err := validateString(linkedfieldsYaml)
			Expect(err.Errors).To(HaveLen(0))
		})
		It("checks out valid all-nil values", func() {
			var linkedfieldsYaml = `
linkedfieldstest:
  randomintfield: 1
  strfield:
  pstrfield:
  intfield: 0 # is nil
  pintfield:  # is nil
  aintfield:
`
			err := validateString(linkedfieldsYaml)
			Expect(err.Errors).To(HaveLen(0))
		})
		It("rejects both errors - first fields", func() {
			var linkedfieldsYaml = `
linkedfieldstest:
  strfield: aa
  pstrfield: aa
  intfield: 
  pintfield:
  aintfield:
`
			err := validateString(linkedfieldsYaml)
			Expect(err.Errors).To(HaveLen(2))
		})
		It("rejects both errors - second fields", func() {
			var linkedfieldsYaml = `
linkedfieldstest:
  strfield: 
  pstrfield: 
  intfield: 1  # is non-nil
  pintfield: 0 # is non-nil
  aintfield:
`
			err := validateString(linkedfieldsYaml)
			Expect(err.Errors).To(HaveLen(2))
		})
		It("rejects one error - 0 value is nil on pointer", func() {
			var linkedfieldsYaml = `
linkedfieldstest:
  strfield: aa
  pstrfield: aa
  intfield: 0 	# is nil
  pintfield: 0  # is non-nil
  aintfield: [1,2]
`
			err := validateString(linkedfieldsYaml)
			Expect(err.Errors).To(HaveLen(1))
		})
	})

	Context("AtLeastOneOf Marker", func() {
		It("no error on all-nil sub-fields (marker not run)", func() {
			var atLeastOneOfYaml = `
atleastoneoftest:
  strfield: "" # is nil
  pstrfield:   # is nil
  intfield: 0  # is nil
  pintfield:   # is nil
  aintfield:   # is nil
`
			err := validateString(atLeastOneOfYaml)
			Expect(err.Errors).To(HaveLen(0))
		})
		It("rejects out all-nil values twice", func() {
			var atLeastOneOfYaml = `
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
		})
		It("rejects almost-all-nil values once", func() {
			var atLeastOneOfYaml = `
atleastoneoftest:
  strfield:
  pstrfield:
  intfield: 1 # is not-nil
  pintfield:  # is nil
  aintfield:
`
			err := validateString(atLeastOneOfYaml)
			Expect(err.Errors).To(HaveLen(1))
		})
		It("accepts both valid value groups", func() {
			var atLeastOneOfYaml = `
atleastoneoftest:
  strfield:
  pstrfield: a
  intfield: 1 # is not-nil
  pintfield:  # is nil
  aintfield:
`
			err := validateString(atLeastOneOfYaml)
			Expect(err.Errors).To(HaveLen(0))
		})
		It("accepts both valid value groups", func() {
			var atLeastOneOfYaml = `
atleastoneoftest:
  strfield: a
  pstrfield: 
  intfield: # is nil
  pintfield:  # is nil
  aintfield: []
`
			err := validateString(atLeastOneOfYaml)
			Expect(err.Errors).To(HaveLen(0))
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

	marshalledStruct, err := testStructFromYaml([]byte(yamlStr))
	retErr := ddmark.ValidateStructMultierror(marshalledStruct, "test_suite", "ddmark-api")

	if err != nil {
		retErr = multierror.Append(retErr, err)
	}

	return retErr
}
