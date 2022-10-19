// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package gcp

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestGCP(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CloudService GCP Suite")
}

var _ = BeforeSuite(func() {
})

var _ = AfterSuite(func() {
})

var _ = Describe("GCP Parsing", func() {
	Context("Parse GCP IP Range file", func() {
		ipRangeFile := "{\"syncToken\":\"1000000000\",\"createDate\":\"2022-09-01-22-03-06\",\"prefixes\":[{\"ipv4Prefix\": \"34.80.0.0/15\",\"service\": \"Google Cloud\",\"scope\": \"asia-east1\"},{\"ipv4Prefix\": \"34.80.0.0/15\",\"service\": \"Google Cloud\",\"scope\": \"asia-east1\"},{\"ipv4Prefix\": \"34.80.0.0/15\",\"service\": \"Google Cloud\",\"scope\": \"asia-east1\"},{\"ipv4Prefix\": \"34.80.0.0/15\",\"service\": \"Google Cloud\",\"scope\": \"asia-east1\"}]}"
		gcpManager := New()

		info, err := gcpManager.ConvertToGenericIPRanges([]byte(ipRangeFile))

		It("should parse the ip range file", func() {
			By("Ensuring that no error was thrown")
			Expect(err).To(BeNil())

			By("Ensuring that the right version string was parsed")
			Expect(info.Version).To(Equal("1000000000"))

			By("Ensuring that we have the right info")
			Expect(len(info.IPRanges["Google Cloud"])).To(Equal(4))
		})
	})

	Context("Verify GCP New version of the file", func() {
		ipRangeFile := "{\"syncToken\":\"1000000000\",\"createDate\":\"2022-09-01-22-03-06\",\"prefixes\":[{\"ipv4Prefix\": \"34.80.0.0/15\",\"service\": \"Google Cloud\",\"scope\": \"asia-east1\"},{\"ipv4Prefix\": \"34.80.0.0/15\",\"service\": \"Google Cloud\",\"scope\": \"asia-east1\"},{\"ipv4Prefix\": \"34.80.0.0/15\",\"service\": \"Google Cloud\",\"scope\": \"asia-east1\"},{\"ipv4Prefix\": \"34.80.0.0/15\",\"service\": \"Google Cloud\",\"scope\": \"asia-east1\"}]}"
		awsManager := New()

		isNewVersion := awsManager.IsNewVersion([]byte(ipRangeFile), "20")

		It("Should indicate is a new version", func() {
			By("Ensuring that the version is new")
			Expect(isNewVersion).To(Equal(true))
		})
	})

})
