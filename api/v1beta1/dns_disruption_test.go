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
		Context("Success cases - New record-based format", func() {
			It("should validate A record with single IP", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Records: []v1beta1.DNSRecord{
						{
							Hostname: "example.com",
							Record: v1beta1.DNSRecordConfig{
								Type:  "A",
								Value: "192.168.1.1",
								TTL:   60,
							},
						},
					},
				}
				Expect(spec.Validate()).NotTo(HaveOccurred())
			})

			It("should validate A record with multiple IPs for round-robin", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Records: []v1beta1.DNSRecord{
						{
							Hostname: "api.example.com",
							Record: v1beta1.DNSRecordConfig{
								Type:  "A",
								Value: "192.168.1.1,192.168.1.2,192.168.1.3",
								TTL:   30,
							},
						},
					},
				}
				Expect(spec.Validate()).NotTo(HaveOccurred())
			})

			DescribeTable("should validate A and AAAA records with special values",
				func(recordType, specialValue string) {
					spec := &v1beta1.DNSDisruptionSpec{
						Records: []v1beta1.DNSRecord{
							{
								Hostname: "example.com",
								Record: v1beta1.DNSRecordConfig{
									Type:  recordType,
									Value: specialValue,
								},
							},
						},
					}
					Expect(spec.Validate()).NotTo(HaveOccurred())
				},
				Entry("A record with NXDOMAIN", "A", "NXDOMAIN"),
				Entry("A record with DROP", "A", "DROP"),
				Entry("A record with SERVFAIL", "A", "SERVFAIL"),
				Entry("A record with RANDOM", "A", "RANDOM"),
				Entry("AAAA record with NXDOMAIN", "AAAA", "NXDOMAIN"),
				Entry("AAAA record with DROP", "AAAA", "DROP"),
				Entry("AAAA record with SERVFAIL", "AAAA", "SERVFAIL"),
				Entry("AAAA record with RANDOM", "AAAA", "RANDOM"),
			)

			It("should validate AAAA record with IPv6 address", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Records: []v1beta1.DNSRecord{
						{
							Hostname: "ipv6.example.com",
							Record: v1beta1.DNSRecordConfig{
								Type:  "AAAA",
								Value: "2001:db8::1",
								TTL:   120,
							},
						},
					},
				}
				Expect(spec.Validate()).NotTo(HaveOccurred())
			})

			It("should validate AAAA record with multiple IPv6 addresses", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Records: []v1beta1.DNSRecord{
						{
							Hostname: "ipv6.example.com",
							Record: v1beta1.DNSRecordConfig{
								Type:  "AAAA",
								Value: "2001:db8::1,2001:db8::2,2001:db8::3",
								TTL:   60,
							},
						},
					},
				}
				Expect(spec.Validate()).NotTo(HaveOccurred())
			})

			DescribeTable("should validate different DNS record types",
				func(hostname, recordType, value string, ttl uint32) {
					spec := &v1beta1.DNSDisruptionSpec{
						Records: []v1beta1.DNSRecord{
							{
								Hostname: hostname,
								Record: v1beta1.DNSRecordConfig{
									Type:  recordType,
									Value: value,
									TTL:   ttl,
								},
							},
						},
					}
					Expect(spec.Validate()).NotTo(HaveOccurred())
				},
				Entry("CNAME record", "alias.example.com", "CNAME", "target.example.com", uint32(300)),
				Entry("MX record", "example.com", "MX", "10 mail1.example.com,20 mail2.example.com", uint32(3600)),
				Entry("TXT record", "example.com", "TXT", "v=spf1 include:_spf.example.com ~all", uint32(300)),
				Entry("SRV record", "_sip._tcp.example.com", "SRV", "10 60 5060 sipserver.example.com", uint32(600)),
			)

			It("should validate multiple records of different types", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Records: []v1beta1.DNSRecord{
						{
							Hostname: "api.example.com",
							Record: v1beta1.DNSRecordConfig{
								Type:  "A",
								Value: "192.168.1.1,192.168.1.2",
								TTL:   60,
							},
						},
						{
							Hostname: "ipv6.example.com",
							Record: v1beta1.DNSRecordConfig{
								Type:  "AAAA",
								Value: "2001:db8::1",
								TTL:   60,
							},
						},
						{
							Hostname: "alias.example.com",
							Record: v1beta1.DNSRecordConfig{
								Type:  "CNAME",
								Value: "target.example.com",
								TTL:   300,
							},
						},
					},
				}
				Expect(spec.Validate()).NotTo(HaveOccurred())
			})

			It("should validate spec with custom port and protocol", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Records: []v1beta1.DNSRecord{
						{
							Hostname: "example.com",
							Record: v1beta1.DNSRecordConfig{
								Type:  "A",
								Value: "192.168.1.1",
							},
						},
					},
					Port:     5353,
					Protocol: "udp",
				}
				Expect(spec.Validate()).NotTo(HaveOccurred())
			})

			DescribeTable("should accept case-insensitive record types and protocols",
				func(recordType, protocol string) {
					spec := &v1beta1.DNSDisruptionSpec{
						Records: []v1beta1.DNSRecord{
							{
								Hostname: "example.com",
								Record: v1beta1.DNSRecordConfig{
									Type:  recordType,
									Value: "192.168.1.1",
								},
							},
						},
						Protocol: protocol,
					}
					Expect(spec.Validate()).NotTo(HaveOccurred())
				},
				Entry("lowercase A record type", "a", ""),
				Entry("mixed case A record type", "a", ""),
				Entry("uppercase protocol UDP", "A", "UDP"),
				Entry("lowercase protocol tcp", "A", "tcp"),
				Entry("mixed case protocol Both", "A", "Both"),
			)

			It("should trim whitespace and validate hostnames correctly", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Records: []v1beta1.DNSRecord{
						{
							Hostname: "  example.com  ",
							Record: v1beta1.DNSRecordConfig{
								Type:  "A",
								Value: "192.168.1.1",
							},
						},
					},
				}
				Expect(spec.Validate()).NotTo(HaveOccurred())
			})

			It("should validate MX record with multiple entries", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Records: []v1beta1.DNSRecord{
						{
							Hostname: "example.com",
							Record: v1beta1.DNSRecordConfig{
								Type:  "MX",
								Value: "10 mail1.example.com, 20 mail2.example.com, 30 mail3.example.com",
							},
						},
					},
				}
				Expect(spec.Validate()).NotTo(HaveOccurred())
			})

			It("should validate SRV record with multiple entries", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Records: []v1beta1.DNSRecord{
						{
							Hostname: "_sip._tcp.example.com",
							Record: v1beta1.DNSRecordConfig{
								Type:  "SRV",
								Value: "10 60 5060 sipserver1.example.com, 20 40 5060 sipserver2.example.com",
							},
						},
					},
				}
				Expect(spec.Validate()).NotTo(HaveOccurred())
			})
		})

		Context("Error cases", func() {
			DescribeTable("should return error for invalid record configurations",
				func(records []v1beta1.DNSRecord, expectedError string) {
					spec := &v1beta1.DNSDisruptionSpec{
						Records: records,
					}
					err := spec.Validate()
					Expect(err).To(MatchError(ContainSubstring(expectedError)))
				},
				Entry("empty records list", []v1beta1.DNSRecord{}, "records must contain at least one record"),
				Entry("empty hostname", []v1beta1.DNSRecord{
					{
						Hostname: "",
						Record: v1beta1.DNSRecordConfig{
							Type:  "A",
							Value: "192.168.1.1",
						},
					},
				}, "hostname cannot be empty"),
				Entry("invalid record type", []v1beta1.DNSRecord{
					{
						Hostname: "example.com",
						Record: v1beta1.DNSRecordConfig{
							Type:  "INVALID",
							Value: "something",
						},
					},
				}, "must be one of"),
				Entry("empty value", []v1beta1.DNSRecord{
					{
						Hostname: "example.com",
						Record: v1beta1.DNSRecordConfig{
							Type:  "A",
							Value: "",
						},
					},
				}, "value cannot be empty"),
			)

			DescribeTable("should return error for duplicate hostnames",
				func(hostname1, hostname2 string) {
					spec := &v1beta1.DNSDisruptionSpec{
						Records: []v1beta1.DNSRecord{
							{
								Hostname: hostname1,
								Record: v1beta1.DNSRecordConfig{
									Type:  "A",
									Value: "192.168.1.1",
								},
							},
							{
								Hostname: hostname2,
								Record: v1beta1.DNSRecordConfig{
									Type:  "A",
									Value: "NXDOMAIN",
								},
							},
						},
					}
					err := spec.Validate()
					Expect(err).To(MatchError(ContainSubstring("is duplicated")))
				},
				Entry("exact duplicate hostnames", "example.com", "example.com"),
				Entry("case-insensitive duplicate hostnames", "Example.COM", "example.com"),
			)

			DescribeTable("should return error for invalid IP addresses",
				func(recordType, value, expectedError string) {
					spec := &v1beta1.DNSDisruptionSpec{
						Records: []v1beta1.DNSRecord{
							{
								Hostname: "example.com",
								Record: v1beta1.DNSRecordConfig{
									Type:  recordType,
									Value: value,
								},
							},
						},
					}
					err := spec.Validate()
					Expect(err).To(MatchError(ContainSubstring(expectedError)))
				},
				Entry("invalid IPv4 address in A record", "A", "999.999.999.999", "not a valid IPv4"),
				Entry("IPv6 address in A record", "A", "2001:db8::1", "not a valid IPv4"),
				Entry("invalid IPv6 address in AAAA record", "AAAA", "invalid::ipv6::address:::::::", "not a valid IPv6"),
				Entry("IPv4 address in AAAA record", "AAAA", "192.168.1.1", "not a valid IPv6"),
			)

			DescribeTable("should return error for special values in non-A/AAAA records",
				func(recordType, specialValue, expectedError string) {
					spec := &v1beta1.DNSDisruptionSpec{
						Records: []v1beta1.DNSRecord{
							{
								Hostname: "example.com",
								Record: v1beta1.DNSRecordConfig{
									Type:  recordType,
									Value: specialValue,
								},
							},
						},
					}
					err := spec.Validate()
					Expect(err).To(MatchError(ContainSubstring(expectedError)))
				},
				Entry("NXDOMAIN in CNAME record", "CNAME", "NXDOMAIN", "special values not allowed for CNAME"),
				Entry("DROP in CNAME record", "CNAME", "DROP", "special values not allowed for CNAME"),
				Entry("NXDOMAIN in MX record", "MX", "NXDOMAIN", "special values not allowed for MX"),
				Entry("SERVFAIL in TXT record", "TXT", "SERVFAIL", "special values not allowed for TXT"),
				Entry("RANDOM in SRV record", "SRV", "RANDOM", "special values not allowed for SRV"),
			)

			DescribeTable("should return error for invalid record value formats",
				func(recordType, value, expectedError string) {
					spec := &v1beta1.DNSDisruptionSpec{
						Records: []v1beta1.DNSRecord{
							{
								Hostname: "example.com",
								Record: v1beta1.DNSRecordConfig{
									Type:  recordType,
									Value: value,
								},
							},
						},
					}
					err := spec.Validate()
					Expect(err).To(MatchError(ContainSubstring(expectedError)))
				},
				Entry("invalid MX record format - missing priority", "MX", "mail.example.com", "MX record must be in format"),
				Entry("invalid MX record format - non-numeric priority", "MX", "abc mail.example.com", "MX priority must be a number"),
				Entry("invalid SRV record format - too few fields", "SRV", "invalid format", "SRV record must be in format"),
				Entry("invalid SRV record format - non-numeric priority", "SRV", "abc 60 5060 server.example.com", "SRV priority must be a number"),
				Entry("invalid SRV record format - non-numeric weight", "SRV", "10 abc 5060 server.example.com", "SRV weight must be a number"),
				Entry("invalid SRV record format - non-numeric port", "SRV", "10 60 abc server.example.com", "SRV port must be a number"),
			)

			DescribeTable("should return error for invalid port and protocol",
				func(port int, protocol, expectedError string) {
					spec := &v1beta1.DNSDisruptionSpec{
						Records: []v1beta1.DNSRecord{
							{
								Hostname: "example.com",
								Record: v1beta1.DNSRecordConfig{
									Type:  "A",
									Value: "192.168.1.1",
								},
							},
						},
						Port:     port,
						Protocol: protocol,
					}
					err := spec.Validate()
					Expect(err).To(MatchError(ContainSubstring(expectedError)))
				},
				Entry("port too large", 70000, "", "must be between 1 and 65535"),
				Entry("port negative", -1, "", "must be between 1 and 65535"),
				Entry("invalid protocol", 0, "http", "must be one of"),
				Entry("invalid protocol with valid port", 5353, "ftp", "must be one of"),
			)

			It("should return multiple errors for multiple invalid records", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Records: []v1beta1.DNSRecord{
						{
							Hostname: "",
							Record: v1beta1.DNSRecordConfig{
								Type:  "A",
								Value: "",
							},
						},
						{
							Hostname: "example.com",
							Record: v1beta1.DNSRecordConfig{
								Type:  "INVALID",
								Value: "something",
							},
						},
					},
					Port:     70000,
					Protocol: "http",
				}
				err := spec.Validate()
				Expect(err).To(HaveOccurred())
				errStr := err.Error()
				Expect(errStr).To(ContainSubstring("hostname cannot be empty"))
				Expect(errStr).To(ContainSubstring("value cannot be empty"))
				Expect(errStr).To(ContainSubstring("must be one of"))
				Expect(errStr).To(ContainSubstring("must be between 1 and 65535"))
			})

			DescribeTable("should return error for mixed valid and invalid IPs",
				func(recordType, value, expectedError string) {
					spec := &v1beta1.DNSDisruptionSpec{
						Records: []v1beta1.DNSRecord{
							{
								Hostname: "example.com",
								Record: v1beta1.DNSRecordConfig{
									Type:  recordType,
									Value: value,
								},
							},
						},
					}
					err := spec.Validate()
					Expect(err).To(MatchError(ContainSubstring(expectedError)))
				},
				Entry("A record with one invalid IP in list", "A", "192.168.1.1,999.999.999.999", "not a valid IPv4"),
				Entry("AAAA record with one invalid IP in list", "AAAA", "2001:db8::1,invalid::address", "not a valid IPv6"),
			)

			It("should handle whitespace-only hostname as empty", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Records: []v1beta1.DNSRecord{
						{
							Hostname: "   ",
							Record: v1beta1.DNSRecordConfig{
								Type:  "A",
								Value: "192.168.1.1",
							},
						},
					},
				}
				err := spec.Validate()
				Expect(err).To(MatchError(ContainSubstring("hostname cannot be empty")))
			})
		})
	})

	Describe("GenerateArgs", func() {
		Context("New record-based format", func() {
			It("should generate correct arguments for A record", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Records: []v1beta1.DNSRecord{
						{
							Hostname: "example.com",
							Record: v1beta1.DNSRecordConfig{
								Type:  "A",
								Value: "192.168.1.1",
								TTL:   60,
							},
						},
					},
				}
				args := spec.GenerateArgs()
				Expect(args).To(ContainElement("dns-disruption"))
				Expect(args).To(ContainElement("--records"))
				Expect(args).To(ContainElement("example.com:A:192.168.1.1:60"))
				Expect(args).To(ContainElement("--port"))
				Expect(args).To(ContainElement("53"))
				Expect(args).To(ContainElement("--protocol"))
				Expect(args).To(ContainElement("both"))
			})

			It("should generate correct arguments for multiple records", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Records: []v1beta1.DNSRecord{
						{
							Hostname: "api.example.com",
							Record: v1beta1.DNSRecordConfig{
								Type:  "A",
								Value: "192.168.1.1,192.168.1.2",
								TTL:   30,
							},
						},
						{
							Hostname: "alias.example.com",
							Record: v1beta1.DNSRecordConfig{
								Type:  "CNAME",
								Value: "target.example.com",
								TTL:   300,
							},
						},
					},
				}
				args := spec.GenerateArgs()
				Expect(args).To(ContainElement("--records"))
				// Records are joined with semicolon
				recordsArg := ""
				for i, arg := range args {
					if arg == "--records" && i+1 < len(args) {
						recordsArg = args[i+1]
						break
					}
				}
				Expect(recordsArg).To(ContainSubstring("api.example.com:A:192.168.1.1,192.168.1.2:30"))
				Expect(recordsArg).To(ContainSubstring("alias.example.com:CNAME:target.example.com:300"))
			})

			It("should use custom port and protocol", func() {
				spec := &v1beta1.DNSDisruptionSpec{
					Records: []v1beta1.DNSRecord{
						{
							Hostname: "example.com",
							Record: v1beta1.DNSRecordConfig{
								Type:  "A",
								Value: "192.168.1.1",
							},
						},
					},
					Port:     5353,
					Protocol: "udp",
				}
				args := spec.GenerateArgs()
				Expect(args).To(ContainElement("5353"))
				Expect(args).To(ContainElement("udp"))
			})
		})
	})
})
