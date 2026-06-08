// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

//go:build !linux
// +build !linux

package process_test

import (
	"github.com/DataDog/chaos-controller/process"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manager stubs (non-Linux)", func() {
	var mgr process.Manager

	BeforeEach(func() {
		mgr = process.NewManager(false)
	})

	It("Prioritize returns unsupported error", func() {
		err := mgr.Prioritize()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("unsupported"))
	})

	It("ThreadID returns -1", func() {
		Expect(mgr.ThreadID()).To(Equal(-1))
	})

	It("ProcessID returns -1", func() {
		Expect(mgr.ProcessID()).To(Equal(-1))
	})

	It("SetAffinity returns unsupported error", func() {
		err := mgr.SetAffinity([]int{0})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("unsupported"))
	})
})
