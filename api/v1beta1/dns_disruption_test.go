// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package v1beta1_test

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DNSDisruptionSpec", func() {
	Describe("Validate", func() {
		Context("Success cases", func() {
			It("should validate spec with nxdomain failure mode", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Domains:     []string{"example.com"},
					FailureMode: "nxdomain",
				}
				Expect(spec.Validate()).NotTo(HaveOccurred())
			})

			It("should validate spec with drop failure mode", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Domains:     []string{"example.com", "api.example.com"},
					FailureMode: "drop",
				}
				Expect(spec.Validate()).NotTo(HaveOccurred())
			})

			It("should validate spec with servfail failure mode", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Domains:     []string{"test.com"},
					FailureMode: "servfail",
				}
				Expect(spec.Validate()).NotTo(HaveOccurred())
			})

			It("should validate spec with random-ip failure mode", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Domains:     []string{"api.production.com"},
					FailureMode: "random-ip",
				}
				Expect(spec.Validate()).NotTo(HaveOccurred())
			})

			It("should validate spec with custom port", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Domains:     []string{"example.com"},
					FailureMode: "nxdomain",
					Port:        5353,
				}
				Expect(spec.Validate()).NotTo(HaveOccurred())
			})

			It("should validate spec with udp protocol", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Domains:     []string{"example.com"},
					FailureMode: "nxdomain",
					Protocol:    "udp",
				}
				Expect(spec.Validate()).NotTo(HaveOccurred())
			})

			It("should validate spec with tcp protocol", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Domains:     []string{"example.com"},
					FailureMode: "nxdomain",
					Protocol:    "tcp",
				}
				Expect(spec.Validate()).NotTo(HaveOccurred())
			})

			It("should validate spec with both protocols", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Domains:     []string{"example.com"},
					FailureMode: "nxdomain",
					Protocol:    "both",
				}
				Expect(spec.Validate()).NotTo(HaveOccurred())
			})

			It("should validate spec with all optional fields", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Domains:     []string{"example.com", "test.com"},
					FailureMode: "servfail",
					Port:        8053,
					Protocol:    "udp",
				}
				Expect(spec.Validate()).NotTo(HaveOccurred())
			})

			It("should accept case-insensitive failure modes", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Domains:     []string{"example.com"},
					FailureMode: "NXDOMAIN",
				}
				Expect(spec.Validate()).NotTo(HaveOccurred())
			})

			It("should accept case-insensitive protocols", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Domains:     []string{"example.com"},
					FailureMode: "drop",
					Protocol:    "UDP",
				}
				Expect(spec.Validate()).NotTo(HaveOccurred())
			})
		})

		Context("Error cases", func() {
			It("should return error when domains list is empty", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Domains:     []string{},
					FailureMode: "nxdomain",
				}
				err := spec.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("must contain at least one domain"))
			})

			It("should return error when domains list contains empty string", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Domains:     []string{"example.com", "", "test.com"},
					FailureMode: "nxdomain",
				}
				err := spec.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("cannot be empty"))
			})

			It("should return error when domains list contains whitespace-only string", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Domains:     []string{"example.com", "   ", "test.com"},
					FailureMode: "nxdomain",
				}
				err := spec.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("cannot be empty"))
			})

			It("should return error for invalid failure mode", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Domains:     []string{"example.com"},
					FailureMode: "invalid-mode",
				}
				err := spec.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("must be one of"))
			})

			It("should return error when port is too low", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Domains:     []string{"example.com"},
					FailureMode: "nxdomain",
					Port:        0,
				}
				err := spec.Validate()
				Expect(err).NotTo(HaveOccurred()) // Port 0 is treated as default

				spec.Port = -1
				err = spec.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("must be between 1 and 65535"))
			})

			It("should return error when port is too high", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Domains:     []string{"example.com"},
					FailureMode: "nxdomain",
					Port:        65536,
				}
				err := spec.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("must be between 1 and 65535"))
			})

			It("should return error for invalid protocol", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Domains:     []string{"example.com"},
					FailureMode: "nxdomain",
					Protocol:    "http",
				}
				err := spec.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("must be one of"))
			})

			It("should return multiple errors when multiple fields are invalid", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Domains:     []string{},
					FailureMode: "invalid",
					Port:        70000,
					Protocol:    "invalid-proto",
				}
				err := spec.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("domains"))
				Expect(err.Error()).To(ContainSubstring("failureMode"))
				Expect(err.Error()).To(ContainSubstring("port"))
				Expect(err.Error()).To(ContainSubstring("protocol"))
			})
		})
	})

	Describe("GenerateArgs", func() {
		Context("Success cases", func() {
			It("should generate correct arguments for nxdomain mode", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Domains:     []string{"example.com"},
					FailureMode: "nxdomain",
				}
				args := spec.GenerateArgs()
				Expect(args).To(ContainElement("dns-disruption"))
				Expect(args).To(ContainElement("--domains"))
				Expect(args).To(ContainElement("example.com"))
				Expect(args).To(ContainElement("--failure-mode"))
				Expect(args).To(ContainElement("nxdomain"))
				Expect(args).To(ContainElement("--port"))
				Expect(args).To(ContainElement("53"))
				Expect(args).To(ContainElement("--protocol"))
				Expect(args).To(ContainElement("both"))
			})

			It("should generate correct arguments for drop mode", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Domains:     []string{"example.com"},
					FailureMode: "drop",
				}
				args := spec.GenerateArgs()
				Expect(args).To(ContainElement("--failure-mode"))
				Expect(args).To(ContainElement("drop"))
			})

			It("should generate correct arguments for servfail mode", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Domains:     []string{"example.com"},
					FailureMode: "servfail",
				}
				args := spec.GenerateArgs()
				Expect(args).To(ContainElement("--failure-mode"))
				Expect(args).To(ContainElement("servfail"))
			})

			It("should generate correct arguments for random-ip mode", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Domains:     []string{"example.com"},
					FailureMode: "random-ip",
				}
				args := spec.GenerateArgs()
				Expect(args).To(ContainElement("--failure-mode"))
				Expect(args).To(ContainElement("random-ip"))
			})

			It("should generate comma-separated domains for multiple domains", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Domains:     []string{"example.com", "api.example.com", "test.com"},
					FailureMode: "nxdomain",
				}
				args := spec.GenerateArgs()
				Expect(args).To(ContainElement("--domains"))
				Expect(args).To(ContainElement("example.com,api.example.com,test.com"))
			})

			It("should use custom port when specified", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Domains:     []string{"example.com"},
					FailureMode: "nxdomain",
					Port:        5353,
				}
				args := spec.GenerateArgs()
				Expect(args).To(ContainElement("--port"))
				Expect(args).To(ContainElement("5353"))
			})

			It("should use default port 53 when not specified", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Domains:     []string{"example.com"},
					FailureMode: "nxdomain",
				}
				args := spec.GenerateArgs()
				Expect(args).To(ContainElement("--port"))
				Expect(args).To(ContainElement("53"))
			})

			It("should use udp protocol when specified", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Domains:     []string{"example.com"},
					FailureMode: "nxdomain",
					Protocol:    "udp",
				}
				args := spec.GenerateArgs()
				Expect(args).To(ContainElement("--protocol"))
				Expect(args).To(ContainElement("udp"))
			})

			It("should use tcp protocol when specified", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Domains:     []string{"example.com"},
					FailureMode: "nxdomain",
					Protocol:    "tcp",
				}
				args := spec.GenerateArgs()
				Expect(args).To(ContainElement("--protocol"))
				Expect(args).To(ContainElement("tcp"))
			})

			It("should use default 'both' protocol when not specified", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Domains:     []string{"example.com"},
					FailureMode: "nxdomain",
				}
				args := spec.GenerateArgs()
				Expect(args).To(ContainElement("--protocol"))
				Expect(args).To(ContainElement("both"))
			})

			It("should normalize failure mode to lowercase", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Domains:     []string{"example.com"},
					FailureMode: "NXDOMAIN",
				}
				args := spec.GenerateArgs()
				Expect(args).To(ContainElement("nxdomain"))
			})

			It("should normalize protocol to lowercase", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Domains:     []string{"example.com"},
					FailureMode: "nxdomain",
					Protocol:    "UDP",
				}
				args := spec.GenerateArgs()
				Expect(args).To(ContainElement("udp"))
			})
		})
	})

	Describe("Explain", func() {
		It("should provide explanation for nxdomain mode", func() {
			spec := &v1beta1.DNSDisruptionSpec{
				Domains:     []string{"example.com"},
				FailureMode: "nxdomain",
			}
			explanation := spec.Explain()
			Expect(explanation).To(HaveLen(2))
			Expect(explanation[1]).To(ContainSubstring("example.com"))
			Expect(explanation[1]).To(ContainSubstring("NXDOMAIN"))
			Expect(explanation[1]).To(ContainSubstring("domain does not exist"))
		})

		It("should provide explanation for drop mode", func() {
			spec := &v1beta1.DNSDisruptionSpec{
				Domains:     []string{"example.com"},
				FailureMode: "drop",
			}
			explanation := spec.Explain()
			Expect(explanation[1]).To(ContainSubstring("dropped"))
			Expect(explanation[1]).To(ContainSubstring("timeout"))
		})

		It("should provide explanation for servfail mode", func() {
			spec := &v1beta1.DNSDisruptionSpec{
				Domains:     []string{"example.com"},
				FailureMode: "servfail",
			}
			explanation := spec.Explain()
			Expect(explanation[1]).To(ContainSubstring("SERVFAIL"))
			Expect(explanation[1]).To(ContainSubstring("server failure"))
		})

		It("should provide explanation for random-ip mode", func() {
			spec := &v1beta1.DNSDisruptionSpec{
				Domains:     []string{"example.com"},
				FailureMode: "random-ip",
			}
			explanation := spec.Explain()
			Expect(explanation[1]).To(ContainSubstring("random"))
			Expect(explanation[1]).To(ContainSubstring("IP"))
		})

		It("should list multiple domains in explanation", func() {
			spec := &v1beta1.DNSDisruptionSpec{
				Domains:     []string{"example.com", "api.example.com", "test.com"},
				FailureMode: "nxdomain",
			}
			explanation := spec.Explain()
			Expect(explanation[1]).To(ContainSubstring("example.com"))
			Expect(explanation[1]).To(ContainSubstring("api.example.com"))
			Expect(explanation[1]).To(ContainSubstring("test.com"))
		})

		It("should mention custom port in explanation", func() {
			spec := &v1beta1.DNSDisruptionSpec{
				Domains:     []string{"example.com"},
				FailureMode: "nxdomain",
				Port:        8053,
			}
			explanation := spec.Explain()
			Expect(explanation[1]).To(ContainSubstring("8053"))
		})

		It("should mention protocol in explanation", func() {
			spec := &v1beta1.DNSDisruptionSpec{
				Domains:     []string{"example.com"},
				FailureMode: "nxdomain",
				Protocol:    "udp",
			}
			explanation := spec.Explain()
			Expect(explanation[1]).To(ContainSubstring("udp"))
		})
	})
})
