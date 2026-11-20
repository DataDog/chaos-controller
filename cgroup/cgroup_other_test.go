// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

//go:build !linux
// +build !linux

package cgroup_test

import (
	"github.com/DataDog/chaos-controller/cgroup"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zaptest"
)

var _ = Describe("NewManager (non-Linux)", func() {
	It("returns error (not implemented on non-Linux)", func() {
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		_, err := cgroup.NewManager(false, 1, "/sys/fs/cgroup", log)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("not implemented"))
	})
})
