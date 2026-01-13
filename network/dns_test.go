// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

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

		Context("Success cases", func() {
			It("should return the DNS configuration with single nameserver", func() {
				resolvConfPath := filepath.Join(tempDir, "resolv.conf")
				content := "nameserver 8.8.8.8\n"
				err := os.WriteFile(resolvConfPath, []byte(content), 0644)
				Expect(err).NotTo(HaveOccurred())

				config, err := network.ReadResolvConfFileForTest(resolvConfPath)

				Expect(err).NotTo(HaveOccurred())
				Expect(config).NotTo(BeNil())
				Expect(config.Servers).To(ConsistOf("8.8.8.8"))
			})

			It("should return the DNS configuration with multiple nameservers", func() {
				resolvConfPath := filepath.Join(tempDir, "resolv.conf")
				content := "nameserver 8.8.8.8\nnameserver 8.8.4.4\nnameserver 1.1.1.1\n"
				err := os.WriteFile(resolvConfPath, []byte(content), 0644)
				Expect(err).NotTo(HaveOccurred())

				config, err := network.ReadResolvConfFileForTest(resolvConfPath)

				Expect(err).NotTo(HaveOccurred())
				Expect(config).NotTo(BeNil())
				Expect(config.Servers).To(ConsistOf("8.8.8.8", "8.8.4.4", "1.1.1.1"))
			})

			It("should parse IPv6 nameservers", func() {
				resolvConfPath := filepath.Join(tempDir, "resolv.conf")
				content := "nameserver 2001:4860:4860::8888\nnameserver 2001:4860:4860::8844\n"
				err := os.WriteFile(resolvConfPath, []byte(content), 0644)
				Expect(err).NotTo(HaveOccurred())

				config, err := network.ReadResolvConfFileForTest(resolvConfPath)

				Expect(err).NotTo(HaveOccurred())
				Expect(config).NotTo(BeNil())
				Expect(config.Servers).To(ConsistOf("2001:4860:4860::8888", "2001:4860:4860::8844"))
			})

			It("should parse mixed IPv4 and IPv6 nameservers", func() {
				resolvConfPath := filepath.Join(tempDir, "resolv.conf")
				content := "nameserver 8.8.8.8\nnameserver 2001:4860:4860::8888\nnameserver 1.1.1.1\n"
				err := os.WriteFile(resolvConfPath, []byte(content), 0644)
				Expect(err).NotTo(HaveOccurred())

				config, err := network.ReadResolvConfFileForTest(resolvConfPath)

				Expect(err).NotTo(HaveOccurred())
				Expect(config).NotTo(BeNil())
				Expect(config.Servers).To(ConsistOf("8.8.8.8", "2001:4860:4860::8888", "1.1.1.1"))
			})

			It("should parse resolv.conf with search domains", func() {
				resolvConfPath := filepath.Join(tempDir, "resolv.conf")
				content := "nameserver 8.8.8.8\nsearch example.com internal.example.com\n"
				err := os.WriteFile(resolvConfPath, []byte(content), 0644)
				Expect(err).NotTo(HaveOccurred())

				config, err := network.ReadResolvConfFileForTest(resolvConfPath)

				Expect(err).NotTo(HaveOccurred())
				Expect(config).NotTo(BeNil())
				Expect(config.Servers).To(ConsistOf("8.8.8.8"))
				Expect(config.Search).To(ConsistOf("example.com", "internal.example.com"))
			})

			It("should parse resolv.conf with options", func() {
				resolvConfPath := filepath.Join(tempDir, "resolv.conf")
				content := "nameserver 8.8.8.8\noptions ndots:5 timeout:2 attempts:3\n"
				err := os.WriteFile(resolvConfPath, []byte(content), 0644)
				Expect(err).NotTo(HaveOccurred())

				config, err := network.ReadResolvConfFileForTest(resolvConfPath)

				Expect(err).NotTo(HaveOccurred())
				Expect(config).NotTo(BeNil())
				Expect(config.Servers).To(ConsistOf("8.8.8.8"))
				Expect(config.Ndots).To(Equal(5))
				Expect(config.Timeout).To(Equal(2))
				Expect(config.Attempts).To(Equal(3))
			})

			It("should parse full resolv.conf with all directives", func() {
				resolvConfPath := filepath.Join(tempDir, "resolv.conf")
				content := `# This is a comment
nameserver 8.8.8.8
nameserver 8.8.4.4
search example.com
options ndots:5 timeout:2 attempts:3
`
				err := os.WriteFile(resolvConfPath, []byte(content), 0644)
				Expect(err).NotTo(HaveOccurred())

				config, err := network.ReadResolvConfFileForTest(resolvConfPath)

				Expect(err).NotTo(HaveOccurred())
				Expect(config).NotTo(BeNil())
				Expect(config.Servers).To(ConsistOf("8.8.8.8", "8.8.4.4"))
				Expect(config.Search).To(ConsistOf("example.com"))
				Expect(config.Ndots).To(Equal(5))
				Expect(config.Timeout).To(Equal(2))
				Expect(config.Attempts).To(Equal(3))
			})

			It("should handle empty lines and comments", func() {
				resolvConfPath := filepath.Join(tempDir, "resolv.conf")
				content := `# DNS Configuration
nameserver 8.8.8.8

# Backup DNS
nameserver 8.8.4.4

# Search domain
search example.com
`
				err := os.WriteFile(resolvConfPath, []byte(content), 0644)
				Expect(err).NotTo(HaveOccurred())

				config, err := network.ReadResolvConfFileForTest(resolvConfPath)

				Expect(err).NotTo(HaveOccurred())
				Expect(config).NotTo(BeNil())
				Expect(config.Servers).To(ConsistOf("8.8.8.8", "8.8.4.4"))
			})

			It("should ignore whitespace around entries", func() {
				resolvConfPath := filepath.Join(tempDir, "resolv.conf")
				content := "  nameserver   8.8.8.8  \n  nameserver   8.8.4.4  \n"
				err := os.WriteFile(resolvConfPath, []byte(content), 0644)
				Expect(err).NotTo(HaveOccurred())

				config, err := network.ReadResolvConfFileForTest(resolvConfPath)

				Expect(err).NotTo(HaveOccurred())
				Expect(config).NotTo(BeNil())
				Expect(config.Servers).To(ConsistOf("8.8.8.8", "8.8.4.4"))
			})
		})

		Context("Error cases", func() {
			It("should return an error when path does not exist", func() {
				_, err := network.ReadResolvConfFileForTest("/nonexistent/path")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("/nonexistent/path"))
			})

			It("should return an error when file is empty", func() {
				resolvConfPath := filepath.Join(tempDir, "resolv.conf")
				err := os.WriteFile(resolvConfPath, []byte(""), 0644)
				Expect(err).NotTo(HaveOccurred())

				config, err := network.ReadResolvConfFileForTest(resolvConfPath)

				Expect(err).NotTo(HaveOccurred())
				Expect(config).NotTo(BeNil())
				Expect(config.Servers).To(BeEmpty())
			})

			It("should return an error when file contains only comments", func() {
				resolvConfPath := filepath.Join(tempDir, "resolv.conf")
				content := "# Just a comment\n# Another comment\n"
				err := os.WriteFile(resolvConfPath, []byte(content), 0644)
				Expect(err).NotTo(HaveOccurred())

				config, err := network.ReadResolvConfFileForTest(resolvConfPath)

				Expect(err).NotTo(HaveOccurred())
				Expect(config).NotTo(BeNil())
				Expect(config.Servers).To(BeEmpty())
			})

			It("should handle file with no nameserver entries", func() {
				resolvConfPath := filepath.Join(tempDir, "resolv.conf")
				content := "search example.com\noptions ndots:5\n"
				err := os.WriteFile(resolvConfPath, []byte(content), 0644)
				Expect(err).NotTo(HaveOccurred())

				config, err := network.ReadResolvConfFileForTest(resolvConfPath)

				Expect(err).NotTo(HaveOccurred())
				Expect(config).NotTo(BeNil())
				Expect(config.Servers).To(BeEmpty())
				Expect(config.Search).To(ConsistOf("example.com"))
			})

			It("should handle when path is a directory", func() {
				dirPath := filepath.Join(tempDir, "somedir")
				err := os.Mkdir(dirPath, 0755)
				Expect(err).NotTo(HaveOccurred())

				config, err := network.ReadResolvConfFileForTest(dirPath)

				Expect(err).NotTo(HaveOccurred())
				Expect(config).NotTo(BeNil())
				Expect(config.Servers).To(BeEmpty())
			})

			It("should handle malformed nameserver line gracefully", func() {
				resolvConfPath := filepath.Join(tempDir, "resolv.conf")
				content := "nameserver\nnameserver 8.8.8.8\n"
				err := os.WriteFile(resolvConfPath, []byte(content), 0644)
				Expect(err).NotTo(HaveOccurred())

				config, err := network.ReadResolvConfFileForTest(resolvConfPath)

				Expect(err).NotTo(HaveOccurred())
				Expect(config).NotTo(BeNil())
				Expect(config.Servers).To(ConsistOf("8.8.8.8"))
			})
		})

		Context("Edge cases", func() {
			It("should handle resolv.conf with Windows line endings", func() {
				resolvConfPath := filepath.Join(tempDir, "resolv.conf")
				content := "nameserver 8.8.8.8\r\nnameserver 8.8.4.4\r\n"
				err := os.WriteFile(resolvConfPath, []byte(content), 0644)
				Expect(err).NotTo(HaveOccurred())

				config, err := network.ReadResolvConfFileForTest(resolvConfPath)

				Expect(err).NotTo(HaveOccurred())
				Expect(config).NotTo(BeNil())
				Expect(config.Servers).To(ConsistOf("8.8.8.8", "8.8.4.4"))
			})

			It("should handle maximum number of nameservers", func() {
				resolvConfPath := filepath.Join(tempDir, "resolv.conf")
				content := ""
				expectedServers := []string{}
				for i := 1; i <= 10; i++ {
					server := "8.8.8." + string(rune('0'+i))
					content += "nameserver " + server + "\n"
					expectedServers = append(expectedServers, server)
				}
				err := os.WriteFile(resolvConfPath, []byte(content), 0644)
				Expect(err).NotTo(HaveOccurred())

				config, err := network.ReadResolvConfFileForTest(resolvConfPath)

				Expect(err).NotTo(HaveOccurred())
				Expect(config).NotTo(BeNil())
				Expect(len(config.Servers)).To(BeNumerically("<=", 10))
			})

			It("should handle resolv.conf with tabs as separators", func() {
				resolvConfPath := filepath.Join(tempDir, "resolv.conf")
				content := "nameserver\t8.8.8.8\nnameserver\t8.8.4.4\n"
				err := os.WriteFile(resolvConfPath, []byte(content), 0644)
				Expect(err).NotTo(HaveOccurred())

				config, err := network.ReadResolvConfFileForTest(resolvConfPath)

				Expect(err).NotTo(HaveOccurred())
				Expect(config).NotTo(BeNil())
				Expect(config.Servers).To(ConsistOf("8.8.8.8", "8.8.4.4"))
			})

			It("should preserve order of nameservers", func() {
				resolvConfPath := filepath.Join(tempDir, "resolv.conf")
				content := "nameserver 1.1.1.1\nnameserver 8.8.8.8\nnameserver 8.8.4.4\n"
				err := os.WriteFile(resolvConfPath, []byte(content), 0644)
				Expect(err).NotTo(HaveOccurred())

				config, err := network.ReadResolvConfFileForTest(resolvConfPath)

				Expect(err).NotTo(HaveOccurred())
				Expect(config).NotTo(BeNil())
				Expect(config.Servers).To(HaveLen(3))
				Expect(config.Servers[0]).To(Equal("1.1.1.1"))
				Expect(config.Servers[1]).To(Equal("8.8.8.8"))
				Expect(config.Servers[2]).To(Equal("8.8.4.4"))
			})
		})
	})
})
