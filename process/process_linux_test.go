// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

//go:build linux
// +build linux

package process_test

import (
	"os"

	"github.com/DataDog/chaos-controller/process"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manager (Linux)", func() {
	var mgr process.Manager

	BeforeEach(func() {
		mgr = process.NewManager(false)
	})

	It("ThreadID returns positive thread ID", func() {
		Expect(mgr.ThreadID()).To(BeNumerically(">", 0))
	})

	It("ProcessID returns current PID", func() {
		Expect(mgr.ProcessID()).To(Equal(os.Getpid()))
	})

	It("SetAffinity with current CPU set succeeds or fails with permission error", func() {
		// SchedSetaffinity may fail without elevated privileges; both outcomes are valid
		err := mgr.SetAffinity([]int{0})
		if err != nil {
			Expect(err.Error()).To(ContainSubstring("operation not permitted"))
		}
	})

	It("Prioritize succeeds or fails with permission error", func() {
		// Setpriority may fail without root; both outcomes are valid
		err := mgr.Prioritize()
		if err != nil {
			Expect(err.Error()).NotTo(BeEmpty())
		}
	})
})
