// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package datadog

import (
	"testing"

	. "github.com/onsi/ginkgo"
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
	Context("Parse Datadog IP Range file", func() {
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
	})

	Context("Error on parse Datadog IP Range file", func() {
		ipRangeFile := "{\"version\":\"51\",\"modified\":\"2022-09-26-19-13-00\",\"api\":{\"prefixes_ipv4\":[\"3.233.144.0/20\"],\"prefixes_ipv6\":[\"2600:1f18:24e6:b900::/56\"]},\"fake-product\":{\"prefixes_ipv4\":[\"1.2.3.0/20\"]},\"fake-product-2\":{\"prefixes_ipv6\":[\"2600:1f18:24e6:b900::/56\"]}}"
		datadogManager := New()

		info, err := datadogManager.ConvertToGenericIPRanges([]byte(ipRangeFile))

		It("should not parse the ip range file", func() {
			By("Ensuring that an error was thrown")
			Expect(err).ToNot(BeNil())

			By("Ensuring the returned converted IP Ranges is nil")
			Expect(info).To(BeNil())
		})
	})
})
