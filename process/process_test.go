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
