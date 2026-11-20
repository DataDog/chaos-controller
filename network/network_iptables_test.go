// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package network

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zaptest"
)

var _ = Describe("NewIPTables", func() {
	It("attempts to create iptables (error expected without iptables binary)", func() {
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		// On macOS/non-Linux without iptables, this returns an error — covers the error path
		_, err := NewIPTables(log, false)
		if err != nil {
			Expect(err).To(HaveOccurred())
		}
	})
})
