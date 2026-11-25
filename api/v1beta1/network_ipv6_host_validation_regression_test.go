// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package v1beta1_test

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Network IPv6 host admission", func() {
	DescribeTable("rejects address families the injector cannot honor", func(host string) {
		for _, allowed := range []bool{false, true} {
			spec := v1beta1.NetworkDisruptionSpec{Drop: 100}
			hosts := []v1beta1.NetworkDisruptionHostSpec{{Host: host, Protocol: "tcp"}}
			if allowed {
				spec.AllowedHosts = hosts
			} else {
				spec.Hosts = hosts
			}
			Expect(spec.Validate()).To(MatchError(ContainSubstring("IPv6")))
		}
	},
		Entry("IPv6 literal", "2001:db8::1"),
		Entry("IPv6 CIDR", "2001:db8::/64"),
		Entry("IPv6 wildcard CIDR", "::/0"),
	)

	DescribeTable("keeps supported host forms", func(host string) {
		spec := v1beta1.NetworkDisruptionSpec{Drop: 100, Hosts: []v1beta1.NetworkDisruptionHostSpec{{Host: host}}}
		Expect(spec.Validate()).To(Succeed())
	},
		Entry("IPv4 literal", "192.0.2.1"),
		Entry("IPv4 CIDR", "192.0.2.0/24"),
		Entry("IPv4-mapped literal", "::ffff:192.0.2.1"),
		Entry("hostname", "api.example.com"),
		Entry("wildcard host", ""),
	)
})
