// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package process_test

import (
	"os"

	"github.com/DataDog/chaos-controller/process"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/sys/unix"
)

var _ = Describe("Manager (platform-independent)", func() {
	var mgr process.Manager

	BeforeEach(func() {
		mgr = process.NewManager(false)
	})

	Describe("NewManager", func() {
		It("creates a non-nil manager", func() {
			Expect(mgr).NotTo(BeNil())
		})
	})

	Describe("Find", func() {
		It("finds current process by PID", func() {
			proc, err := mgr.Find(os.Getpid())
			Expect(err).NotTo(HaveOccurred())
			Expect(proc).NotTo(BeNil())
		})
	})

	Describe("Exists", func() {
		It("returns true for current process", func() {
			exists, err := mgr.Exists(os.Getpid())
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
		})

		It("returns false for non-existent PID", func() {
			exists, _ := mgr.Exists(99999999)
			Expect(exists).To(BeFalse())
		})
	})

	Describe("Signal", func() {
		It("no-op in dry-run mode", func() {
			dryMgr := process.NewManager(true)
			proc, _ := mgr.Find(os.Getpid())
			Expect(dryMgr.Signal(proc, unix.Signal(0))).To(Succeed())
		})

		It("no-op for PID 0", func() {
			proc := &os.Process{Pid: 0}
			Expect(mgr.Signal(proc, unix.Signal(0))).To(Succeed())
		})

		It("no-op for NotFoundProcessPID", func() {
			proc := &os.Process{Pid: process.NotFoundProcessPID}
			Expect(mgr.Signal(proc, unix.Signal(0))).To(Succeed())
		})

		It("sends signal 0 to current process", func() {
			proc, _ := mgr.Find(os.Getpid())
			Expect(mgr.Signal(proc, unix.Signal(0))).To(Succeed())
		})
	})
})

var _ = Describe("ManagerMock", func() {
	It("covers all mock methods via Return", func() {
		m := process.NewManagerMock(GinkgoT())
		proc := &os.Process{Pid: os.Getpid()}

		m.EXPECT().Prioritize().Return(nil)
		m.EXPECT().ThreadID().Return(1)
		m.EXPECT().ProcessID().Return(2)
		m.EXPECT().SetAffinity([]int{0}).Return(nil)
		m.EXPECT().Find(1).Return(proc, nil)
		m.EXPECT().Exists(1).Return(true, nil)
		m.EXPECT().Signal(proc, unix.Signal(0)).Return(nil)

		Expect(m.Prioritize()).To(Succeed())
		Expect(m.ThreadID()).To(Equal(1))
		Expect(m.ProcessID()).To(Equal(2))
		Expect(m.SetAffinity([]int{0})).To(Succeed())
		p, _ := m.Find(1)
		Expect(p).To(Equal(proc))
		exists, _ := m.Exists(1)
		Expect(exists).To(BeTrue())
		Expect(m.Signal(proc, unix.Signal(0))).To(Succeed())
	})

	It("Exists via Run callback", func() {
		m := process.NewManagerMock(GinkgoT())
		called := false
		m.EXPECT().Exists(1).Run(func(pid int) { called = true }).Return(true, nil)
		_, _ = m.Exists(1)
		Expect(called).To(BeTrue())
	})

	It("Exists via RunAndReturn", func() {
		m := process.NewManagerMock(GinkgoT())
		m.EXPECT().Exists(1).RunAndReturn(func(pid int) (bool, error) { return true, nil })
		exists, _ := m.Exists(1)
		Expect(exists).To(BeTrue())
	})

	It("Find via Run callback", func() {
		m := process.NewManagerMock(GinkgoT())
		called := false
		m.EXPECT().Find(1).Run(func(pid int) { called = true }).Return(nil, nil)
		_, _ = m.Find(1)
		Expect(called).To(BeTrue())
	})

	It("Find via RunAndReturn", func() {
		m := process.NewManagerMock(GinkgoT())
		m.EXPECT().Find(1).RunAndReturn(func(pid int) (*os.Process, error) { return nil, nil })
		_, _ = m.Find(1)
	})

	It("Prioritize via Run callback", func() {
		m := process.NewManagerMock(GinkgoT())
		called := false
		m.EXPECT().Prioritize().Run(func() { called = true }).Return(nil)
		_ = m.Prioritize()
		Expect(called).To(BeTrue())
	})

	It("Prioritize via RunAndReturn", func() {
		m := process.NewManagerMock(GinkgoT())
		m.EXPECT().Prioritize().RunAndReturn(func() error { return nil })
		Expect(m.Prioritize()).To(Succeed())
	})

	It("ProcessID via Run callback", func() {
		m := process.NewManagerMock(GinkgoT())
		called := false
		m.EXPECT().ProcessID().Run(func() { called = true }).Return(0)
		_ = m.ProcessID()
		Expect(called).To(BeTrue())
	})

	It("ProcessID via RunAndReturn", func() {
		m := process.NewManagerMock(GinkgoT())
		m.EXPECT().ProcessID().RunAndReturn(func() int { return 42 })
		Expect(m.ProcessID()).To(Equal(42))
	})

	It("SetAffinity via Run callback", func() {
		m := process.NewManagerMock(GinkgoT())
		called := false
		m.EXPECT().SetAffinity([]int{0}).Run(func(cpus []int) { called = true }).Return(nil)
		_ = m.SetAffinity([]int{0})
		Expect(called).To(BeTrue())
	})

	It("SetAffinity via RunAndReturn", func() {
		m := process.NewManagerMock(GinkgoT())
		m.EXPECT().SetAffinity([]int{0}).RunAndReturn(func(cpus []int) error { return nil })
		Expect(m.SetAffinity([]int{0})).To(Succeed())
	})

	It("Signal via Run callback", func() {
		m := process.NewManagerMock(GinkgoT())
		proc := &os.Process{Pid: os.Getpid()}
		called := false
		m.EXPECT().Signal(proc, unix.Signal(0)).Run(func(p *os.Process, s os.Signal) { called = true }).Return(nil)
		_ = m.Signal(proc, unix.Signal(0))
		Expect(called).To(BeTrue())
	})

	It("Signal via RunAndReturn", func() {
		m := process.NewManagerMock(GinkgoT())
		proc := &os.Process{Pid: os.Getpid()}
		m.EXPECT().Signal(proc, unix.Signal(0)).RunAndReturn(func(p *os.Process, s os.Signal) error { return nil })
		Expect(m.Signal(proc, unix.Signal(0))).To(Succeed())
	})

	It("ThreadID via Run callback", func() {
		m := process.NewManagerMock(GinkgoT())
		called := false
		m.EXPECT().ThreadID().Run(func() { called = true }).Return(0)
		_ = m.ThreadID()
		Expect(called).To(BeTrue())
	})

	It("ThreadID via RunAndReturn", func() {
		m := process.NewManagerMock(GinkgoT())
		m.EXPECT().ThreadID().RunAndReturn(func() int { return 99 })
		Expect(m.ThreadID()).To(Equal(99))
	})
})

