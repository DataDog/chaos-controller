// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package network

import (
	"net"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("DNSClientMock", func() {
	It("ResolveWithStrategy via Return", func() {
		m := NewDNSClientMock(GinkgoT())
		m.EXPECT().ResolveWithStrategy("example.com", "pod").Return([]net.IP{net.ParseIP("1.2.3.4")}, nil)
		ips, err := m.ResolveWithStrategy("example.com", "pod")
		Expect(err).NotTo(HaveOccurred())
		Expect(ips).To(HaveLen(1))
	})

	It("ResolveWithStrategy via Run callback", func() {
		m := NewDNSClientMock(GinkgoT())
		called := false
		m.EXPECT().ResolveWithStrategy(mock.Anything, mock.Anything).Run(func(host, strategy string) { called = true }).Return(nil, nil)
		_, _ = m.ResolveWithStrategy("host", "pod")
		Expect(called).To(BeTrue())
	})

	It("ResolveWithStrategy via RunAndReturn", func() {
		m := NewDNSClientMock(GinkgoT())
		m.EXPECT().ResolveWithStrategy(mock.Anything, mock.Anything).RunAndReturn(func(host, strategy string) ([]net.IP, error) {
			return []net.IP{net.ParseIP("5.6.7.8")}, nil
		})
		ips, _ := m.ResolveWithStrategy("example.com", "pod")
		Expect(ips).To(HaveLen(1))
	})
})

var _ = Describe("DNSResponderMock", func() {
	It("Start and Stop via Return", func() {
		m := NewDNSResponderMock(GinkgoT())
		m.EXPECT().Start().Return(nil)
		m.EXPECT().Stop().Return(nil)
		Expect(m.Start()).To(Succeed())
		Expect(m.Stop()).To(Succeed())
	})

	It("Start via Run/RunAndReturn", func() {
		m1 := NewDNSResponderMock(GinkgoT())
		called := false
		m1.EXPECT().Start().Run(func() { called = true }).Return(nil)
		_ = m1.Start()
		Expect(called).To(BeTrue())

		m2 := NewDNSResponderMock(GinkgoT())
		m2.EXPECT().Start().RunAndReturn(func() error { return nil })
		Expect(m2.Start()).To(Succeed())
	})

	It("Stop via Run/RunAndReturn", func() {
		m1 := NewDNSResponderMock(GinkgoT())
		called := false
		m1.EXPECT().Stop().Run(func() { called = true }).Return(nil)
		_ = m1.Stop()
		Expect(called).To(BeTrue())

		m2 := NewDNSResponderMock(GinkgoT())
		m2.EXPECT().Stop().RunAndReturn(func() error { return nil })
		Expect(m2.Stop()).To(Succeed())
	})
})

var _ = Describe("TrafficControllerMock", func() {
	var (
		srcIP  = &net.IPNet{IP: net.ParseIP("1.2.3.4"), Mask: net.CIDRMask(32, 32)}
		dstIP  = &net.IPNet{IP: net.ParseIP("5.6.7.8"), Mask: net.CIDRMask(32, 32)}
		ifaces = []string{"eth0"}
		protoc = newProtocol(TCP)
		cstate = ConnStateNew
	)

	It("covers all methods via Return", func() {
		m := NewTrafficControllerMock(GinkgoT())
		m.EXPECT().AddBPFFilter(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		m.EXPECT().AddFilter(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(uint32(1001), nil)
		m.EXPECT().AddFlowerFilter(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		m.EXPECT().AddNetem(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		m.EXPECT().AddOutputLimit(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		m.EXPECT().AddPrio(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		m.EXPECT().ClearQdisc(mock.Anything).Return(nil)
		m.EXPECT().ConfigBPFFilter(mock.Anything, mock.Anything).Return(nil)
		m.EXPECT().DeleteFilter(mock.Anything, mock.Anything).Return(nil)

		Expect(m.AddBPFFilter(ifaces, "root", "prog.o", "1:2", "")).To(Succeed())
		_, err := m.AddFilter(ifaces, "root", "", srcIP, dstIP, 80, 0, protoc, cstate, "1:2")
		Expect(err).NotTo(HaveOccurred())
		Expect(m.AddFlowerFilter(ifaces, "root", "", "1:2")).To(Succeed())
		Expect(m.AddNetem(ifaces, "root", "", time.Second, 0, 0, 0, 0)).To(Succeed())
		Expect(m.AddOutputLimit(ifaces, "root", "", 1024)).To(Succeed())
		Expect(m.AddPrio(ifaces, "root", "", 2, [16]uint32{})).To(Succeed())
		Expect(m.ClearQdisc(ifaces)).To(Succeed())
		Expect(m.ConfigBPFFilter(nil, "arg")).To(Succeed())
		Expect(m.DeleteFilter("eth0", 1001)).To(Succeed())
	})

	It("AddBPFFilter via Run/RunAndReturn", func() {
		m1 := NewTrafficControllerMock(GinkgoT())
		called := false
		m1.EXPECT().AddBPFFilter(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Run(func(ifaces []string, p, obj, fid, sec string) { called = true }).Return(nil)
		_ = m1.AddBPFFilter(ifaces, "root", "p.o", "1:2", "")
		Expect(called).To(BeTrue())

		m2 := NewTrafficControllerMock(GinkgoT())
		m2.EXPECT().AddBPFFilter(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			RunAndReturn(func(ifaces []string, p, obj, fid, sec string) error { return nil })
		Expect(m2.AddBPFFilter(ifaces, "root", "p.o", "1:2", "")).To(Succeed())
	})

	It("AddFilter via Run/RunAndReturn", func() {
		m1 := NewTrafficControllerMock(GinkgoT())
		called := false
		m1.EXPECT().AddFilter(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Run(func(ifaces []string, parent, handle string, srcIP, dstIP *net.IPNet, srcPort, dstPort int, prot protocol, state connState, flowid string) {
				called = true
			}).Return(uint32(0), nil)
		_, _ = m1.AddFilter(ifaces, "root", "", srcIP, dstIP, 0, 0, protoc, cstate, "1:2")
		Expect(called).To(BeTrue())

		m2 := NewTrafficControllerMock(GinkgoT())
		m2.EXPECT().AddFilter(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			RunAndReturn(func(ifaces []string, parent, handle string, srcIP, dstIP *net.IPNet, srcPort, dstPort int, prot protocol, state connState, flowid string) (uint32, error) {
				return 0, nil
			})
		_, _ = m2.AddFilter(ifaces, "root", "", nil, nil, 0, 0, protoc, cstate, "1:2")
	})

	It("AddFlowerFilter via Run/RunAndReturn", func() {
		m1 := NewTrafficControllerMock(GinkgoT())
		called := false
		m1.EXPECT().AddFlowerFilter(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Run(func(ifaces []string, parent, handle, flowid string) { called = true }).Return(nil)
		_ = m1.AddFlowerFilter(ifaces, "root", "", "1:2")
		Expect(called).To(BeTrue())

		m2 := NewTrafficControllerMock(GinkgoT())
		m2.EXPECT().AddFlowerFilter(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			RunAndReturn(func(ifaces []string, parent, handle, flowid string) error { return nil })
		Expect(m2.AddFlowerFilter(ifaces, "root", "", "1:2")).To(Succeed())
	})

	It("AddNetem via Run/RunAndReturn", func() {
		m1 := NewTrafficControllerMock(GinkgoT())
		called := false
		m1.EXPECT().AddNetem(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Run(func(ifaces []string, parent, handle string, delay, jitter time.Duration, drop, corrupt, dup int) {
				called = true
			}).Return(nil)
		_ = m1.AddNetem(ifaces, "root", "", 0, 0, 0, 0, 0)
		Expect(called).To(BeTrue())

		m2 := NewTrafficControllerMock(GinkgoT())
		m2.EXPECT().AddNetem(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			RunAndReturn(func(ifaces []string, parent, handle string, delay, jitter time.Duration, drop, corrupt, dup int) error {
				return nil
			})
		Expect(m2.AddNetem(ifaces, "root", "", 0, 0, 0, 0, 0)).To(Succeed())
	})

	It("AddOutputLimit via Run/RunAndReturn", func() {
		m1 := NewTrafficControllerMock(GinkgoT())
		called := false
		m1.EXPECT().AddOutputLimit(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Run(func(ifaces []string, parent, handle string, bps uint) { called = true }).Return(nil)
		_ = m1.AddOutputLimit(ifaces, "root", "", 1024)
		Expect(called).To(BeTrue())

		m2 := NewTrafficControllerMock(GinkgoT())
		m2.EXPECT().AddOutputLimit(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			RunAndReturn(func(ifaces []string, parent, handle string, bps uint) error { return nil })
		Expect(m2.AddOutputLimit(ifaces, "root", "", 1024)).To(Succeed())
	})

	It("AddPrio via Run/RunAndReturn", func() {
		m1 := NewTrafficControllerMock(GinkgoT())
		called := false
		m1.EXPECT().AddPrio(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Run(func(ifaces []string, parent, handle string, bands uint32, priomap [16]uint32) { called = true }).Return(nil)
		_ = m1.AddPrio(ifaces, "root", "", 2, [16]uint32{})
		Expect(called).To(BeTrue())

		m2 := NewTrafficControllerMock(GinkgoT())
		m2.EXPECT().AddPrio(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			RunAndReturn(func(ifaces []string, parent, handle string, bands uint32, priomap [16]uint32) error { return nil })
		Expect(m2.AddPrio(ifaces, "root", "", 2, [16]uint32{})).To(Succeed())
	})

	It("ClearQdisc via Run/RunAndReturn", func() {
		m1 := NewTrafficControllerMock(GinkgoT())
		called := false
		m1.EXPECT().ClearQdisc(mock.Anything).Run(func(ifaces []string) { called = true }).Return(nil)
		_ = m1.ClearQdisc(ifaces)
		Expect(called).To(BeTrue())

		m2 := NewTrafficControllerMock(GinkgoT())
		m2.EXPECT().ClearQdisc(mock.Anything).RunAndReturn(func(ifaces []string) error { return nil })
		Expect(m2.ClearQdisc(ifaces)).To(Succeed())
	})

	It("ConfigBPFFilter via Return only (variadic — RunAndReturn not supported)", func() {
		m := NewTrafficControllerMock(GinkgoT())
		m.EXPECT().ConfigBPFFilter(mock.Anything, mock.Anything).Return(nil)
		Expect(m.ConfigBPFFilter(nil, "arg")).To(Succeed())
	})

	It("DeleteFilter via Run/RunAndReturn", func() {
		m1 := NewTrafficControllerMock(GinkgoT())
		called := false
		m1.EXPECT().DeleteFilter(mock.Anything, mock.Anything).Run(func(iface string, prio uint32) { called = true }).Return(nil)
		_ = m1.DeleteFilter("eth0", 1001)
		Expect(called).To(BeTrue())

		m2 := NewTrafficControllerMock(GinkgoT())
		m2.EXPECT().DeleteFilter(mock.Anything, mock.Anything).RunAndReturn(func(iface string, prio uint32) error { return nil })
		Expect(m2.DeleteFilter("eth0", 1001)).To(Succeed())
	})
})

var _ = Describe("executorMock (internal)", func() {
	It("Run via Run callback", func() {
		m := newExecutorMock(GinkgoT())
		called := false
		m.EXPECT().Run(mock.Anything).Run(func(args []string) { called = true }).Return(0, "", nil)
		_, _, _ = m.Run([]string{"arg"})
		Expect(called).To(BeTrue())
	})

	It("Run via RunAndReturn", func() {
		m := newExecutorMock(GinkgoT())
		m.EXPECT().Run(mock.Anything).RunAndReturn(func(args []string) (int, string, error) { return 1, "out", nil })
		code, out, _ := m.Run([]string{"x"})
		Expect(code).To(Equal(1))
		Expect(out).To(Equal("out"))
	})
})
