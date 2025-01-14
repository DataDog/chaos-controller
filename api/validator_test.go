// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package api_test

import (
	"strings"

	chaostypes "github.com/DataDog/chaos-controller/types"
	. "github.com/onsi/ginkgo/v2"
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

	Describe("validating disruption triggers", func() {
		Context("both offset and notBefore are set", func() {
			BeforeEach(func() {
				yamlDisruptionSpec.WriteString(`
network:
  corrupt: 100
duration: 87600h
triggers:
  createPods:
    notBefore: 2040-01-02T15:04:05-04:00
    offset: 1m
`)
			})

			It("should not validate", func() {
				Expect(errList).To(HaveLen(1))
			})
		})
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
				Expect(errList).To(BeEmpty())
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
				Expect(errList).To(BeEmpty())
			})
		})
	})
})

// TODO write a thousand unit tests for go-validator
// TODO remove the FDescribe
var _ = FDescribe("Validator", func() {
	var (
		err       error
		validator *v1beta1.DisruptionSpec
	)

	JustBeforeEach(func() {
		err = validator.Validate()
	})

	Describe("validating top-level spec", func() {
		var spec *v1beta1.DisruptionSpec

		BeforeEach(func() {
			spec = &v1beta1.DisruptionSpec{
				Count:    &intstr.IntOrString{Type: intstr.String, StrVal: "100%"},
				Selector: map[string]string{"foo": "bar"},
				Network: &v1beta1.NetworkDisruptionSpec{
					Drop: 100,
				},
			}
			validator = spec
		})

		Context("with no count", func() {
			BeforeEach(func() {
				spec.Count = nil
			})

			It("should not validate", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Count is a required field, and must be set"))
			})
		})

		Context("with an invalid level", func() {
			BeforeEach(func() {
				spec.Level = "host"
			})

			It("should not validate", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(" could not permit value \"host\" for field Level, try one of \"pod, node\""))
			})
		})

		Context("with an invalid disruption containing no kinds", func() {
			BeforeEach(func() {
				spec.Network = nil
			})

			It("should not validate", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(" at least one disruption kind must be specified"))
			})
		})

		Context("when using container failure with another kind", func() {
			BeforeEach(func() {
				spec.ContainerFailure = &v1beta1.ContainerFailureSpec{}
			})

			It("should not validate", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("container failure disruptions are not compatible with other disruption kinds. The container failure will remove the impact of the other disruption types"))
			})
		})

		Context("when using node failure with another kind", func() {
			BeforeEach(func() {
				spec.NodeFailure = &v1beta1.NodeFailureSpec{}
			})

			It("should not validate", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("node failure disruptions are not compatible with other disruption kinds. The node failure will remove the impact of the other disruption types"))
			})
		})

		Context("when specifying pulse's active duration without dormant", func() {
			BeforeEach(func() {
				spec.Pulse = &v1beta1.DisruptionPulse{
					ActiveDuration: "15s",
				}
			})

			It("should not validate", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("if spec.pulse.activeDuration or spec.pulse.dormantDuration are specified, then both options must be set"))
			})
		})

		Context("with a valid disruption", func() {
			It("should validate", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("validating grpc spec", func() {

	})

	Describe("validating network failure spec", func() {

	})

	Describe("validating disk failure spec", func() {
		var spec *v1beta1.DisruptionSpec

		BeforeEach(func() {
			spec = &v1beta1.DisruptionSpec{
				Count:    &intstr.IntOrString{Type: intstr.String, StrVal: "100%"},
				Selector: map[string]string{"foo": "bar"},
				DiskFailure: &v1beta1.DiskFailureSpec{
					Paths: []string{"/"},
				},
			}
			validator = spec
		})

		Context("with an empty disk failure spec", func() {
			BeforeEach(func() {
				spec.DiskFailure = &v1beta1.DiskFailureSpec{}
			})
			It("should not validate", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("DiskFailure.Paths is a required field, and must be set"))
			})
		})

		Context("with an invalid exit code", func() {
			BeforeEach(func() {
				spec.DiskFailure.OpenatSyscall = &v1beta1.OpenatSyscallSpec{ExitCode: "EBADEXIT"}
			})

			It("should not validate", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("DiskFailure.OpenatSyscall.ExitCode is set to EBADEXIT, but must be one of the following: \"EACCES, EDQUOT, EEXIST, EFAULT, EFBIG, EINTR, EISDIR, ELOOP, EMFILE, ENAMETOOLONG, ENFILE, ENODEV, ENOENT, ENOMEM, ENOSPC, ENOTDIR, ENXIO, EOVERFLOW, EPERM, EROFS, ETXTBSY, EWOULDBLOCK\""))
			})
		})
	})

	Describe("validating disk pressure spec", func() {
		var spec *v1beta1.DisruptionSpec

		BeforeEach(func() {
			spec = &v1beta1.DisruptionSpec{
				Count: &intstr.IntOrString{Type: intstr.String, StrVal: "100%"},
				DiskPressure: &v1beta1.DiskPressureSpec{
					Path:       "",
					Throttling: v1beta1.DiskPressureThrottlingSpec{},
				},
				Selector: map[string]string{"foo": "bar"},
			}
			validator = spec
		})

		Context("with throttling left empty", func() {
			It("should not validate", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring(".DiskPressure.Throttling is a required field, and must be set"))
			})
		})

		Context("with throttling", func() {
			BeforeEach(func() {
				readBytesPerSec := 1024
				spec.DiskPressure.Throttling.ReadBytesPerSec = &readBytesPerSec
			})

			It("should validate", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})
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
				Expect(err).To(HaveOccurred())
			})
		})

		Context("with level set to pod", func() {
			BeforeEach(func() {
				spec.Level = chaostypes.DisruptionLevelPod
			})
			It("should validate", func() {
				Expect(err).ToNot(HaveOccurred())
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

// run validation through the Validate() interface
func ValidateDisruptionSpecFromString(yamlStr string) (errorList []error) {
	var marshalledStruct v1beta1.DisruptionSpec

	marshalledStruct, err := disruptionSpecFromYaml([]byte(yamlStr))
	if err != nil {
		errorList = append(errorList, err)
	}

	err = marshalledStruct.Validate()
	if err != nil {
		errorList = append(errorList, err)
	}

	return errorList
}