var _ = Describe("Runtime", func() {
	It("NewRuntime creates a runtime", func() {
		rt := process.NewRuntime(false)
		Expect(rt).NotTo(BeNil())
	})

	It("GOMAXPROCS returns previous setting", func() {
		rt := process.NewRuntime(false)
		prev := rt.GOMAXPROCS(1)
		Expect(prev).To(BeNumerically(">", 0))
		rt.GOMAXPROCS(prev)
	})

	It("LockOSThread and UnlockOSThread don't panic", func() {
		rt := process.NewRuntime(false)
		Expect(func() {
			rt.LockOSThread()
			rt.UnlockOSThread()
		}).NotTo(Panic())
	})
})

var _ = Describe("RuntimeMock", func() {
	It("GOMAXPROCS via Return", func() {
		m := process.NewRuntimeMock(GinkgoT())
		m.EXPECT().GOMAXPROCS(1).Return(4)
		Expect(m.GOMAXPROCS(1)).To(Equal(4))
	})

	It("GOMAXPROCS via Run callback", func() {
		m := process.NewRuntimeMock(GinkgoT())
		called := false
		m.EXPECT().GOMAXPROCS(1).Run(func(n int) { called = true }).Return(4)
		_ = m.GOMAXPROCS(1)
		Expect(called).To(BeTrue())
	})

	It("GOMAXPROCS via RunAndReturn", func() {
		m := process.NewRuntimeMock(GinkgoT())
		m.EXPECT().GOMAXPROCS(1).RunAndReturn(func(n int) int { return 8 })
		Expect(m.GOMAXPROCS(1)).To(Equal(8))
	})

	It("LockOSThread via Return", func() {
		m := process.NewRuntimeMock(GinkgoT())
		m.EXPECT().LockOSThread().Return()
		m.LockOSThread()
	})

	It("LockOSThread via Run callback", func() {
		m := process.NewRuntimeMock(GinkgoT())
		called := false
		m.EXPECT().LockOSThread().Run(func() { called = true }).Return()
		m.LockOSThread()
		Expect(called).To(BeTrue())
	})

	It("LockOSThread via RunAndReturn", func() {
		m := process.NewRuntimeMock(GinkgoT())
		m.EXPECT().LockOSThread().RunAndReturn(func() {})
		m.LockOSThread()
	})

	It("UnlockOSThread via Return", func() {
		m := process.NewRuntimeMock(GinkgoT())
		m.EXPECT().UnlockOSThread().Return()
		m.UnlockOSThread()
	})

	It("UnlockOSThread via Run callback", func() {
		m := process.NewRuntimeMock(GinkgoT())
		called := false
		m.EXPECT().UnlockOSThread().Run(func() { called = true }).Return()
		m.UnlockOSThread()
		Expect(called).To(BeTrue())
	})

	It("UnlockOSThread via RunAndReturn", func() {
		m := process.NewRuntimeMock(GinkgoT())
		m.EXPECT().UnlockOSThread().RunAndReturn(func() {})
		m.UnlockOSThread()
	})
})
