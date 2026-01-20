// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

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
		yamlDisruptionSpec.WriteString("\n  service: demo-curl")
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
				yamlDisruptionSpec.WriteString("\n  path: /mnt/path")
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

var _ = Describe("Validator", func() {
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
		var spec *v1beta1.DisruptionSpec

		BeforeEach(func() {
			spec = &v1beta1.DisruptionSpec{
				Count:    &intstr.IntOrString{Type: intstr.String, StrVal: "100%"},
				Selector: map[string]string{"foo": "bar"},
				GRPC: &v1beta1.GRPCDisruptionSpec{
					Port: 8443,
					Endpoints: []v1beta1.EndpointAlteration{
						{
							TargetEndpoint:   "/getTest",
							ErrorToReturn:    "NOT_FOUND",
							OverrideToReturn: "",
							QueryPercent:     0,
						},
					},
				},
			}
			validator = spec
		})

		Context("with a valid spec", func() {
			It("should validate", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("with an invalid port", func() {
			BeforeEach(func() {
				spec.GRPC.Port = 70000
			})

			It("should not validate", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("GRPC.Port is set to 70000, but must be less or equal to 65535"))
			})
		})

		Context("without endpoints", func() {
			BeforeEach(func() {
				spec.GRPC.Endpoints = nil
			})

			It("should not validate", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("GRPC.Endpoints is a required field, and must be set"))
			})
		})

		Context("with an alteration without a target endpoint", func() {
			BeforeEach(func() {
				spec.GRPC.Endpoints[0].TargetEndpoint = ""
			})

			It("should not validate", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("TargetEndpoint is a required field, and must be set"))
			})
		})

		Context("with an alteration with an invalid error", func() {
			BeforeEach(func() {
				spec.GRPC.Endpoints[0].ErrorToReturn = "TOO_MANY_GRPCS"
				spec.GRPC.Endpoints[0].OverrideToReturn = ""
			})

			It("should not validate", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("ErrorToReturn is set to TOO_MANY_GRPCS, but must be one of the following: \"OK, CANCELED, UNKNOWN, INVALID_ARGUMENT, DEADLINE_EXCEEDED, NOT_FOUND, ALREADY_EXISTS, PERMISSION_DENIED, RESOURCE_EXHAUSTED, FAILED_PRECONDITION, ABORTED, OUT_OF_RANGE, UNIMPLEMENTED, INTERNAL, UNAVAILABLE, DATA_LOSS, UNAUTHENTICATED\""))
			})
		})

		Context("with an alteration without an override nor error", func() {
			BeforeEach(func() {
				spec.GRPC.Endpoints[0].ErrorToReturn = ""
				spec.GRPC.Endpoints[0].OverrideToReturn = ""
			})

			It("should not validate", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("gRPC disruption must have exactly one of ErrorToReturn or OverrideToReturn specified"))
			})
		})

		Context("with an alteration with an invalid query percentage", func() {
			BeforeEach(func() {
				spec.GRPC.Endpoints[0].QueryPercent = 267
			})

			It("should not validate", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("QueryPercent is set to 267, but must be less or equal to 100"))
			})
		})
	})

	Describe("validating network failure spec", func() {
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

		// We can skip the "valid spec" case as we use a network disruption for the top-level spec tests

		Context("without a failure type", func() {})

		Context("with invalid failure numbers", func() {
			BeforeEach(func() {
				spec.Network.Drop = -1
				spec.Network.Duplicate = 102
				spec.Network.Corrupt = 103
				spec.Network.Delay = 75000
				spec.Network.DelayJitter = 200
				spec.Network.BandwidthLimit = -1
			})

			It("should not validate", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("Network.Drop is set to -1, but must be greater or equal to 0"))
				Expect(err.Error()).Should(ContainSubstring("Network.Duplicate is set to 102, but must be less or equal to 100"))
				Expect(err.Error()).Should(ContainSubstring("Network.Corrupt is set to 103, but must be less or equal to 100"))
				Expect(err.Error()).Should(ContainSubstring("Network.Delay is set to 75000, but must be less or equal to 60000"))
				Expect(err.Error()).Should(ContainSubstring("Network.DelayJitter is set to 200, but must be less or equal to 100"))
				Expect(err.Error()).Should(ContainSubstring("Network.BandwidthLimit is set to -1, but must be greater or equal to 0"))
			})
		})

		Context("with an invalid host spec", func() {
			BeforeEach(func() {
				spec.Network.Hosts = []v1beta1.NetworkDisruptionHostSpec{
					{
						Host:      "optional!",
						Port:      -1,
						Protocol:  "grpc",
						Flow:      "away",
						ConnState: "all",
					},
				}
			})

			It("should not validate", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("Port is set to -1, but must be greater or equal to 0"))
				Expect(err.Error()).Should(ContainSubstring("Protocol is set to grpc, but must be one of the following: \"udp, tcp\""))
				Expect(err.Error()).Should(ContainSubstring("Flow is set to away, but must be one of the following: \"ingress, egress\""))
				Expect(err.Error()).Should(ContainSubstring("ConnState is set to all, but must be one of the following: \"new, est\""))
			})
		})

		Context("with an invalid service spec", func() {
			BeforeEach(func() {
				spec.Network.Services = []v1beta1.NetworkDisruptionServiceSpec{
					{
						Name:      "",
						Namespace: "",
						Ports: []v1beta1.NetworkDisruptionServicePortSpec{
							{
								Port: 8000000,
							},
						},
					},
				}
			})

			It("should not validate", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("Port is set to 8000000, but must be less or equal to 65535"))
			})
		})

		Context("with an invalid cloud spec", func() {
			BeforeEach(func() {
				spec.Network.Cloud = &v1beta1.NetworkDisruptionCloudSpec{
					DatadogServiceList: &[]v1beta1.NetworkDisruptionCloudServiceSpec{
						{
							ServiceName: "",
							Protocol:    "http",
							Flow:        "both",
							ConnState:   "old",
						},
					},
				}

			})

			It("should not validate", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("ServiceName is a required field, and must be set"))
				Expect(err.Error()).Should(ContainSubstring("Protocol is set to http, but must be one of the following: \"tcp, udp\""))
				Expect(err.Error()).Should(ContainSubstring("Flow is set to both, but must be one of the following: \"ingress, egress\""))
				Expect(err.Error()).Should(ContainSubstring("ConnState is set to old, but must be one of the following: \"new, est\""))
			})
		})

		Context("with an empty cloud spec", func() {
			BeforeEach(func() {
				spec.Network.Cloud = &v1beta1.NetworkDisruptionCloudSpec{}
			})

			It("should not validate", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("if network.cloud is specified, at least one of cloud.aws, cloud.gcp, or cloud.datadog must be set"))
			})
		})
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
				spec.DiskPressure.Path = "/mnt/example"
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
