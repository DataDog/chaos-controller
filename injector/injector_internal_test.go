// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

//go:build !linux
// +build !linux

package injector

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("munmapMemory (non-Linux stub)", func() {
	It("returns nil for any input", func() {
		err := munmapMemory([]byte("test"))
		Expect(err).NotTo(HaveOccurred())
	})

	It("returns nil for nil input", func() {
		Expect(munmapMemory(nil)).To(Succeed())
	})
})
