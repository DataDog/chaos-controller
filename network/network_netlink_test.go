// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package network

import (
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
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

var _ = Describe("NetlinkAdapterMock", func() {
	It("covers mock methods via Return and Run", func() {
		m := NewNetlinkAdapterMock(GinkgoT())
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		link := NewNetlinkLinkMock(GinkgoT())
		route := NewNetlinkRouteMock(GinkgoT())

		m.EXPECT().LinkList(mock.Anything, mock.Anything).Return([]NetlinkLink{link}, nil)
		m.EXPECT().LinkByIndex(1).Return(link, nil)
		m.EXPECT().LinkByName("eth0").Return(link, nil)
		m.EXPECT().DefaultRoutes().Return([]NetlinkRoute{route}, nil)

		links, _ := m.LinkList(false, log)
		Expect(links).To(HaveLen(1))
		l, _ := m.LinkByIndex(1)
		Expect(l).To(Equal(link))
		ln, _ := m.LinkByName("eth0")
		Expect(ln).To(Equal(link))
		routes, _ := m.DefaultRoutes()
		Expect(routes).To(HaveLen(1))
	})

	It("DefaultRoutes via Run callback", func() {
		m := NewNetlinkAdapterMock(GinkgoT())
		called := false
		m.EXPECT().DefaultRoutes().Run(func() { called = true }).Return(nil, nil)
		_, _ = m.DefaultRoutes()
		Expect(called).To(BeTrue())
	})

	It("DefaultRoutes via RunAndReturn", func() {
		m := NewNetlinkAdapterMock(GinkgoT())
		m.EXPECT().DefaultRoutes().RunAndReturn(func() ([]NetlinkRoute, error) { return nil, nil })
		_, _ = m.DefaultRoutes()
	})

	It("LinkByIndex via Run callback", func() {
		m := NewNetlinkAdapterMock(GinkgoT())
		called := false
		m.EXPECT().LinkByIndex(1).Run(func(idx int) { called = true }).Return(nil, nil)
		_, _ = m.LinkByIndex(1)
		Expect(called).To(BeTrue())
	})

	It("LinkByIndex via RunAndReturn", func() {
		m := NewNetlinkAdapterMock(GinkgoT())
		m.EXPECT().LinkByIndex(1).RunAndReturn(func(idx int) (NetlinkLink, error) { return nil, nil })
		_, _ = m.LinkByIndex(1)
	})

	It("LinkByName via Run callback", func() {
		m := NewNetlinkAdapterMock(GinkgoT())
		called := false
		m.EXPECT().LinkByName("eth0").Run(func(name string) { called = true }).Return(nil, nil)
		_, _ = m.LinkByName("eth0")
		Expect(called).To(BeTrue())
	})

	It("LinkByName via RunAndReturn", func() {
		m := NewNetlinkAdapterMock(GinkgoT())
		m.EXPECT().LinkByName("eth0").RunAndReturn(func(name string) (NetlinkLink, error) { return nil, nil })
		_, _ = m.LinkByName("eth0")
	})

	It("LinkList via Run callback", func() {
		m := NewNetlinkAdapterMock(GinkgoT())
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		called := false
		m.EXPECT().LinkList(mock.Anything, mock.Anything).Run(func(useLocalhost bool, l *zap.SugaredLogger) { called = true }).Return(nil, nil)
		_, _ = m.LinkList(false, log)
		Expect(called).To(BeTrue())
	})

	It("LinkList via RunAndReturn", func() {
		m := NewNetlinkAdapterMock(GinkgoT())
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		m.EXPECT().LinkList(mock.Anything, mock.Anything).RunAndReturn(func(useLocalhost bool, l *zap.SugaredLogger) ([]NetlinkLink, error) { return nil, nil })
		_, _ = m.LinkList(false, log)
	})
})

var _ = Describe("NetlinkLinkMock", func() {
	It("covers Name, SetTxQLen, TxQLen via Return", func() {
		m := NewNetlinkLinkMock(GinkgoT())
		m.EXPECT().Name().Return("eth0")
		m.EXPECT().SetTxQLen(1000).Return(nil)
		m.EXPECT().TxQLen().Return(1000)

		Expect(m.Name()).To(Equal("eth0"))
		Expect(m.SetTxQLen(1000)).To(Succeed())
		Expect(m.TxQLen()).To(Equal(1000))
	})

	It("Name via Run/RunAndReturn", func() {
		m1 := NewNetlinkLinkMock(GinkgoT())
		called := false
		m1.EXPECT().Name().Run(func() { called = true }).Return("lo")
		m1.Name()
		Expect(called).To(BeTrue())

		m2 := NewNetlinkLinkMock(GinkgoT())
		m2.EXPECT().Name().RunAndReturn(func() string { return "eth0" })
		Expect(m2.Name()).To(Equal("eth0"))
	})

	It("SetTxQLen via Run/RunAndReturn", func() {
		m1 := NewNetlinkLinkMock(GinkgoT())
		called := false
		m1.EXPECT().SetTxQLen(500).Run(func(q int) { called = true }).Return(nil)
		_ = m1.SetTxQLen(500)
		Expect(called).To(BeTrue())

		m2 := NewNetlinkLinkMock(GinkgoT())
		m2.EXPECT().SetTxQLen(500).RunAndReturn(func(q int) error { return nil })
		Expect(m2.SetTxQLen(500)).To(Succeed())
	})

	It("TxQLen via Run/RunAndReturn", func() {
		m1 := NewNetlinkLinkMock(GinkgoT())
		called := false
		m1.EXPECT().TxQLen().Run(func() { called = true }).Return(0)
		_ = m1.TxQLen()
		Expect(called).To(BeTrue())

		m2 := NewNetlinkLinkMock(GinkgoT())
		m2.EXPECT().TxQLen().RunAndReturn(func() int { return 100 })
		Expect(m2.TxQLen()).To(Equal(100))
	})
})

var _ = Describe("NetlinkRouteMock", func() {
	It("covers Gateway and Link via Return", func() {
		m := NewNetlinkRouteMock(GinkgoT())
		link := NewNetlinkLinkMock(GinkgoT())
		gw := net.ParseIP("10.0.0.1")

		m.EXPECT().Gateway().Return(gw)
		m.EXPECT().Link().Return(link)

		Expect(m.Gateway()).To(Equal(gw))
		Expect(m.Link()).To(Equal(link))
	})

	It("Gateway via Run/RunAndReturn", func() {
		m1 := NewNetlinkRouteMock(GinkgoT())
		called := false
		m1.EXPECT().Gateway().Run(func() { called = true }).Return(nil)
		_ = m1.Gateway()
		Expect(called).To(BeTrue())

		m2 := NewNetlinkRouteMock(GinkgoT())
		m2.EXPECT().Gateway().RunAndReturn(func() net.IP { return nil })
		_ = m2.Gateway()
	})

	It("Link via Run/RunAndReturn", func() {
		m1 := NewNetlinkRouteMock(GinkgoT())
		called := false
		m1.EXPECT().Link().Run(func() { called = true }).Return(nil)
		_ = m1.Link()
		Expect(called).To(BeTrue())

		m2 := NewNetlinkRouteMock(GinkgoT())
		m2.EXPECT().Link().RunAndReturn(func() NetlinkLink { return nil })
		_ = m2.Link()
	})
})
