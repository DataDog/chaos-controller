// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package cgroup_test

import (
	"github.com/DataDog/chaos-controller/cgroup"
	"github.com/DataDog/chaos-controller/cpuset"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zaptest"
)

var _ = Describe("NewManager (non-Linux)", func() {
	It("returns error (not implemented on non-Linux)", func() {
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		_, err := cgroup.NewManager(false, 1, "/sys/fs/cgroup", log)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("not implemented"))
	})
})

var _ = Describe("ManagerMock", func() {
	It("covers all mock methods via Return", func() {
		m := cgroup.NewManagerMock(GinkgoT())
		cs := cpuset.NewCPUSet(0, 1)

		m.EXPECT().IsCgroupV2().Return(true)
		m.EXPECT().Join(1).Return(nil)
		m.EXPECT().Read("cpu", "cpuset").Return("0-1", nil)
		m.EXPECT().ReadCPUSet().Return(cs, nil)
		m.EXPECT().Write("cpu", "cpuset", "0-1").Return(nil)
		m.EXPECT().RelativePath("cpu").Return("/cgroup/cpu")

		Expect(m.IsCgroupV2()).To(BeTrue())
		Expect(m.Join(1)).To(Succeed())
		v, _ := m.Read("cpu", "cpuset")
		Expect(v).To(Equal("0-1"))
		result, _ := m.ReadCPUSet()
		Expect(result.Equals(cs)).To(BeTrue())
		Expect(m.Write("cpu", "cpuset", "0-1")).To(Succeed())
		Expect(m.RelativePath("cpu")).To(Equal("/cgroup/cpu"))
	})

	It("IsCgroupV2 via Run/RunAndReturn", func() {
		m1 := cgroup.NewManagerMock(GinkgoT())
		called := false
		m1.EXPECT().IsCgroupV2().Run(func() { called = true }).Return(false)
		m1.IsCgroupV2()
		Expect(called).To(BeTrue())

		m2 := cgroup.NewManagerMock(GinkgoT())
		m2.EXPECT().IsCgroupV2().RunAndReturn(func() bool { return true })
		Expect(m2.IsCgroupV2()).To(BeTrue())
	})

	It("Join via Run/RunAndReturn", func() {
		m1 := cgroup.NewManagerMock(GinkgoT())
		called := false
		m1.EXPECT().Join(1).Run(func(pid int) { called = true }).Return(nil)
		_ = m1.Join(1)
		Expect(called).To(BeTrue())

		m2 := cgroup.NewManagerMock(GinkgoT())
		m2.EXPECT().Join(1).RunAndReturn(func(pid int) error { return nil })
		Expect(m2.Join(1)).To(Succeed())
	})

	It("Read via Run/RunAndReturn", func() {
		m1 := cgroup.NewManagerMock(GinkgoT())
		called := false
		m1.EXPECT().Read("cpu", "file").Run(func(ctrl, file string) { called = true }).Return("", nil)
		_, _ = m1.Read("cpu", "file")
		Expect(called).To(BeTrue())

		m2 := cgroup.NewManagerMock(GinkgoT())
		m2.EXPECT().Read("cpu", "file").RunAndReturn(func(ctrl, file string) (string, error) { return "v", nil })
		v, _ := m2.Read("cpu", "file")
		Expect(v).To(Equal("v"))
	})

	It("ReadCPUSet via Run/RunAndReturn", func() {
		cs := cpuset.NewCPUSet(0)
		m1 := cgroup.NewManagerMock(GinkgoT())
		called := false
		m1.EXPECT().ReadCPUSet().Run(func() { called = true }).Return(cs, nil)
		_, _ = m1.ReadCPUSet()
		Expect(called).To(BeTrue())

		m2 := cgroup.NewManagerMock(GinkgoT())
		m2.EXPECT().ReadCPUSet().RunAndReturn(func() (cpuset.CPUSet, error) { return cs, nil })
		_, _ = m2.ReadCPUSet()
	})

	It("RelativePath via Run/RunAndReturn", func() {
		m1 := cgroup.NewManagerMock(GinkgoT())
		called := false
		m1.EXPECT().RelativePath("cpu").Run(func(ctrl string) { called = true }).Return("/cg/cpu")
		m1.RelativePath("cpu")
		Expect(called).To(BeTrue())

		m2 := cgroup.NewManagerMock(GinkgoT())
		m2.EXPECT().RelativePath("cpu").RunAndReturn(func(ctrl string) string { return "/cg" })
		Expect(m2.RelativePath("cpu")).To(Equal("/cg"))
	})

	It("Write via Run/RunAndReturn", func() {
		m1 := cgroup.NewManagerMock(GinkgoT())
		called := false
		m1.EXPECT().Write("cpu", "file", "val").Run(func(ctrl, file, data string) { called = true }).Return(nil)
		_ = m1.Write("cpu", "file", "val")
		Expect(called).To(BeTrue())

		m2 := cgroup.NewManagerMock(GinkgoT())
		m2.EXPECT().Write("cpu", "file", "val").RunAndReturn(func(ctrl, file, data string) error { return nil })
		Expect(m2.Write("cpu", "file", "val")).To(Succeed())
	})
})
