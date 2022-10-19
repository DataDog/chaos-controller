// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package aws

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAWS(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CloudService AWS Suite")
}

var _ = BeforeSuite(func() {
})

var _ = AfterSuite(func() {
})

var _ = Describe("AWS Parsing", func() {
	Context("Parse AWS IP Range file", func() {
		ipRangeFile := "{\"syncToken\":\"1000000000\",\"createDate\":\"2022-09-01-22-03-06\",\"prefixes\":[{\"ip_prefix\":\"3.2.34.0/26\",\"region\":\"af-south-1\",\"service\":\"AMAZON\",\"network_border_group\":\"af-south-1\"},{\"ip_prefix\":\"3.5.140.0/22\",\"region\":\"ap-northeast-2\",\"service\":\"AMAZON\",\"network_border_group\":\"ap-northeast-2\"},{\"ip_prefix\":\"13.34.37.64/27\",\"region\":\"ap-southeast-4\",\"service\":\"S3\",\"network_border_group\":\"ap-southeast-4\"}],\"ipv6_prefixes\":[{\"ipv6_prefix\":\"2600:1ff2:4000::/40\",\"region\":\"us-west-2\",\"service\":\"AMAZON\",\"network_border_group\":\"us-west-2\"}]}"
		awsManager := New()

		info, err := awsManager.ConvertToGenericIPRanges([]byte(ipRangeFile))

		It("should parse the ip range file", func() {
			By("Ensuring that no error was thrown")
			Expect(err).To(BeNil())

			By("Ensuring that the right version string was parsed")
			Expect(info.Version).To(Equal("1000000000"))

			By("Ensuring that we have the right info")
			Expect(len(info.IPRanges["AMAZON"])).To(Equal(0))
			Expect(len(info.IPRanges["S3"])).To(Equal(1))
		})

		It("should error on parsing", func() {
			ipRangeFile = "{\"syncToken\":\"1000000000\",\"createDate\":\"2022-09-01-22-03-06\",\"prefixes\":{\"ip_prefix\":\"3.2.34.0/26\",\"region\":\"af-south-1\",\"service\":\"AMAZON\",\"network_border_group\":\"af-south-1\"}}"
			_, err := awsManager.ConvertToGenericIPRanges([]byte(ipRangeFile))

			By("Ensuring that error was thrown")
			Expect(err).ToNot(BeNil())
		})
	})

	Context("Verify AWS New version of the file", func() {
		ipRangeFile := "{\"syncToken\":\"1000000000\",\"createDate\":\"2022-09-01-22-03-06\",\"prefixes\":[{\"ip_prefix\":\"3.2.34.0/26\",\"region\":\"af-south-1\",\"service\":\"AMAZON\",\"network_border_group\":\"af-south-1\"},{\"ip_prefix\":\"3.5.140.0/22\",\"region\":\"ap-northeast-2\",\"service\":\"AMAZON\",\"network_border_group\":\"ap-northeast-2\"},{\"ip_prefix\":\"13.34.37.64/27\",\"region\":\"ap-southeast-4\",\"service\":\"S3\",\"network_border_group\":\"ap-southeast-4\"}],\"ipv6_prefixes\":[{\"ipv6_prefix\":\"2600:1ff2:4000::/40\",\"region\":\"us-west-2\",\"service\":\"AMAZON\",\"network_border_group\":\"us-west-2\"}]}"
		awsManager := New()

		isNewVersion := awsManager.IsNewVersion([]byte(ipRangeFile), "20")

		It("Should indicate is a new version", func() {
			By("Ensuring that the version is new")
			Expect(isNewVersion).To(Equal(true))
		})
	})
})
