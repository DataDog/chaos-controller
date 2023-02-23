// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package api_test

import (
	"strings"

	chaostypes "github.com/DataDog/chaos-controller/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8syaml "sigs.k8s.io/yaml"

	"github.com/DataDog/chaos-controller/api/v1beta1"
)

var _ = Describe("Validator", func() {
	var (
		yamlDisruptionSpec strings.Builder
		errList            []error
	)

	BeforeEach(func() {
		yamlDisruptionSpec.Reset()
		yamlDisruptionSpec.WriteString("\nselector:")
		yamlDisruptionSpec.WriteString("\n  app: demo-curl")
		yamlDisruptionSpec.WriteString("\ncount: 1")
	})

	JustBeforeEach(func() {
		errList = ValidateDisruptionSpecFromString(yamlDisruptionSpec.String())
	})

	Describe("validating network spec", func() {
		BeforeEach(func() {
			yamlDisruptionSpec.WriteString("\nnetwork:")
		})

		Context("with an empty disruption", func() {
			It("should not validate", func() {
				Expect(errList).To(HaveLen(1))
			})
		})

		Context("with a non-empty disruption", func() {
			BeforeEach(func() {
				yamlDisruptionSpec.WriteString("\n  corrupt: 100")
			})

			It("should validate", func() {
				Expect(errList).To(HaveLen(0))
			})
		})
	})

	Describe("validating disk pressure spec", func() {
		BeforeEach(func() {
			yamlDisruptionSpec.WriteString("\ndiskPressure:")
		})

		Context("with an empty disruption", func() {
			It("should not validate", func() {
				Expect(errList).To(HaveLen(1))
			})
		})

		Context("with a non-empty disruption", func() {
			BeforeEach(func() {
				yamlDisruptionSpec.WriteString("\n  throttling:")
				yamlDisruptionSpec.WriteString("\n    writeBytesPerSec: 1024")
				yamlDisruptionSpec.WriteString("\n    readBytesPerSec: 1024")
			})

			It("should validate", func() {
				Expect(errList).To(HaveLen(0))
			})
		})
	})
})

var _ = Describe("Validator", func() {
	var (
		err       error
		validator *v1beta1.DisruptionSpec
	)

	JustBeforeEach(func() {
		err = validator.Validate()
	})
	Describe("validating container failure spec", func() {
		var spec *v1beta1.DisruptionSpec

		BeforeEach(func() {
			spec = &v1beta1.DisruptionSpec{
				Count:            &intstr.IntOrString{Type: intstr.String, StrVal: "100%"},
				ContainerFailure: &v1beta1.ContainerFailureSpec{},
				Selector:         map[string]string{"foo": "bar"},
			}
			validator = spec
		})

		Context("with level set to node", func() {
			BeforeEach(func() {
				spec.Level = chaostypes.DisruptionLevelNode
			})
			It("should not validate", func() {
				Expect(err).To(Not(BeNil()))
			})
		})

		Context("with level set to pod", func() {
			BeforeEach(func() {
				spec.Level = chaostypes.DisruptionLevelPod
			})
			It("should validate", func() {
				Expect(err).To(BeNil())
			})
		})
	})
})

// unmarshall a file into a DisruptionSpec
func disruptionSpecFromYaml(yamlBytes []byte) (v1beta1.DisruptionSpec, error) {
	parsedSpec := v1beta1.DisruptionSpec{}
	err := k8syaml.UnmarshalStrict(yamlBytes, &parsedSpec)

	if err != nil {
		return v1beta1.DisruptionSpec{}, err
	}

	return parsedSpec, nil
}

// run ddmark and validation through the Validate() interface
func ValidateDisruptionSpecFromString(yamlStr string) []error {
	var marshalledStruct v1beta1.DisruptionSpec

	marshalledStruct, err := disruptionSpecFromYaml([]byte(yamlStr))
	errorList := ddMarkClient.ValidateStruct(marshalledStruct, "test_suite")

	if err != nil {
		errorList = append(errorList, err)
	}

	err = marshalledStruct.Validate()
	if err != nil {
		errorList = append(errorList, err)
	}

	return errorList
}
