// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package network_test

import (
	"go.uber.org/zap"

	"github.com/DataDog/chaos-controller/network"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DNSResponder", func() {
	var (
		responder network.DNSResponder
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
				Records: []network.DNSRecordEntry{
					{
						Hostname:   "example.com",
						RecordType: "A",
						Value:      "NXDOMAIN",
						TTL:        300,
					},
				},
				UDPPort:  port,
				Protocol: "udp",
				Logger:   logger,
			}

			responder = network.NewDNSResponder(config)
			Expect(responder).NotTo(BeNil())
		})
	})
})
