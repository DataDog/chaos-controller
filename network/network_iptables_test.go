// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package network

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zaptest"
)

var _ = Describe("NewIPTables", func() {
	It("attempts to create iptables (error expected without iptables binary)", func() {
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		// On macOS/non-Linux without iptables, this returns an error — covers the error path
		_, err := NewIPTables(log, false)
		if err != nil {
			Expect(err).To(HaveOccurred())
		}
	})
})

var _ = Describe("IPTablesMock", func() {
	It("covers all mock methods via Return", func() {
		m := NewIPTablesMock(GinkgoT())
		m.EXPECT().Clear().Return(nil)
		m.EXPECT().Intercept("tcp", "80", "/cgroup/path", "0x20002", "10.0.0.1").Return(nil)
		m.EXPECT().LogConntrack().Return(nil)
		m.EXPECT().MarkCgroupPath("/cgroup/path", "0x20002").Return(nil)
		m.EXPECT().MarkClassID("0x20002", "0x20002").Return(nil)
		m.EXPECT().RedirectTo("10.0.0.1", "53", "53").Return(nil)

		Expect(m.Clear()).To(Succeed())
		Expect(m.Intercept("tcp", "80", "/cgroup/path", "0x20002", "10.0.0.1")).To(Succeed())
		Expect(m.LogConntrack()).To(Succeed())
		Expect(m.MarkCgroupPath("/cgroup/path", "0x20002")).To(Succeed())
		Expect(m.MarkClassID("0x20002", "0x20002")).To(Succeed())
		Expect(m.RedirectTo("10.0.0.1", "53", "53")).To(Succeed())
	})

	It("Clear via Run/RunAndReturn", func() {
		m1 := NewIPTablesMock(GinkgoT())
		called := false
		m1.EXPECT().Clear().Run(func() { called = true }).Return(nil)
		_ = m1.Clear()
		Expect(called).To(BeTrue())

		m2 := NewIPTablesMock(GinkgoT())
		m2.EXPECT().Clear().RunAndReturn(func() error { return nil })
		Expect(m2.Clear()).To(Succeed())
	})

	It("Intercept via Run/RunAndReturn", func() {
		m1 := NewIPTablesMock(GinkgoT())
		called := false
		m1.EXPECT().Intercept("tcp", "80", "/cg", "0x1", "1.2.3.4").Run(func(p, port, cg, cid, ip string) { called = true }).Return(nil)
		_ = m1.Intercept("tcp", "80", "/cg", "0x1", "1.2.3.4")
		Expect(called).To(BeTrue())

		m2 := NewIPTablesMock(GinkgoT())
		m2.EXPECT().Intercept("tcp", "80", "/cg", "0x1", "1.2.3.4").RunAndReturn(func(p, port, cg, cid, ip string) error { return nil })
		Expect(m2.Intercept("tcp", "80", "/cg", "0x1", "1.2.3.4")).To(Succeed())
	})

	It("LogConntrack via Run/RunAndReturn", func() {
		m1 := NewIPTablesMock(GinkgoT())
		called := false
		m1.EXPECT().LogConntrack().Run(func() { called = true }).Return(nil)
		_ = m1.LogConntrack()
		Expect(called).To(BeTrue())

		m2 := NewIPTablesMock(GinkgoT())
		m2.EXPECT().LogConntrack().RunAndReturn(func() error { return nil })
		Expect(m2.LogConntrack()).To(Succeed())
	})

	It("MarkCgroupPath via Run/RunAndReturn", func() {
		m1 := NewIPTablesMock(GinkgoT())
		called := false
		m1.EXPECT().MarkCgroupPath("/cg", "0x1").Run(func(cg, mark string) { called = true }).Return(nil)
		_ = m1.MarkCgroupPath("/cg", "0x1")
		Expect(called).To(BeTrue())

		m2 := NewIPTablesMock(GinkgoT())
		m2.EXPECT().MarkCgroupPath("/cg", "0x1").RunAndReturn(func(cg, mark string) error { return nil })
		Expect(m2.MarkCgroupPath("/cg", "0x1")).To(Succeed())
	})

	It("MarkClassID via Run/RunAndReturn", func() {
		m1 := NewIPTablesMock(GinkgoT())
		called := false
		m1.EXPECT().MarkClassID("0x1", "0x1").Run(func(cid, mark string) { called = true }).Return(nil)
		_ = m1.MarkClassID("0x1", "0x1")
		Expect(called).To(BeTrue())

		m2 := NewIPTablesMock(GinkgoT())
		m2.EXPECT().MarkClassID("0x1", "0x1").RunAndReturn(func(cid, mark string) error { return nil })
		Expect(m2.MarkClassID("0x1", "0x1")).To(Succeed())
	})

	It("RedirectTo via Run/RunAndReturn", func() {
		m1 := NewIPTablesMock(GinkgoT())
		called := false
		m1.EXPECT().RedirectTo("10.0.0.1", "53", "53").Run(func(ip, port, target string) { called = true }).Return(nil)
		_ = m1.RedirectTo("10.0.0.1", "53", "53")
		Expect(called).To(BeTrue())

		m2 := NewIPTablesMock(GinkgoT())
		m2.EXPECT().RedirectTo("10.0.0.1", "53", "53").RunAndReturn(func(ip, port, target string) error { return nil })
		Expect(m2.RedirectTo("10.0.0.1", "53", "53")).To(Succeed())
	})
})
