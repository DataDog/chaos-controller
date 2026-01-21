// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package network_test

import (
	"fmt"
	"time"

	"github.com/DataDog/chaos-controller/network"
	"github.com/miekg/dns"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

var _ = Describe("DNSResponder", func() {
	var (
		responder *network.DNSResponder
		logger    *zap.SugaredLogger
		port      int
	)

	BeforeEach(func() {
		// Use a high port number to avoid conflicts
		port = 15353

		// Create a test logger
		zapLogger, err := zap.NewDevelopment()
		Expect(err).NotTo(HaveOccurred())
		logger = zapLogger.Sugar()
	})

	AfterEach(func() {
		if responder != nil {
			err := responder.Stop()
			Expect(err).NotTo(HaveOccurred())
		}
	})

	Describe("NewDNSResponder", func() {
		It("should create a new DNS responder with valid config", func() {
			config := network.DNSResponderConfig{
				TargetDomains: []string{"example.com"},
				FailureMode:   "nxdomain",
				Port:          port,
				Protocol:      "udp",
				Logger:        logger,
			}

			responder = network.NewDNSResponder(config)
			Expect(responder).NotTo(BeNil())
		})
	})

	Describe("Start and Stop", func() {
		It("should start and stop UDP server successfully", func() {
			config := network.DNSResponderConfig{
				TargetDomains: []string{"example.com"},
				FailureMode:   "nxdomain",
				Port:          port,
				Protocol:      "udp",
				Logger:        logger,
			}

			responder = network.NewDNSResponder(config)
			err := responder.Start()
			Expect(err).NotTo(HaveOccurred())

			// Give the server time to start
			time.Sleep(100 * time.Millisecond)

			err = responder.Stop()
			Expect(err).NotTo(HaveOccurred())
			responder = nil
		})

		It("should start and stop TCP server successfully", func() {
			config := network.DNSResponderConfig{
				TargetDomains: []string{"example.com"},
				FailureMode:   "nxdomain",
				Port:          port,
				Protocol:      "tcp",
				Logger:        logger,
			}

			responder = network.NewDNSResponder(config)
			err := responder.Start()
			Expect(err).NotTo(HaveOccurred())

			// Give the server time to start
			time.Sleep(100 * time.Millisecond)

			err = responder.Stop()
			Expect(err).NotTo(HaveOccurred())
			responder = nil
		})

		It("should start and stop both UDP and TCP servers successfully", func() {
			config := network.DNSResponderConfig{
				TargetDomains: []string{"example.com"},
				FailureMode:   "nxdomain",
				Port:          port,
				Protocol:      "both",
				Logger:        logger,
			}

			responder = network.NewDNSResponder(config)
			err := responder.Start()
			Expect(err).NotTo(HaveOccurred())

			// Give the server time to start
			time.Sleep(100 * time.Millisecond)

			err = responder.Stop()
			Expect(err).NotTo(HaveOccurred())
			responder = nil
		})
	})

	Describe("DNS Query Handling", func() {
		Context("NXDOMAIN failure mode", func() {
			BeforeEach(func() {
				config := network.DNSResponderConfig{
					TargetDomains: []string{"example.com"},
					FailureMode:   "nxdomain",
					Port:          port,
					Protocol:      "udp",
					Logger:        logger,
				}

				responder = network.NewDNSResponder(config)
				err := responder.Start()
				Expect(err).NotTo(HaveOccurred())

				// Give the server time to start
				time.Sleep(100 * time.Millisecond)
			})

			It("should return NXDOMAIN for target domain", func() {
				client := new(dns.Client)
				msg := new(dns.Msg)
				msg.SetQuestion("example.com.", dns.TypeA)

				response, _, err := client.Exchange(msg, fmt.Sprintf("127.0.0.1:%d", port))
				Expect(err).NotTo(HaveOccurred())
				Expect(response).NotTo(BeNil())
				Expect(response.Rcode).To(Equal(dns.RcodeNameError))
			})

			It("should return NXDOMAIN for subdomain of target", func() {
				client := new(dns.Client)
				msg := new(dns.Msg)
				msg.SetQuestion("www.example.com.", dns.TypeA)

				response, _, err := client.Exchange(msg, fmt.Sprintf("127.0.0.1:%d", port))
				Expect(err).NotTo(HaveOccurred())
				Expect(response).NotTo(BeNil())
				Expect(response.Rcode).To(Equal(dns.RcodeNameError))
			})

			It("should not respond for non-target domain", func() {
				client := new(dns.Client)
				client.Timeout = 500 * time.Millisecond
				msg := new(dns.Msg)
				msg.SetQuestion("notarget.com.", dns.TypeA)

				response, _, err := client.Exchange(msg, fmt.Sprintf("127.0.0.1:%d", port))
				// Should timeout since responder ignores non-target domains
				Expect(err).To(HaveOccurred())
				Expect(response).To(BeNil())
			})
		})

		Context("SERVFAIL failure mode", func() {
			BeforeEach(func() {
				config := network.DNSResponderConfig{
					TargetDomains: []string{"test.com"},
					FailureMode:   "servfail",
					Port:          port,
					Protocol:      "udp",
					Logger:        logger,
				}

				responder = network.NewDNSResponder(config)
				err := responder.Start()
				Expect(err).NotTo(HaveOccurred())

				// Give the server time to start
				time.Sleep(100 * time.Millisecond)
			})

			It("should return SERVFAIL for target domain", func() {
				client := new(dns.Client)
				msg := new(dns.Msg)
				msg.SetQuestion("test.com.", dns.TypeA)

				response, _, err := client.Exchange(msg, fmt.Sprintf("127.0.0.1:%d", port))
				Expect(err).NotTo(HaveOccurred())
				Expect(response).NotTo(BeNil())
				Expect(response.Rcode).To(Equal(dns.RcodeServerFailure))
			})
		})

		Context("DROP failure mode", func() {
			BeforeEach(func() {
				config := network.DNSResponderConfig{
					TargetDomains: []string{"drop.com"},
					FailureMode:   "drop",
					Port:          port,
					Protocol:      "udp",
					Logger:        logger,
				}

				responder = network.NewDNSResponder(config)
				err := responder.Start()
				Expect(err).NotTo(HaveOccurred())

				// Give the server time to start
				time.Sleep(100 * time.Millisecond)
			})

			It("should not respond (timeout) for target domain", func() {
				client := new(dns.Client)
				client.Timeout = 500 * time.Millisecond
				msg := new(dns.Msg)
				msg.SetQuestion("drop.com.", dns.TypeA)

				response, _, err := client.Exchange(msg, fmt.Sprintf("127.0.0.1:%d", port))
				// Should timeout since responder drops the query
				Expect(err).To(HaveOccurred())
				Expect(response).To(BeNil())
			})
		})

		Context("RANDOM-IP failure mode", func() {
			BeforeEach(func() {
				config := network.DNSResponderConfig{
					TargetDomains: []string{"random.com"},
					FailureMode:   "random-ip",
					Port:          port,
					Protocol:      "udp",
					Logger:        logger,
				}

				responder = network.NewDNSResponder(config)
				err := responder.Start()
				Expect(err).NotTo(HaveOccurred())

				// Give the server time to start
				time.Sleep(100 * time.Millisecond)
			})

			It("should return a random IP for target domain", func() {
				client := new(dns.Client)
				msg := new(dns.Msg)
				msg.SetQuestion("random.com.", dns.TypeA)

				response, _, err := client.Exchange(msg, fmt.Sprintf("127.0.0.1:%d", port))
				Expect(err).NotTo(HaveOccurred())
				Expect(response).NotTo(BeNil())
				Expect(response.Rcode).To(Equal(dns.RcodeSuccess))
				Expect(response.Answer).To(HaveLen(1))

				// Check that we got an A record
				aRecord, ok := response.Answer[0].(*dns.A)
				Expect(ok).To(BeTrue())
				Expect(aRecord).NotTo(BeNil())

				// Check that IP is in the 192.0.2.0/24 range (TEST-NET-1)
				ip := aRecord.A.String()
				Expect(ip).To(MatchRegexp(`^192\.0\.2\.\d+$`))
			})
		})

		Context("Multiple target domains", func() {
			BeforeEach(func() {
				config := network.DNSResponderConfig{
					TargetDomains: []string{"example.com", "test.com", "api.example.com"},
					FailureMode:   "nxdomain",
					Port:          port,
					Protocol:      "udp",
					Logger:        logger,
				}

				responder = network.NewDNSResponder(config)
				err := responder.Start()
				Expect(err).NotTo(HaveOccurred())

				// Give the server time to start
				time.Sleep(100 * time.Millisecond)
			})

			It("should disrupt all target domains", func() {
				client := new(dns.Client)

				// Test example.com
				msg1 := new(dns.Msg)
				msg1.SetQuestion("example.com.", dns.TypeA)
				response1, _, err := client.Exchange(msg1, fmt.Sprintf("127.0.0.1:%d", port))
				Expect(err).NotTo(HaveOccurred())
				Expect(response1.Rcode).To(Equal(dns.RcodeNameError))

				// Test test.com
				msg2 := new(dns.Msg)
				msg2.SetQuestion("test.com.", dns.TypeA)
				response2, _, err := client.Exchange(msg2, fmt.Sprintf("127.0.0.1:%d", port))
				Expect(err).NotTo(HaveOccurred())
				Expect(response2.Rcode).To(Equal(dns.RcodeNameError))

				// Test api.example.com
				msg3 := new(dns.Msg)
				msg3.SetQuestion("api.example.com.", dns.TypeA)
				response3, _, err := client.Exchange(msg3, fmt.Sprintf("127.0.0.1:%d", port))
				Expect(err).NotTo(HaveOccurred())
				Expect(response3.Rcode).To(Equal(dns.RcodeNameError))
			})

			It("should not respond for domains not in target list", func() {
				client := new(dns.Client)
				client.Timeout = 500 * time.Millisecond
				msg := new(dns.Msg)
				msg.SetQuestion("nottargeted.com.", dns.TypeA)

				response, _, err := client.Exchange(msg, fmt.Sprintf("127.0.0.1:%d", port))
				Expect(err).To(HaveOccurred())
				Expect(response).To(BeNil())
			})
		})

		Context("TCP protocol", func() {
			BeforeEach(func() {
				config := network.DNSResponderConfig{
					TargetDomains: []string{"tcp.example.com"},
					FailureMode:   "nxdomain",
					Port:          port,
					Protocol:      "tcp",
					Logger:        logger,
				}

				responder = network.NewDNSResponder(config)
				err := responder.Start()
				Expect(err).NotTo(HaveOccurred())

				// Give the server time to start
				time.Sleep(100 * time.Millisecond)
			})

			It("should handle TCP DNS queries", func() {
				client := new(dns.Client)
				client.Net = "tcp"
				msg := new(dns.Msg)
				msg.SetQuestion("tcp.example.com.", dns.TypeA)

				response, _, err := client.Exchange(msg, fmt.Sprintf("127.0.0.1:%d", port))
				Expect(err).NotTo(HaveOccurred())
				Expect(response).NotTo(BeNil())
				Expect(response.Rcode).To(Equal(dns.RcodeNameError))
			})
		})
	})
})
