// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package cgroup

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("allCGroupManagerMock", func() {
	It("covers all mock methods via Return", func() {
		m := newAllCGroupManagerMock(GinkgoT())
		m.EXPECT().EnterPid(mock.Anything, mock.Anything).Return(nil)
		m.EXPECT().GetPaths().Return(map[string]string{"cpu": "/sys/fs/cgroup/cpu"})
		m.EXPECT().Path("cpu").Return("/cgroup/cpu")
		m.EXPECT().PathExists(mock.Anything).Return(true)
		m.EXPECT().ReadFile("dir", "file").Return("content", nil)
		m.EXPECT().WriteFile("dir", "file", "data").Return(nil)

		Expect(m.EnterPid(map[string]string{}, 1)).To(Succeed())
		Expect(m.GetPaths()).To(HaveKeyWithValue("cpu", "/sys/fs/cgroup/cpu"))
		Expect(m.Path("cpu")).To(Equal("/cgroup/cpu"))
		Expect(m.PathExists("/path")).To(BeTrue())
		v, _ := m.ReadFile("dir", "file")
		Expect(v).To(Equal("content"))
		Expect(m.WriteFile("dir", "file", "data")).To(Succeed())
	})

	It("EnterPid via Run callback", func() {
		m := newAllCGroupManagerMock(GinkgoT())
		called := false
		m.EXPECT().EnterPid(mock.Anything, mock.Anything).Run(func(paths map[string]string, pid int) { called = true }).Return(nil)
		_ = m.EnterPid(nil, 1)
		Expect(called).To(BeTrue())
	})

	It("EnterPid via RunAndReturn", func() {
		m := newAllCGroupManagerMock(GinkgoT())
		m.EXPECT().EnterPid(mock.Anything, mock.Anything).RunAndReturn(func(paths map[string]string, pid int) error { return nil })
		Expect(m.EnterPid(nil, 1)).To(Succeed())
	})

	It("GetPaths via Run callback", func() {
		m := newAllCGroupManagerMock(GinkgoT())
		called := false
		m.EXPECT().GetPaths().Run(func() { called = true }).Return(nil)
		_ = m.GetPaths()
		Expect(called).To(BeTrue())
	})

	It("GetPaths via RunAndReturn", func() {
		m := newAllCGroupManagerMock(GinkgoT())
		m.EXPECT().GetPaths().RunAndReturn(func() map[string]string { return nil })
		_ = m.GetPaths()
	})

	It("Path via Run callback", func() {
		m := newAllCGroupManagerMock(GinkgoT())
		called := false
		m.EXPECT().Path(mock.Anything).Run(func(s string) { called = true }).Return("")
		_ = m.Path("x")
		Expect(called).To(BeTrue())
	})

	It("Path via RunAndReturn", func() {
		m := newAllCGroupManagerMock(GinkgoT())
		m.EXPECT().Path(mock.Anything).RunAndReturn(func(s string) string { return "/cg" })
		Expect(m.Path("cpu")).To(Equal("/cg"))
	})

	It("PathExists via Run callback", func() {
		m := newAllCGroupManagerMock(GinkgoT())
		called := false
		m.EXPECT().PathExists(mock.Anything).Run(func(s string) { called = true }).Return(false)
		_ = m.PathExists("x")
		Expect(called).To(BeTrue())
	})

	It("PathExists via RunAndReturn", func() {
		m := newAllCGroupManagerMock(GinkgoT())
		m.EXPECT().PathExists(mock.Anything).RunAndReturn(func(s string) bool { return true })
		Expect(m.PathExists("x")).To(BeTrue())
	})

	It("ReadFile via Run callback", func() {
		m := newAllCGroupManagerMock(GinkgoT())
		called := false
		m.EXPECT().ReadFile(mock.Anything, mock.Anything).Run(func(dir, file string) { called = true }).Return("", nil)
		_, _ = m.ReadFile("d", "f")
		Expect(called).To(BeTrue())
	})

	It("ReadFile via RunAndReturn", func() {
		m := newAllCGroupManagerMock(GinkgoT())
		m.EXPECT().ReadFile(mock.Anything, mock.Anything).RunAndReturn(func(dir, file string) (string, error) { return "val", nil })
		v, _ := m.ReadFile("d", "f")
		Expect(v).To(Equal("val"))
	})

	It("WriteFile via Run callback", func() {
		m := newAllCGroupManagerMock(GinkgoT())
		called := false
		m.EXPECT().WriteFile(mock.Anything, mock.Anything, mock.Anything).Run(func(dir, file, data string) { called = true }).Return(nil)
		_ = m.WriteFile("d", "f", "v")
		Expect(called).To(BeTrue())
	})

	It("WriteFile via RunAndReturn", func() {
		m := newAllCGroupManagerMock(GinkgoT())
		m.EXPECT().WriteFile(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(dir, file, data string) error { return nil })
		Expect(m.WriteFile("d", "f", "v")).To(Succeed())
	})
})

