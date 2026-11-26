// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector_test

import (
	"io/fs"
	"os"
	"path/filepath"

	chaosapi "github.com/DataDog/chaos-controller/api"
	"github.com/DataDog/chaos-controller/cgroup"
	"github.com/DataDog/chaos-controller/env"
	. "github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/o11y/tags"
	"github.com/DataDog/chaos-controller/process"
	"github.com/DataDog/chaos-controller/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

var _ = Describe("Node memory accounting", func() {
	var (
		config   Config
		cgroups  *cgroup.ManagerMock
		manager  *process.ManagerMock
		logs     *observer.ObservedLogs
		procRoot string
	)

	const processID = 42

	writeMeminfo := func(content string) {
		Expect(os.WriteFile(filepath.Join(procRoot, "meminfo"), []byte(content), 0o600)).To(Succeed())
	}

	BeforeEach(func() {
		procRoot = GinkgoT().TempDir()
		GinkgoT().Setenv(env.InjectorMountProc, procRoot)
		writeMeminfo("MemTotal: 1024 kB\nMemFree: 64 kB\nMemAvailable: 768 kB\n")
		cgroups = cgroup.NewManagerMock(GinkgoT())
		manager = process.NewManagerMock(GinkgoT())
		core, observed := observer.New(zap.InfoLevel)
		logs = observed
		config = Config{
			Log: zap.New(core).Sugar(), Cgroup: cgroups,
			Disruption: chaosapi.DisruptionArgs{Level: types.DisruptionLevelNode, DryRun: true},
		}
		manager.EXPECT().ProcessID().Return(processID).Once()
		cgroups.EXPECT().Join(processID).Return(nil).Once()
	})

	// Optional legacy reads let the regression fail on the returned baseline error,
	// rather than on an unexpected mock call. The fixed path must not use them.
	legacyCgroup := func(v2 bool, limit string, readErr error) {
		cgroups.EXPECT().IsCgroupV2().Return(v2).Maybe()
		if v2 {
			cgroups.EXPECT().Read("", "memory.max").Return(limit, readErr).Maybe()
		} else {
			cgroups.EXPECT().Read("memory", "memory.limit_in_bytes").Return(limit, readErr).Maybe()
		}
	}

	injectorForTest := func() Injector {
		inj := NewMemoryStressInjector(config, 75, 0, manager)
		DeferCleanup(inj.Clean)
		return inj
	}

	assertAllocationTarget := func() {
		entries := logs.FilterMessage("starting memory stress").All()
		Expect(entries).To(HaveLen(1))
		fields := entries[0].ContextMap()
		AddReportEntry("node memory target", fields)
		Expect(fields[tags.MemoryLimitKey]).To(Equal(int64(1024 * 1024)))
		Expect(fields[tags.CurrentUsageKey]).To(Equal(int64(256 * 1024)))
		Expect(fields[tags.TargetBytesKey]).To(Equal(int64(512 * 1024)))
	}

	DescribeTable("reads node memory independently of PID 1's cgroup",
		func(v2 bool, limit string, readErr error) {
			legacyCgroup(v2, limit, readErr)
			Expect(injectorForTest().Inject()).To(Succeed())
			assertAllocationTarget()
			cgroups.AssertNotCalled(GinkgoT(), "IsCgroupV2")
			cgroups.AssertNotCalled(GinkgoT(), "Read", mock.Anything, mock.Anything)
		},
		Entry("cgroup v2 root has no memory.max", true, "", fs.ErrNotExist),
		Entry("cgroup v2 PID 1 group is unlimited", true, "max", nil),
		Entry("cgroup v1 PID 1 group is unlimited", false, "9223372036854771712", nil),
	)

	It("does not allocate when node usage already exceeds the target", func() {
		legacyCgroup(true, "max", nil)
		writeMeminfo("MemTotal: 1024 kB\nMemFree: 64 kB\nMemAvailable: 128 kB\n")
		Expect(injectorForTest().Inject()).To(Succeed())
		Expect(logs.FilterMessage("starting memory stress").All()).To(BeEmpty())
		Expect(logs.FilterMessage("current memory usage already exceeds target, nothing to allocate").All()).To(HaveLen(1))
		cgroups.AssertNotCalled(GinkgoT(), "Read", mock.Anything, mock.Anything)
	})

	It("keeps pod-level memory accounting in the container cgroup", func() {
		config.Disruption.Level = types.DisruptionLevelPod
		Expect(os.Remove(filepath.Join(procRoot, "meminfo"))).To(Succeed())
		cgroups.EXPECT().IsCgroupV2().Return(true)
		cgroups.EXPECT().Read("", "memory.max").Return("1048576", nil).Once()
		cgroups.EXPECT().Read("", "memory.current").Return("262144", nil).Once()
		Expect(injectorForTest().Inject()).To(Succeed())
		assertAllocationTarget()
	})

	It("returns a node memory error when meminfo cannot be read", func() {
		legacyCgroup(true, "max", nil)
		Expect(os.Remove(filepath.Join(procRoot, "meminfo"))).To(Succeed())
		Expect(injectorForTest().Inject()).To(MatchError(ContainSubstring("unable to read node meminfo")))
	})

	DescribeTable("rejects unusable node memory information",
		func(content, expectedError string) {
			legacyCgroup(true, "max", nil)
			writeMeminfo(content)
			Expect(injectorForTest().Inject()).To(MatchError(ContainSubstring(expectedError)))
		},
		Entry("missing MemAvailable", "MemTotal: 1024 kB\n", "meminfo must include MemTotal and MemAvailable"),
		Entry("missing MemTotal", "MemAvailable: 768 kB\n", "meminfo must include MemTotal and MemAvailable"),
		Entry("non-numeric memory", "MemTotal: invalid kB\nMemAvailable: 768 kB\n", "unable to read node meminfo"),
		Entry("available exceeds total", "MemTotal: 1024 kB\nMemAvailable: 2048 kB\n", "invalid node memory statistics"),
	)
})
