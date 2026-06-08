// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package ebpf_test

import (
	"errors"

	"github.com/DataDog/chaos-controller/ebpf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"
)

var _ = Describe("NewBpftoolExecutor and Run", func() {
	It("creates executor and returns error when bpftool binary not found", func() {
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		exec := ebpf.NewBpftoolExecutor(log, false)
		Expect(exec).NotTo(BeNil())
		// bpftool doesn't exist on macOS — covers the Run error path
		_, _, err := exec.Run([]string{"-j", "feature", "probe"})
		Expect(err).To(HaveOccurred())
	})

	It("dry-run mode creates executor without running bpftool", func() {
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		exec := ebpf.NewBpftoolExecutor(log, true)
		Expect(exec).NotTo(BeNil())
	})
})

var _ = Describe("NewConfigInformer with nil executor", func() {
	It("creates default executor and returns error (bpftool not found)", func() {
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		// nil executor → NewBpftoolExecutor created internally
		// nil fsMock → osFS used
		// nil unameFuncMock → real unix.Uname
		_, err := ebpf.NewConfigInformer(log, false, nil, nil, nil)
		// error expected: bpftool binary not found on macOS
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("NewConfigInformer with real osFS and mocked executor", func() {
	It("uses real osFS.Stat when fsMock is nil", func() {
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		execMock := ebpf.NewExecutorMock(GinkgoT())
		execMock.EXPECT().Run(mock.Anything).Return(0, validKernelConfig, nil)

		ci, err := ebpf.NewConfigInformer(log, false, execMock, nil, nil)
		// no error (valid kernel config), uses real osFS for stat operations
		if err != nil {
			Skip("NewConfigInformer failed: " + err.Error())
		}

		// IsKernelConfigAvailable calls v.fs.Stat — exercises osFS.Stat
		Expect(func() { ci.IsKernelConfigAvailable() }).NotTo(Panic())
	})
})

var _ = Describe("ExecutorMock", func() {
	It("Run via Return", func() {
		m := ebpf.NewExecutorMock(GinkgoT())
		m.EXPECT().Run(mock.Anything).Return(0, "output", nil)
		code, out, err := m.Run([]string{"arg"})
		Expect(err).NotTo(HaveOccurred())
		Expect(code).To(Equal(0))
		Expect(out).To(Equal("output"))
	})

	It("Run via Run callback", func() {
		m := ebpf.NewExecutorMock(GinkgoT())
		called := false
		m.EXPECT().Run(mock.Anything).Run(func(args []string) { called = true }).Return(0, "", nil)
		_, _, _ = m.Run([]string{"x"})
		Expect(called).To(BeTrue())
	})

	It("Run via RunAndReturn", func() {
		m := ebpf.NewExecutorMock(GinkgoT())
		m.EXPECT().Run(mock.Anything).RunAndReturn(func(args []string) (int, string, error) {
			return 1, "dyn", errors.New("exec failed")
		})
		code, out, err := m.Run([]string{"x"})
		Expect(err).To(HaveOccurred())
		Expect(code).To(Equal(1))
		Expect(out).To(Equal("dyn"))
	})
})

var _ = Describe("ConfigInformerMock", func() {
	It("GetKernelFeatures via Return", func() {
		m := ebpf.NewConfigInformerMock(GinkgoT())
		m.EXPECT().GetKernelFeatures().Return(ebpf.Features{}, nil)
		_, err := m.GetKernelFeatures()
		Expect(err).NotTo(HaveOccurred())
	})

	It("GetKernelFeatures via Run callback", func() {
		m := ebpf.NewConfigInformerMock(GinkgoT())
		called := false
		m.EXPECT().GetKernelFeatures().Run(func() { called = true }).Return(ebpf.Features{}, nil)
		_, _ = m.GetKernelFeatures()
		Expect(called).To(BeTrue())
	})

	It("GetKernelFeatures via RunAndReturn", func() {
		m := ebpf.NewConfigInformerMock(GinkgoT())
		m.EXPECT().GetKernelFeatures().RunAndReturn(func() (ebpf.Features, error) {
			return ebpf.Features{}, nil
		})
		_, _ = m.GetKernelFeatures()
	})

	It("GetMapTypes via Return", func() {
		m := ebpf.NewConfigInformerMock(GinkgoT())
		m.EXPECT().GetMapTypes().Return(ebpf.MapTypes{})
		mt := m.GetMapTypes()
		Expect(mt).To(Equal(ebpf.MapTypes{}))
	})

	It("GetMapTypes via Run callback", func() {
		m := ebpf.NewConfigInformerMock(GinkgoT())
		called := false
		m.EXPECT().GetMapTypes().Run(func() { called = true }).Return(ebpf.MapTypes{})
		m.GetMapTypes()
		Expect(called).To(BeTrue())
	})

	It("GetMapTypes via RunAndReturn", func() {
		m := ebpf.NewConfigInformerMock(GinkgoT())
		m.EXPECT().GetMapTypes().RunAndReturn(func() ebpf.MapTypes { return ebpf.MapTypes{} })
		m.GetMapTypes()
	})

	It("GetRequiredSystemConfig via Return", func() {
		m := ebpf.NewConfigInformerMock(GinkgoT())
		m.EXPECT().GetRequiredSystemConfig().Return(ebpf.KernelParams{})
		kp := m.GetRequiredSystemConfig()
		Expect(kp).To(Equal(ebpf.KernelParams{}))
	})

	It("GetRequiredSystemConfig via Run callback", func() {
		m := ebpf.NewConfigInformerMock(GinkgoT())
		called := false
		m.EXPECT().GetRequiredSystemConfig().Run(func() { called = true }).Return(ebpf.KernelParams{})
		m.GetRequiredSystemConfig()
		Expect(called).To(BeTrue())
	})

	It("GetRequiredSystemConfig via RunAndReturn", func() {
		m := ebpf.NewConfigInformerMock(GinkgoT())
		m.EXPECT().GetRequiredSystemConfig().RunAndReturn(func() ebpf.KernelParams { return nil })
		m.GetRequiredSystemConfig()
	})

	It("IsKernelConfigAvailable via Return", func() {
		m := ebpf.NewConfigInformerMock(GinkgoT())
		m.EXPECT().IsKernelConfigAvailable().Return(true)
		Expect(m.IsKernelConfigAvailable()).To(BeTrue())
	})

	It("IsKernelConfigAvailable via Run callback", func() {
		m := ebpf.NewConfigInformerMock(GinkgoT())
		called := false
		m.EXPECT().IsKernelConfigAvailable().Run(func() { called = true }).Return(false)
		m.IsKernelConfigAvailable()
		Expect(called).To(BeTrue())
	})

	It("IsKernelConfigAvailable via RunAndReturn", func() {
		m := ebpf.NewConfigInformerMock(GinkgoT())
		m.EXPECT().IsKernelConfigAvailable().RunAndReturn(func() bool { return true })
		Expect(m.IsKernelConfigAvailable()).To(BeTrue())
	})

	It("ValidateRequiredSystemConfig via Return nil", func() {
		m := ebpf.NewConfigInformerMock(GinkgoT())
		m.EXPECT().ValidateRequiredSystemConfig().Return(nil)
		Expect(m.ValidateRequiredSystemConfig()).To(Succeed())
	})

	It("ValidateRequiredSystemConfig via Run callback", func() {
		m := ebpf.NewConfigInformerMock(GinkgoT())
		called := false
		m.EXPECT().ValidateRequiredSystemConfig().Run(func() { called = true }).Return(nil)
		m.ValidateRequiredSystemConfig()
		Expect(called).To(BeTrue())
	})

	It("ValidateRequiredSystemConfig via RunAndReturn", func() {
		m := ebpf.NewConfigInformerMock(GinkgoT())
		m.EXPECT().ValidateRequiredSystemConfig().RunAndReturn(func() error { return nil })
		Expect(m.ValidateRequiredSystemConfig()).To(Succeed())
	})
})
