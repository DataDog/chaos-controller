// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

//go:build linux
// +build linux

package netns_test

import (
	"fmt"
	"os"

	"github.com/DataDog/chaos-controller/env"
	"github.com/DataDog/chaos-controller/netns"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zaptest"
)

var _ = Describe("NewManager (Linux)", func() {
	It("returns error when target netns path does not exist", func() {
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		Expect(os.Setenv(env.InjectorMountProc, "/proc/")).To(Succeed())
		DeferCleanup(os.Unsetenv, env.InjectorMountProc)

		// PID 99999999 almost certainly doesn't exist
		_, err := netns.NewManager(log, 99999999)
		Expect(err).To(HaveOccurred())
	})

	It("creates manager for current process network namespace", func() {
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		Expect(os.Setenv(env.InjectorMountProc, "/proc/")).To(Succeed())
		DeferCleanup(os.Unsetenv, env.InjectorMountProc)

		pid := uint32(os.Getpid())
		// Check that the netns path exists first
		path := fmt.Sprintf("/proc/%d/ns/net", pid)
		if _, err := os.Stat(path); err != nil {
			Skip("netns path not accessible: " + err.Error())
		}

		mgr, err := netns.NewManager(log, pid)
		Expect(err).NotTo(HaveOccurred())
		Expect(mgr).NotTo(BeNil())
	})
})
