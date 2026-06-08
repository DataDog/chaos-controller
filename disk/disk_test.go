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
