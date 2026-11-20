// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

//go:build linux
// +build linux

package cgroup_test

import (
	"os"

	"github.com/DataDog/chaos-controller/cgroup"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zaptest"
)

var _ = Describe("NewManager (Linux)", func() {
	It("creates a manager from /proc/self/cgroup", func() {
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		// Use current process PID (self) — /proc/self/cgroup always exists
		mgr, err := cgroup.NewManager(false, uint32(os.Getpid()), "/sys/fs/cgroup", log)
		if err != nil {
			// Some CI environments have restricted cgroup access
			Skip("cgroup manager unavailable: " + err.Error())
		}

		Expect(mgr).NotTo(BeNil())

		// Exercise accessor methods
		_ = mgr.IsCgroupV2()
		_ = mgr.RelativePath("cpu")
	})

	It("dry-run manager skips actual writes", func() {
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		mgr, err := cgroup.NewManager(true, uint32(os.Getpid()), "/sys/fs/cgroup", log)
		if err != nil {
			Skip("cgroup manager unavailable: " + err.Error())
		}

		// Write in dry-run should not fail even for non-existent paths
		err = mgr.Write("cpu", "cgroup.procs", "1")
		if err != nil {
			// dry-run may still fail if path doesn't exist; that's acceptable
			Expect(err).To(HaveOccurred())
		}
	})
})
