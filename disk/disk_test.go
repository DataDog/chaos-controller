// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package disk_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/DataDog/chaos-controller/disk"
)

var _ = Describe("InformerMock", func() {
	It("Major returns configured value via Return", func() {
		m := disk.NewInformerMock(GinkgoT())
		m.EXPECT().Major().Return(8)
		Expect(m.Major()).To(Equal(8))
	})

	It("Major uses RunAndReturn", func() {
		m := disk.NewInformerMock(GinkgoT())
		m.EXPECT().Major().RunAndReturn(func() int { return 42 })
		Expect(m.Major()).To(Equal(42))
	})

	It("Major uses Run callback", func() {
		m := disk.NewInformerMock(GinkgoT())
		called := false
		m.EXPECT().Major().Run(func() { called = true }).Return(1)
		m.Major()
		Expect(called).To(BeTrue())
	})

	It("Source returns configured value via Return", func() {
		m := disk.NewInformerMock(GinkgoT())
		m.EXPECT().Source().Return("/dev/sda")
		Expect(m.Source()).To(Equal("/dev/sda"))
	})

	It("Source uses RunAndReturn", func() {
		m := disk.NewInformerMock(GinkgoT())
		m.EXPECT().Source().RunAndReturn(func() string { return "/dev/nvme0n1" })
		Expect(m.Source()).To(Equal("/dev/nvme0n1"))
	})

	It("Source uses Run callback", func() {
		m := disk.NewInformerMock(GinkgoT())
		called := false
		m.EXPECT().Source().Run(func() { called = true }).Return("")
		m.Source()
		Expect(called).To(BeTrue())
	})
})

var _ = Describe("FromPath", func() {
	It("returns error for non-existent path", func() {
		_, err := disk.FromPath("/this/path/does/not/exist/chaos-controller-test")
		Expect(err).To(HaveOccurred())
	})

	It("attempts to inspect an existing path (df+ls coverage)", func() {
		// Either succeeds or fails depending on OS/environment.
		// The test only verifies the function doesn't panic.
		Expect(func() { disk.FromPath("/tmp") }).NotTo(Panic())
	})
})
