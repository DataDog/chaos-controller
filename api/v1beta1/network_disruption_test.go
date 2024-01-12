// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package v1beta1

import (
	"fmt"
	"math/rand"
	"net/http"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const SpacesErrorMessagePrefix = "should not contains spaces"

var _ = Describe("NetworkDisruptionSpec", func() {
	When("'Format' method is called", func() {
		Context("with filters", func() {
			It("expects good formatting for multiple hosts", func() {
				// Arrange
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

				// Action
				result := disruptionSpec.Format()

				// Assert
				Expect(result).To(Equal(expected))
			})

			It("expects good formatting for multiple services", func() {
				// Arrange
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

				expected := "Network disruption corrupting 100% of the traffic going to demo-service/demo-namespace and going to demo-worker/demo-namespace on port(s) worker-port/8180"

				// Action
				result := disruptionSpec.Format()

				// Assert
				Expect(result).To(Equal(expected))
			})

			It("expects good formatting for cloud network disruption", func() {
				// Arrange
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

				// Action
				result := disruptionSpec.Format()

				// Assert
				Expect(result).To(Equal(expected))
			})

			It("expects good formatting for a big network disruption", func() {
				// Arrange
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

				expected := "Network disruption corrupting 100% of the traffic going to 1.2.3.4:9000, coming from 2.2.3.4:8000, going to demo-service/demo-namespace on port(s) demo-service-port/8180, going to demo-worker/demo-namespace, going to S3 and going to synthetics"

				// Action
				result := disruptionSpec.Format()

				// Assert
				Expect(result).To(Equal(expected))
			})

			It("expects no formatting for empty network disruption", func() {
				// Arrange
				disruptionSpec := NetworkDisruptionSpec{
					Hosts:    []NetworkDisruptionHostSpec{},
					Services: []NetworkDisruptionServiceSpec{},
				}

				expected := ""

				// Action
				result := disruptionSpec.Format()

				// Assert
				Expect(result).To(Equal(expected))
			})
		})

	})

	When("'Validate' method is called", func() {
		Describe("test Paths field cases", func() {
			DescribeTable("with valid paths",
				func(paths HTTPPaths) {
					// Arrange
					disruptionSpec := NetworkDisruptionSpec{
						HTTP: &NetworkHTTPFilters{
							Paths: paths,
						},
					}

					// Action
					err := disruptionSpec.Validate()

					// Assert
					Expect(err).ShouldNot(HaveOccurred())
				},
				Entry("with a random path",
					HTTPPaths{
						HTTPPath(DefaultHTTPPathFilter + randStringRunes(MaxNetworkPathCharacters-1)),
					},
				),
				Entry("with a simple path",
					HTTPPaths{
						DefaultHTTPPathFilter,
					},
				),
				Entry("with multiple valid paths",
					HTTPPaths{
						"/lorem",
						"/ipsum",
						"/lorem/ipsum",
					},
				),
			)

			Describe("error cases", func() {
				DescribeTable("with invalid paths",
					func(invalidPaths HTTPPaths, expectedErrorMessage string) {
						// Arrange
						disruptionSpec := NetworkDisruptionSpec{
							HTTP: &NetworkHTTPFilters{
								Paths: invalidPaths,
							},
						}

						// Action
						err := disruptionSpec.Validate()

						// Assert
						Expect(err).Should(HaveOccurred())
						Expect(err.Error()).Should(ContainSubstring(HTTPPathsFilterErrorPrefix + expectedErrorMessage))
					},
					Entry("When the path exceeding the limit",
						HTTPPaths{HTTPPath("/" + randStringRunes(MaxNetworkPathCharacters))},
						fmt.Sprintf("should not exceed %d characters", MaxNetworkPathCharacters),
					),
					Entry("When the path does not start with /",
						HTTPPaths{"invalid-path"},
						"should start with a /",
					),
					Entry("When the path contains spaces",
						HTTPPaths{"/invalid path"},
						SpacesErrorMessagePrefix,
					),
					Entry("When the path is empty",
						HTTPPaths{"  "},
						SpacesErrorMessagePrefix,
					),
					Entry("When the path contains a spaces at the end",
						HTTPPaths{"/ "},
						SpacesErrorMessagePrefix,
					),
					Entry("When the path contains a spaces at the start",
						HTTPPaths{" /"},
						SpacesErrorMessagePrefix,
					),
					Entry("With two identical paths",
						HTTPPaths{DefaultHTTPPathFilter, DefaultHTTPPathFilter},
						"should not contain duplicated paths. Count: 2; Path: "+DefaultHTTPPathFilter,
					),
					Entry("With the default path and another path",
						HTTPPaths{DefaultHTTPPathFilter, "/test"},
						"no needs to define other paths if the / path is defined because it already catches all paths",
					),
					Entry("When the number of paths are greater than the limit",
						func() (httpPaths HTTPPaths) {
							for i := 0; i < MaxNetworkPaths+1; i++ {
								httpPaths = append(httpPaths, HTTPPath(fmt.Sprintf("/path%d", i)))
							}
							return httpPaths
						}(),
						fmt.Sprintf("the number of paths must not be greater than %d; Number of paths: %d", MaxNetworkPaths, MaxNetworkPaths+1),
					),
				)

				Describe("with more than two identical paths", func() {
					It("should not print multiple time the duplicate error", func() {
						// Arrange
						duplicatedPaths := HTTPPaths{
							DefaultHTTPPathFilter,
							DefaultHTTPPathFilter,
							DefaultHTTPPathFilter,
						}

						disruptionSpec := NetworkDisruptionSpec{
							HTTP: &NetworkHTTPFilters{
								Paths: duplicatedPaths,
							},
						}

						// Action
						err := disruptionSpec.Validate()

						// Assert
						Expect(err).Should(HaveOccurred())
						errorMessage := err.Error()
						expectedMessage := "should not contain duplicated paths. Count: 3; Path: " + DefaultHTTPPathFilter
						Expect(strings.Count(errorMessage, expectedMessage)).To(Equal(1))
					})
				})
			})
		})

		Describe("test deprecated fields cases", func() {
			port := 8080
			DescribeTable("with deprecated field defined",
				func(invalidDisruptionSpec NetworkDisruptionSpec, expectedErrorMessage string) {
					// Action
					err := invalidDisruptionSpec.Validate()

					// Assert
					Expect(err).Should(HaveOccurred())
					errorMessage := err.Error()
					Expect(strings.Count(errorMessage, expectedErrorMessage)).To(Equal(1))
				},
				Entry("When the DeprecatedPort is defined",
					NetworkDisruptionSpec{DeprecatedPort: &port},
					"the port specification at the network disruption level is deprecated; apply to network disruption hosts instead",
				),
				Entry("When the DeprecatedFlow is defined",
					NetworkDisruptionSpec{DeprecatedFlow: "lorem"},
					"the flow specification at the network disruption level is deprecated; apply to network disruption hosts instead",
				),
				Entry("When the DeprecatedMethod is defined",
					NetworkDisruptionSpec{
						HTTP: &NetworkHTTPFilters{
							DeprecatedMethod: "ALL",
						},
					},
					"the Method specification at the HTTP network disruption level is no longer supported; use Methods HTTP field instead",
				),
				Entry("When the DeprecatedPath is defined",
					NetworkDisruptionSpec{
						HTTP: &NetworkHTTPFilters{
							DeprecatedPath: DefaultHTTPPathFilter,
						},
					},
					"the Path specification at the HTTP network disruption level is no longer supported; use Paths HTTP field instead",
				),
			)
		})

		Describe("test Methods field cases", func() {
			DescribeTable("with valid methods",
				func(methods HTTPMethods) {
					// Arrange
					disruptionSpec := NetworkDisruptionSpec{
						HTTP: &NetworkHTTPFilters{
							Methods: methods,
						},
					}

					// Action
					err := disruptionSpec.Validate()

					// Assert
					Expect(err).ShouldNot(HaveOccurred())
				},
				Entry("with an empty method", nil),
				Entry("with a single method",
					HTTPMethods{
						http.MethodPost,
					},
				),
				Entry("with all methods",
					HTTPMethods{
						http.MethodPost,
						http.MethodConnect,
						http.MethodDelete,
						http.MethodGet,
						http.MethodHead,
						http.MethodPatch,
						http.MethodOptions,
						http.MethodTrace,
						http.MethodPut,
					},
				),
			)

			Describe("error cases", func() {
				DescribeTable("with invalid methods",
					func(methods HTTPMethods, expectedErrorMessage string) {
						// Arrange
						disruptionSpec := NetworkDisruptionSpec{
							HTTP: &NetworkHTTPFilters{
								Methods: methods,
							},
						}

						// Action
						err := disruptionSpec.Validate()

						// Assert
						Expect(err).Should(HaveOccurred())
						Expect(err.Error()).Should(ContainSubstring(HTTPMethodsFilterErrorPrefix + expectedErrorMessage))
					},
					Entry("When the methods is not an http method",
						HTTPMethods{"lorem"},
						"should be a GET, DELETE, POST, PUT, HEAD, PATCH, CONNECT, OPTIONS or TRACE. Invalid value: lorem",
					),
					Entry("When the methods contains duplicate",
						HTTPMethods{http.MethodPost, http.MethodPost},
						"should not contain duplicated methods. Count: 2; Method: POST",
					),
					Entry("When the number of methods are greater than the limit",
						func() (httpMethods HTTPMethods) {
							for i := 0; i < MaxNetworkMethods+1; i++ {
								httpMethods = append(httpMethods, fmt.Sprintf("unknow%d", i))
							}
							return httpMethods
						}(),
						fmt.Sprintf("the number of methods must not be greater than %d; Number of methods: %d", MaxNetworkMethods, MaxNetworkMethods+1),
					),
				)

				Describe("with more than two identical methods", func() {
					It("should not print multiple time the duplicate error", func() {
						// Arrange
						duplicatedMethods := HTTPMethods{
							http.MethodPut,
							http.MethodPut,
							http.MethodPut,
						}

						disruptionSpec := NetworkDisruptionSpec{
							HTTP: &NetworkHTTPFilters{
								Methods: duplicatedMethods,
							},
						}

						// Action
						err := disruptionSpec.Validate()

						// Assert
						Expect(err).Should(HaveOccurred())
						errorMessage := err.Error()
						expectedMessage := "should not contain duplicated methods. Count: 3; Method: " + http.MethodPut
						Expect(strings.Count(errorMessage, expectedMessage)).To(Equal(1))
					})
				})
			})
		})

		Describe("test option limits", func() {
			It("rejects bandwidthLimits below 32 bytes", func() {
				disruptionSpec := NetworkDisruptionSpec{BandwidthLimit: 0}
				err := disruptionSpec.Validate()
				Expect(err).ShouldNot(HaveOccurred())

				disruptionSpec.BandwidthLimit = 32
				err = disruptionSpec.Validate()
				Expect(err).ShouldNot(HaveOccurred())

				disruptionSpec.BandwidthLimit = 16
				err = disruptionSpec.Validate()
				Expect(err).Should(HaveOccurred())
			})
		})

		DescribeTable("multi error cases",
			func(invalidPaths HTTPPaths, invalidMethods HTTPMethods, expectedErrorMessages []string) {
				// Arrange
				disruptionSpec := NetworkDisruptionSpec{
					HTTP: &NetworkHTTPFilters{
						Paths:   invalidPaths,
						Methods: invalidMethods,
					},
				}

				// Action
				err := disruptionSpec.Validate()

				// Assert
				Expect(err).Should(HaveOccurred())
				for _, expectedErrorMessage := range expectedErrorMessages {
					Expect(err.Error()).Should(ContainSubstring(expectedErrorMessage))
				}
			},
			Entry("When the path exceeding the limit and the methods does not exists",
				HTTPPaths{HTTPPath("/" + randStringRunes(MaxNetworkPathCharacters))},
				HTTPMethods{"lorem"},
				[]string{
					fmt.Sprintf("should not exceed %d characters", MaxNetworkPathCharacters),
					"should be a GET, DELETE, POST, PUT, HEAD, PATCH, CONNECT, OPTIONS or TRACE. Invalid value: lorem",
				},
			),
			Entry("When the path does not start with a / and the methods are duplicated",
				HTTPPaths{"invalid-path"},
				HTTPMethods{http.MethodPut, http.MethodPut},
				[]string{
					"should start with a /",
					"should not contain duplicated methods. Count: 2; Method: PUT",
				},
			),
			Entry("with a valid path and duplicated methods and a non http method",
				HTTPPaths{DefaultHTTPPathFilter},
				HTTPMethods{http.MethodDelete, http.MethodDelete, "lorem"},
				[]string{
					"should be a GET, DELETE, POST, PUT, HEAD, PATCH, CONNECT, OPTIONS or TRACE. Invalid value: lorem",
					"should not contain duplicated methods. Count: 2; Method: DELETE",
				},
			),
			Entry("with a valid path and multiple duplicated methods",
				HTTPPaths{DefaultHTTPPathFilter},
				HTTPMethods{http.MethodDelete, http.MethodDelete, http.MethodGet, http.MethodGet, http.MethodGet},
				[]string{
					"should not contain duplicated methods. Count: 3; Method: GET",
					"should not contain duplicated methods. Count: 2; Method: DELETE",
				},
			),
		)

	})

	When("'HasHTTPFilters' method is called", func() {
		Context("with a nil NetworkHTTPFilters field", func() {
			It("should return false", func() {
				// Arrange
				disruptionSpec := NetworkDisruptionSpec{}

				// Action && Assert
				Expect(disruptionSpec.HasHTTPFilters()).To(BeFalse())
			})
		})

		Context("without methods and default path", func() {
			It("should return false", func() {
				// Arrange
				disruptionSpec := NetworkDisruptionSpec{
					HTTP: &NetworkHTTPFilters{
						Paths: HTTPPaths{DefaultHTTPPathFilter},
					},
				}

				// Action && Assert
				Expect(disruptionSpec.HasHTTPFilters()).To(BeFalse())
			})
		})

		DescribeTable("with custom method and/or path",
			func(paths HTTPPaths, methods HTTPMethods) {
				// Arrange
				disruptionSpec := NetworkDisruptionSpec{
					HTTP: &NetworkHTTPFilters{
						Methods: methods,
						Paths:   paths,
					},
				}

				// Action && Assert
				Expect(disruptionSpec.HasHTTPFilters()).Should(BeTrue())
			},
			Entry("custom method", HTTPPaths{DefaultHTTPPathFilter}, HTTPMethods{http.MethodPut}),
			Entry("custom path", HTTPPaths{"/test"}, HTTPMethods{}),
			Entry("custom path and method", HTTPPaths{"/test"}, HTTPMethods{http.MethodDelete}),
			Entry("all custom methods", HTTPPaths{DefaultHTTPPathFilter}, HTTPMethods{
				http.MethodPost,
				http.MethodConnect,
				http.MethodDelete,
				http.MethodGet,
				http.MethodHead,
				http.MethodPatch,
				http.MethodOptions,
				http.MethodTrace,
				http.MethodPut,
			}),
		)
	})

	When("NetworkDisruptionServiceSpecFromString is called", func() {
		It("handles ports with non-alpha names", func() {
			// Arrange
			expected := []NetworkDisruptionServiceSpec{{
				Name:      "demo-service",
				Namespace: "demo-namespace",
				Ports: []NetworkDisruptionServicePortSpec{
					{
						Name: "demo-port",
						Port: 8080,
					},
					{
						Port: 8180,
					},
				},
			}}

			testString := []string{"demo-service;demo-namespace;8080-demo-port;8180-"}

			// Action
			actual, err := NetworkDisruptionServiceSpecFromString(testString)

			// Assert
			Expect(err).ShouldNot(HaveOccurred())
			Expect(actual).Should(Equal(expected))
		})
	})

	When("'GenerateArgs' method is called", func() {

		var (
			defaultNetworkDisruption = NetworkDisruptionSpec{
				Hosts: []NetworkDisruptionHostSpec{
					{
						Host:      "lorem",
						Port:      8080,
						Protocol:  "TCP",
						Flow:      "ingress",
						ConnState: "open",
					},
				},
				AllowedHosts: []NetworkDisruptionHostSpec{
					{
						Host:      "localhost",
						Port:      9090,
						Protocol:  "UDP",
						Flow:      "egress",
						ConnState: "closed",
					},
				},
				DisableDefaultAllowedHosts: true,
				Services: []NetworkDisruptionServiceSpec{
					{
						Name:      "name",
						Namespace: "namespace",
						Ports: []NetworkDisruptionServicePortSpec{
							{
								Name: "default",
								Port: 9191,
							},
						},
					},
				},
				Drop:           1,
				Duplicate:      2,
				Corrupt:        3,
				Delay:          4,
				DelayJitter:    5,
				BandwidthLimit: 6,
			}
			defaultExpectedArgs = []string{
				"network-disruption",
				"--corrupt",
				"3",
				"--drop",
				"1",
				"--duplicate",
				"2",
				"--delay",
				"4",
				"--delay-jitter",
				"5",
				"--bandwidth-limit",
				"6",
				"--hosts",
				"lorem;8080;TCP;ingress;open",
				"--allowed-hosts",
				"localhost;9090;UDP;egress;closed",
				"--services",
				"name;namespace;9191-default",
			}
		)

		DescribeTable("success cases", func(disruption NetworkDisruptionSpec, expectedArgs []string) {
			// Action
			args := disruption.GenerateArgs()

			// Assert
			Expect(args).Should(Equal(expectedArgs))
		},
			Entry("with a regular disruption",
				defaultNetworkDisruption,
				defaultExpectedArgs,
			),
			Entry("with an HTTP filter",
				func() NetworkDisruptionSpec {
					networkDisruption := defaultNetworkDisruption.DeepCopy()
					networkDisruption.HTTP = &NetworkHTTPFilters{
						Paths: HTTPPaths{DefaultHTTPPathFilter},
						Methods: HTTPMethods{
							http.MethodPost,
							http.MethodPut,
						},
					}

					return *networkDisruption
				}(),
				func() []string {
					expectedArgs := defaultExpectedArgs
					expectedArgs = append(expectedArgs, "--path", DefaultHTTPPathFilter)
					expectedArgs = append(expectedArgs, "--method", http.MethodPost, "--method", http.MethodPut)

					return expectedArgs
				}(),
			),
			Entry("with an HTTP path filter",
				func() NetworkDisruptionSpec {
					networkDisruption := defaultNetworkDisruption.DeepCopy()
					networkDisruption.HTTP = &NetworkHTTPFilters{
						Paths: HTTPPaths{DefaultHTTPPathFilter},
					}

					return *networkDisruption
				}(),
				func() []string {
					expectedArgs := defaultExpectedArgs
					expectedArgs = append(expectedArgs, "--path", DefaultHTTPPathFilter)

					return expectedArgs
				}(),
			),
			Entry("with an HTTP methods filter",
				func() NetworkDisruptionSpec {
					networkDisruption := defaultNetworkDisruption.DeepCopy()
					networkDisruption.HTTP = &NetworkHTTPFilters{
						Methods: HTTPMethods{
							http.MethodPost,
							http.MethodPut,
						},
					}

					return *networkDisruption
				}(),
				func() []string {
					expectedArgs := defaultExpectedArgs
					expectedArgs = append(expectedArgs, "--method", http.MethodPost, "--method", http.MethodPut)

					return expectedArgs
				}(),
			))
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
