// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package v1beta1

import (
	"math/rand"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const SpacesErrorMessagePrefix = "should not contains spaces"

var _ = Describe("NetworkDisruptionSpec", func() {
	When("'Format' method is called", func() {
		Context("with filters", func() {
			It("expects good formatting for multiple hosts", func() {
				disruptionSpec := NetworkDisruptionSpec{
					Hosts: []NetworkDisruptionHostSpec{
						{
							Host: "1.2.3.4",
							Port: 9000,
						},
						{
							Host: "2.2.3.4",
							Port: 8000,
							Flow: "ingress",
						},
					},
					Drop: 100,
				}

				expected := "Network disruption dropping 100% of the traffic going to 1.2.3.4:9000 and coming from 2.2.3.4:8000"
				result := disruptionSpec.Format()

				Expect(result).To(Equal(expected))
			})

			It("expects good formatting for multiple services", func() {
				disruptionSpec := NetworkDisruptionSpec{
					Services: []NetworkDisruptionServiceSpec{
						{
							Name:      "demo-service",
							Namespace: "demo-namespace",
						},
						{
							Name:      "demo-worker",
							Namespace: "demo-namespace",
							Ports: []NetworkDisruptionServicePortSpec{
								{
									Name: "worker-port",
									Port: 8180,
								},
							},
						},
					},
					Corrupt: 100,
				}

				expected := "Network disruption corrupting 100% of the traffic going to demo-service/demo-namespace and going to demo-worker/demo-namespace on port worker-port/8180"
				result := disruptionSpec.Format()

				Expect(result).To(Equal(expected))
			})

			It("expects good formatting for cloud network disruption", func() {
				disruptionSpec := NetworkDisruptionSpec{
					Cloud: &NetworkDisruptionCloudSpec{
						AWSServiceList: &[]NetworkDisruptionCloudServiceSpec{
							{
								ServiceName: "S3",
							},
						},
						DatadogServiceList: &[]NetworkDisruptionCloudServiceSpec{
							{
								ServiceName: "synthetics",
							},
						},
					},
					Delay:       100,
					DelayJitter: 50,
				}

				expected := "Network disruption delaying of 100ms the traffic with 50ms of delay jitter going to S3 and going to synthetics"
				result := disruptionSpec.Format()

				Expect(result).To(Equal(expected))
			})

			It("expects good formatting for a big network disruption", func() {
				disruptionSpec := NetworkDisruptionSpec{
					Hosts: []NetworkDisruptionHostSpec{
						{
							Host: "1.2.3.4",
							Port: 9000,
						},
						{
							Host: "2.2.3.4",
							Port: 8000,
							Flow: "ingress",
						},
					},
					Services: []NetworkDisruptionServiceSpec{
						{
							Name:      "demo-service",
							Namespace: "demo-namespace",
							Ports: []NetworkDisruptionServicePortSpec{
								{
									Name: "demo-service-port",
									Port: 8180,
								},
							},
						},
						{
							Name:      "demo-worker",
							Namespace: "demo-namespace",
						},
					},
					Cloud: &NetworkDisruptionCloudSpec{
						AWSServiceList: &[]NetworkDisruptionCloudServiceSpec{
							{
								ServiceName: "S3",
							},
						},
						DatadogServiceList: &[]NetworkDisruptionCloudServiceSpec{
							{
								ServiceName: "synthetics",
							},
						},
					},
					Corrupt: 100,
				}

				expected := "Network disruption corrupting 100% of the traffic going to 1.2.3.4:9000, coming from 2.2.3.4:8000, going to demo-service/demo-namespace on port demo-service-port/8180, going to demo-worker/demo-namespace, going to S3 and going to synthetics"
				result := disruptionSpec.Format()

				Expect(result).To(Equal(expected))
			})

			It("expects no formatting for empty network disruption", func() {
				disruptionSpec := NetworkDisruptionSpec{
					Hosts:    []NetworkDisruptionHostSpec{},
					Services: []NetworkDisruptionServiceSpec{},
				}

				expected := ""
				result := disruptionSpec.Format()

				Expect(result).To(Equal(expected))
			})
		})

	})
	When("'Validate' method is called", func() {
		Describe("test path field cases", func() {
			DescribeTable("with valid paths",
				func(path string) {
					// Arrange
					disruptionSpec := NetworkDisruptionSpec{
						HTTP: &NetworkHTTPFilters{
							Path: path,
						},
					}

					// Action
					err := disruptionSpec.Validate()

					// Assert
					Expect(err).ShouldNot(HaveOccurred())
				},
				Entry("with a random path",
					"/"+randStringRunes(99),
				),
				Entry("with a simple path",
					"/",
				),
			)
			DescribeTable("with invalid paths",
				func(invalidPath, expectedErrorMessage string) {
					// Arrange
					disruptionSpec := NetworkDisruptionSpec{
						HTTP: &NetworkHTTPFilters{
							Path: invalidPath,
						},
					}

					// Action
					err := disruptionSpec.Validate()

					// Assert
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(ContainSubstring("the path specification at the network disruption level is not valid; " + expectedErrorMessage))
				},
				Entry("When the path exceeding the limit",
					"/"+randStringRunes(100),
					"should not exceed 100 characters",
				),
				Entry("When the path does not start with /",
					"invalid-path",
					"should start with a /",
				),
				Entry("When the path contains spaces",
					"/invalid path",
					SpacesErrorMessagePrefix,
				),
				Entry("When the path is empty",
					"  ",
					SpacesErrorMessagePrefix,
				),
				Entry("When the path contains a spaces at the end",
					"/ ",
					SpacesErrorMessagePrefix,
				),
				Entry("When the path contains a spaces at the start",
					" /",
					SpacesErrorMessagePrefix,
				),
			)
		})
		Describe("test deprecated fields cases", func() {
			port := 8080
			DescribeTable("with deprecated field defined",
				func(invalidDisruptionSpec NetworkDisruptionSpec, expectedErrorMessage string) {
					// Action
					err := invalidDisruptionSpec.Validate()

					// Assert
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(ContainSubstring(expectedErrorMessage))
				},
				Entry("When the DeprecatedPort is defined",
					NetworkDisruptionSpec{DeprecatedPort: &port},
					"the port specification at the network disruption level is deprecated; apply to network disruption hosts instead",
				),
				Entry("When the DeprecatedFlow is defined",
					NetworkDisruptionSpec{DeprecatedFlow: "lorem"},
					"the flow specification at the network disruption level is deprecated; apply to network disruption hosts instead",
				),
			)
		})
	})
	When("'HasHTTPFilters' method is called", func() {
		Context("with a nil NetworkHTTPFilters field", func() {
			It("should return false", func() {
				disruptionSpec := NetworkDisruptionSpec{}

				// Action && Assert
				Expect(disruptionSpec.HasHTTPFilters()).To(BeFalse())
			})
		})
		Context("with default method and path", func() {
			It("should return false", func() {
				// Arrange
				disruptionSpec := NetworkDisruptionSpec{
					HTTP: &NetworkHTTPFilters{
						Method: DefaultHTTPMethodFilter,
						Path:   DefaultHTTPPathFilter,
					},
				}

				// Action && Assert
				Expect(disruptionSpec.HasHTTPFilters()).To(BeFalse())
			})
		})
		DescribeTable("with custom method and/or path",
			func(path, method string) {
				// Arrange
				disruptionSpec := NetworkDisruptionSpec{
					HTTP: &NetworkHTTPFilters{
						Method: method,
						Path:   path,
					},
				}

				// Action && Assert
				Expect(disruptionSpec.HasHTTPFilters()).Should(BeTrue())
			},
			Entry("custom method", DefaultHTTPPathFilter, "get"),
			Entry("custom path", "/test", DefaultHTTPMethodFilter),
			Entry("custom path and method", "/test", "delete"),
		)
	})
})

func randStringRunes(n int) string {
	letterRunes := []rune("/abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
