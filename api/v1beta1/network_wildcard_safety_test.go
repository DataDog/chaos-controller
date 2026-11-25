// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package v1beta1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Network wildcard host safety", func() {
	BeforeEach(func() {
		savedClient := k8sClient
		k8sClient = nil
		DeferCleanup(func() { k8sClient = savedClient })
	})

	DescribeTable("rejects an unbounded host rule even when other rules are scoped", func(hosts []NetworkDisruptionHostSpec, includeService, wantCaught bool) {
		network := &NetworkDisruptionSpec{Drop: 100, Hosts: hosts}
		if includeService {
			network.Services = []NetworkDisruptionServiceSpec{{Name: "backend", Namespace: "ns"}}
		}
		Expect(network.Validate()).To(Succeed())
		disruption := &Disruption{Spec: DisruptionSpec{Network: network}}
		Expect(safetyNetMissingNetworkFilters(disruption)).To(Equal(wantCaught))
	},
		Entry("empty host entry", []NetworkDisruptionHostSpec{{}}, false, true),
		Entry("wildcard mixed with a scoped host", []NetworkDisruptionHostSpec{{Host: "example.com"}, {}}, false, true),
		Entry("wildcard mixed with a service", []NetworkDisruptionHostSpec{{}}, true, true),
		Entry("resolver alone does not restrict traffic", []NetworkDisruptionHostSpec{{DNSResolver: "pod"}}, false, true),
		Entry("no rules", nil, false, true),
		Entry("port restriction", []NetworkDisruptionHostSpec{{Port: 443}}, false, false),
		Entry("protocol restriction", []NetworkDisruptionHostSpec{{Protocol: "tcp"}}, false, false),
		Entry("connection-state restriction", []NetworkDisruptionHostSpec{{ConnState: "new"}}, false, false),
		Entry("hostname restriction", []NetworkDisruptionHostSpec{{Host: "example.com"}}, false, false),
	)

	It("preserves the explicit safety override", func() {
		disruption := &Disruption{Spec: DisruptionSpec{
			Network:    &NetworkDisruptionSpec{Drop: 100, Hosts: []NetworkDisruptionHostSpec{{}}},
			Unsafemode: &UnsafemodeSpec{DisableNeitherHostNorPort: true},
		}}
		Expect(safetyNetMissingNetworkFilters(disruption)).To(BeFalse())
	})
})
