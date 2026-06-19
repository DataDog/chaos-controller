// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package disk

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("disk struct", func() {
	It("Major returns the stored major number", func() {
		d := disk{major: 8, source: "/dev/sda"}
		Expect(d.Major()).To(Equal(8))
	})

	It("Source returns the stored source path", func() {
		d := disk{major: 8, source: "/dev/sda"}
		Expect(d.Source()).To(Equal("/dev/sda"))
	})
})

var _ = Describe("FromPath", func() {
	It("returns error for non-existent path", func() {
		_, err := FromPath("/this/path/does/not/exist/chaos-controller-test")
		Expect(err).To(HaveOccurred())
	})
})
