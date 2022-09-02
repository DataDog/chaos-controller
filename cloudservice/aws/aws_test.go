package aws

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Label Selector Validation", func() {
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
			Expect(len(info.IPRanges["AMAZON"])).To(Equal(2))
			Expect(len(info.IPRanges["S3"])).To(Equal(1))
		})
	})
})