var _ = Describe("instCGroupManagerMock", func() {
	It("covers GetPaths and Path via Return", func() {
		m := newInstCGroupManagerMock(GinkgoT())
		m.EXPECT().GetPaths().Return(map[string]string{"cpu": "/cg/cpu"})
		m.EXPECT().Path("cpu").Return("/cg/cpu")

		Expect(m.GetPaths()).To(HaveKeyWithValue("cpu", "/cg/cpu"))
		Expect(m.Path("cpu")).To(Equal("/cg/cpu"))
	})

	It("GetPaths via Run/RunAndReturn", func() {
		m1 := newInstCGroupManagerMock(GinkgoT())
		called := false
		m1.EXPECT().GetPaths().Run(func() { called = true }).Return(nil)
		_ = m1.GetPaths()
		Expect(called).To(BeTrue())

		m2 := newInstCGroupManagerMock(GinkgoT())
		m2.EXPECT().GetPaths().RunAndReturn(func() map[string]string { return nil })
		_ = m2.GetPaths()
	})

	It("Path via Run/RunAndReturn", func() {
		m1 := newInstCGroupManagerMock(GinkgoT())
		called := false
		m1.EXPECT().Path(mock.Anything).Run(func(s string) { called = true }).Return("")
		_ = m1.Path("cpu")
		Expect(called).To(BeTrue())

		m2 := newInstCGroupManagerMock(GinkgoT())
		m2.EXPECT().Path(mock.Anything).RunAndReturn(func(s string) string { return "/cg" })
		Expect(m2.Path("cpu")).To(Equal("/cg"))
	})
})

var _ = Describe("pkgCGroupManagerMock", func() {
	It("covers all methods via Return", func() {
		m := newPkgCGroupManagerMock(GinkgoT())
		m.EXPECT().EnterPid(mock.Anything, mock.Anything).Return(nil)
		m.EXPECT().PathExists(mock.Anything).Return(false)
		m.EXPECT().ReadFile("d", "f").Return("v", nil)
		m.EXPECT().WriteFile("d", "f", "v").Return(nil)

		Expect(m.EnterPid(nil, 1)).To(Succeed())
		Expect(m.PathExists("/x")).To(BeFalse())
		val, _ := m.ReadFile("d", "f")
		Expect(val).To(Equal("v"))
		Expect(m.WriteFile("d", "f", "v")).To(Succeed())
	})

	It("EnterPid via Run/RunAndReturn", func() {
		m1 := newPkgCGroupManagerMock(GinkgoT())
		called := false
		m1.EXPECT().EnterPid(mock.Anything, mock.Anything).Run(func(paths map[string]string, pid int) { called = true }).Return(nil)
		_ = m1.EnterPid(nil, 1)
		Expect(called).To(BeTrue())

		m2 := newPkgCGroupManagerMock(GinkgoT())
		m2.EXPECT().EnterPid(mock.Anything, mock.Anything).RunAndReturn(func(paths map[string]string, pid int) error { return nil })
		Expect(m2.EnterPid(nil, 1)).To(Succeed())
	})

	It("PathExists via Run/RunAndReturn", func() {
		m1 := newPkgCGroupManagerMock(GinkgoT())
		called := false
		m1.EXPECT().PathExists(mock.Anything).Run(func(s string) { called = true }).Return(false)
		_ = m1.PathExists("x")
		Expect(called).To(BeTrue())

		m2 := newPkgCGroupManagerMock(GinkgoT())
		m2.EXPECT().PathExists(mock.Anything).RunAndReturn(func(s string) bool { return true })
		Expect(m2.PathExists("x")).To(BeTrue())
	})

	It("ReadFile via Run/RunAndReturn", func() {
		m1 := newPkgCGroupManagerMock(GinkgoT())
		called := false
		m1.EXPECT().ReadFile(mock.Anything, mock.Anything).Run(func(dir, file string) { called = true }).Return("", nil)
		_, _ = m1.ReadFile("d", "f")
		Expect(called).To(BeTrue())

		m2 := newPkgCGroupManagerMock(GinkgoT())
		m2.EXPECT().ReadFile(mock.Anything, mock.Anything).RunAndReturn(func(dir, file string) (string, error) { return "v", nil })
		v, _ := m2.ReadFile("d", "f")
		Expect(v).To(Equal("v"))
	})

	It("WriteFile via Run/RunAndReturn", func() {
		m1 := newPkgCGroupManagerMock(GinkgoT())
		called := false
		m1.EXPECT().WriteFile(mock.Anything, mock.Anything, mock.Anything).Run(func(dir, file, data string) { called = true }).Return(nil)
		_ = m1.WriteFile("d", "f", "v")
		Expect(called).To(BeTrue())

		m2 := newPkgCGroupManagerMock(GinkgoT())
		m2.EXPECT().WriteFile(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(dir, file, data string) error { return nil })
		Expect(m2.WriteFile("d", "f", "v")).To(Succeed())
	})
})
