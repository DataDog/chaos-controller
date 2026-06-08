// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package network

import (
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NewNetlinkAdapter", func() {
	It("returns a non-nil adapter", func() {
		adapter := NewNetlinkAdapter()
		Expect(adapter).NotTo(BeNil())
	})
})

var _ = Describe("netlinkLink value type", func() {
	It("Name returns stored name", func() {
		link := &netlinkLink{name: "eth0", txQLen: 1000}
		Expect(link.Name()).To(Equal("eth0"))
	})

	It("TxQLen returns stored qlen", func() {
		link := &netlinkLink{name: "eth0", txQLen: 500}
		Expect(link.TxQLen()).To(Equal(500))
	})
})

var _ = Describe("netlinkRoute value type", func() {
	It("Link returns stored link", func() {
		link := &netlinkLink{name: "eth0"}
		route := netlinkRoute{link: link, gw: net.ParseIP("192.168.1.1")}
		Expect(route.Link()).To(Equal(link))
	})

	It("Gateway returns stored gateway", func() {
		gw := net.ParseIP("10.0.0.1")
		route := netlinkRoute{link: &netlinkLink{name: "eth0"}, gw: gw}
		Expect(route.Gateway()).To(Equal(gw))
	})

	It("String returns formatted string", func() {
		link := &netlinkLink{name: "eth0"}
		gw := net.ParseIP("10.0.0.1")
		route := netlinkRoute{link: link, gw: gw}
		Expect(route.String()).To(ContainSubstring("eth0"))
		Expect(route.String()).To(ContainSubstring("10.0.0.1"))
	})
})
