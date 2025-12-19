// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package network_test

import (
	"os"

	"path/filepath"

	"github.com/DataDog/chaos-controller/network"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DNS Path Configuration", func() {
	Describe("readResolvConfFile", func() {
		var tempDir string

		BeforeEach(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "dns-test-*")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			err := os.RemoveAll(tempDir)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when path exists and is valid", func() {
			It("should return the DNS configuration", func() {
				// Arrange
				// Create a valid resolv.conf file
				resolvConfPath := filepath.Join(tempDir, "resolv.conf")
				content := "nameserver 8.8.8.8\nnameserver 8.8.4.4\n"
				err := os.WriteFile(resolvConfPath, []byte(content), 0644)
				Expect(err).NotTo(HaveOccurred())

				// Act
				config, err := network.ReadResolvConfFileForTest(resolvConfPath)

				// Assert
				Expect(err).NotTo(HaveOccurred())
				Expect(config).NotTo(BeNil())
				Expect(config.Servers).To(ConsistOf("8.8.8.8", "8.8.4.4"))
			})
		})

		Context("when path does not exist", func() {
			It("should return an error", func() {
				// Act
				_, err := network.ReadResolvConfFileForTest("/nonexistent/path")

				// Assert
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("/nonexistent/path"))
			})
		})
	})
})
