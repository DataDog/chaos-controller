// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

//go:build linux

package netns_test

import (
	"os"
	"path/filepath"

	"github.com/DataDog/chaos-controller/env"
	"github.com/DataDog/chaos-controller/netns"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

var _ = Describe("failed namespace manager construction", func() {
	It("closes the root namespace handle when the target lookup fails", func() {
		GinkgoT().Setenv(env.InjectorMountProc, GinkgoT().TempDir()+"/")
		namespace, err := os.Readlink("/proc/thread-self/ns/net")
		Expect(err).ToNot(HaveOccurred())
		countHandles := func() int {
			entries, err := os.ReadDir("/proc/self/fd")
			Expect(err).ToNot(HaveOccurred())
			count := 0
			for _, entry := range entries {
				target, err := os.Readlink(filepath.Join("/proc/self/fd", entry.Name()))
				if err == nil && target == namespace {
					count++
				}
			}
			return count
		}
		before := countHandles()

		for range 5 {
			manager, err := netns.NewManager(zap.NewNop().Sugar(), 1)
			Expect(err).To(HaveOccurred())
			Expect(manager).To(BeNil())
		}

		Expect(countHandles()).To(Equal(before), "failed construction must not retain namespace descriptors")
	})
})
