// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package datadog

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDatadog(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CloudService Datadog Suite")
}

var _ = BeforeSuite(func() {
})

var _ = AfterSuite(func() {
})

var _ = Describe("Datadog Parsing", func() {
	Context("Golden path", func() {
		ipRangeFile := "{\"version\":51,\"modified\":\"2022-09-26-19-13-00\",\"api\":{\"prefixes_ipv4\":[\"3.233.144.0/20\"],\"prefixes_ipv6\":[\"2600:1f18:24e6:b900::/56\"]},\"fake-product\":{\"prefixes_ipv4\":[\"1.2.3.0/20\"]},\"fake-product-2\":{\"prefixes_ipv6\":[\"2600:1f18:24e6:b900::/56\"]}}"
		datadogManager := New()

		info, err := datadogManager.ConvertToGenericIPRanges([]byte(ipRangeFile))

		It("should parse the ip range file", func() {
			By("Ensuring that no error was thrown")
			Expect(err).To(BeNil())

			By("Ensuring that the right version string was parsed")
			Expect(info.Version).To(Equal("51"))

			By("Ensuring that we have the right info")
			Expect(len(info.IPRanges["api"])).To(Equal(1))

			_, ok := info.IPRanges["fake-product-2"]
			Expect(ok).To(BeFalse())
		})

		It("should have a new version", func() {
			newVersion, err := datadogManager.IsNewVersion([]byte(ipRangeFile), "50")

			By("Ensuring that no error was thrown")
			Expect(err).To(BeNil())

			By("Ensuring that the right version string was parsed")
			Expect(newVersion).To(BeTrue())
		})

		It("should not have a new version", func() {
			newVersion, err := datadogManager.IsNewVersion([]byte(ipRangeFile), "51")

			By("Ensuring that no error was thrown")
			Expect(err).To(BeNil())

			By("Ensuring that the right version string was parsed")
			Expect(newVersion).To(BeFalse())
		})
	})

	Context("Verify Datadog handle of errors", func() {
		It("should error on parsing of Datadog IP Range file", func() {
			ipRangeFile := "{\"version\":\"51\",\"modified\":\"2022-09-26-19-13-00\",\"api\":{\"prefixes_ipv4\":[\"3.233.144.0/20\"],\"prefixes_ipv6\":[\"2600:1f18:24e6:b900::/56\"]},\"fake-product\":{\"prefixes_ipv4\":[\"1.2.3.0/20\"]},\"fake-product-2\":{\"prefixes_ipv6\":[\"2600:1f18:24e6:b900::/56\"]}}"
			datadogManager := New()

			info, err := datadogManager.ConvertToGenericIPRanges([]byte(ipRangeFile))

			By("Ensuring that an error was thrown")
			Expect(err).ToNot(BeNil())

			By("Ensuring the returned converted IP Ranges is nil")
			Expect(info).To(BeNil())
		})

		It("Should throw an error on empty ip ranges file", func() {
			ipRangeFile := ""
			datadogManager := New()

			_, errConvert := datadogManager.ConvertToGenericIPRanges([]byte(ipRangeFile))
			_, errIsNewVersion := datadogManager.IsNewVersion([]byte(ipRangeFile), "20")

			By("Ensuring that an error was thrown on ConvertToGenericIPRanges")
			Expect(errConvert).ToNot(BeNil())

			By("Ensuring that an error was thrown on IsNewVersion")
			Expect(errIsNewVersion).ToNot(BeNil())
		})

		It("Should throw an error on empty ip ranges file", func() {
			datadogManager := New()

			_, errConvert := datadogManager.ConvertToGenericIPRanges(make([]byte, 0))
			_, errIsNewVersion := datadogManager.IsNewVersion(make([]byte, 0), "20")

			By("Ensuring that an error was thrown on ConvertToGenericIPRanges")
			Expect(errConvert).ToNot(BeNil())

			By("Ensuring that an error was thrown on IsNewVersion")
			Expect(errIsNewVersion).ToNot(BeNil())
		})

		It("Should throw an error on nil ip ranges file", func() {
			datadogManager := New()

			_, errConvert := datadogManager.ConvertToGenericIPRanges(nil)
			_, errIsNewVersion := datadogManager.IsNewVersion(nil, "20")

			By("Ensuring that an error was thrown on ConvertToGenericIPRanges")
			Expect(errConvert).ToNot(BeNil())

			By("Ensuring that an error was thrown on IsNewVersion")
			Expect(errIsNewVersion).ToNot(BeNil())
		})
	})
})
