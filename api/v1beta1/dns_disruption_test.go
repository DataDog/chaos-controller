// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package v1beta1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DNSDisruptionSpec", func() {
	When("'Validate' method is called", func() {
		Describe("test regex validation cases", func() {
			It("errors only on invalid regex", func() {
				disruptionSpec := DNSDisruptionSpec{
					{
						Hostname: "hostname.tld",
						Record: DNSRecord{
							Type:  "",
							Value: "",
						},
					},
				}

				err := disruptionSpec.Validate()
				Expect(err).ToNot(HaveOccurred())

				disruptionSpec[0].Hostname = "*.hostname.tld"

				err = disruptionSpec.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("not a valid regular expression"))

				disruptionSpec[0].Hostname = ".*.hostname.tld"
				err = disruptionSpec.Validate()
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
