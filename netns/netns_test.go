// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package netns_test

import (
	"os"

	"github.com/DataDog/chaos-controller/env"
	"github.com/DataDog/chaos-controller/netns"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zaptest"
)

var _ = Describe("NewManager", func() {
	It("returns error when InjectorMountProc env var not set", func() {
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		_, err := netns.NewManager(log, 1)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("CHAOS_INJECTOR_MOUNT_PROC"))
	})

	It("returns error when InjectorMountProc is set but netns unavailable", func() {
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		Expect(os.Setenv(env.InjectorMountProc, "/nonexistent/proc/")).To(Succeed())
		DeferCleanup(os.Unsetenv, env.InjectorMountProc)

		_, err := netns.NewManager(log, 1)
		Expect(err).To(HaveOccurred())
	})
})
