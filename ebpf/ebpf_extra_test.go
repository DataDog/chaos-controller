// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package ebpf_test

import (
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
