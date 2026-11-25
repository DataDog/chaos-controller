// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector_test

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/DataDog/chaos-controller/cgroup"
	. "github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/o11y/tags"
	"github.com/DataDog/chaos-controller/process"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

var _ = Describe("Memory stress", func() {
	var (
		config  Config
		inj     Injector
		cgroups *cgroup.ManagerMock
		manager *process.ManagerMock
	)

	const processID = 42

	BeforeEach(func() {
		cgroups = cgroup.NewManagerMock(GinkgoT())
		manager = process.NewManagerMock(GinkgoT())

		config = Config{
			Log:    log,
			Cgroup: cgroups,
		}
	})

	JustBeforeEach(func() {
		inj = NewMemoryStressInjector(config, 80, time.Duration(0), manager)
	})

	Describe("Inject", func() {
		Context("when cgroup Join fails", func() {
			BeforeEach(func() {
				cgroups.EXPECT().Join(processID).Return(fmt.Errorf("join error")).Once()
				manager.EXPECT().ProcessID().Return(processID).Once()
			})

			It("returns an error", func() {
				Expect(inj.Inject()).To(MatchError("unable to join cgroup for process '42': join error"))
			})
		})

		Describe("cgroupv2", func() {
			BeforeEach(func() {
				cgroups.EXPECT().Join(processID).Return(nil).Once()
				manager.EXPECT().ProcessID().Return(processID).Once()
				cgroups.EXPECT().IsCgroupV2().Return(true)
			})

			Context("when memory.max read fails", func() {
				BeforeEach(func() {
					cgroups.EXPECT().Read("", "memory.max").Return("", fmt.Errorf("read error")).Once()
				})

				It("returns an error", func() {
					Expect(inj.Inject()).To(MatchError(ContainSubstring("unable to read memory limit")))
				})
			})

			Context("when memory.max is 'max' (unlimited)", func() {
				BeforeEach(func() {
					cgroups.EXPECT().Read("", "memory.max").Return("max", nil).Once()
				})

				It("returns an error containing 'unlimited'", func() {
					Expect(inj.Inject()).To(MatchError(ContainSubstring("unlimited")))
				})
			})
		})

		Describe("cgroupv1", func() {
			BeforeEach(func() {
				cgroups.EXPECT().Join(processID).Return(nil).Once()
				manager.EXPECT().ProcessID().Return(processID).Once()
				cgroups.EXPECT().IsCgroupV2().Return(false)
			})

			Context("when memory.limit_in_bytes read fails", func() {
				BeforeEach(func() {
					cgroups.EXPECT().Read("memory", "memory.limit_in_bytes").Return("", fmt.Errorf("read error")).Once()
				})

				It("returns an error", func() {
					Expect(inj.Inject()).To(MatchError(ContainSubstring("unable to read memory limit")))
				})
			})

			Context("when memory.limit_in_bytes is the unlimited sentinel (9223372036854771712)", func() {
				BeforeEach(func() {
					cgroups.EXPECT().Read("memory", "memory.limit_in_bytes").Return("9223372036854771712", nil).Once()
				})

				It("returns an error containing 'unlimited'", func() {
					Expect(inj.Inject()).To(MatchError(ContainSubstring("unlimited")))
				})
			})

			Context("when memory.limit_in_bytes is set to a real limit", func() {
				const memLimit = "536870912" // 512 MiB
				const memUsage = "104857600" // 100 MiB

				BeforeEach(func() {
					cgroups.EXPECT().Read("memory", "memory.limit_in_bytes").Return(memLimit, nil).Once()
					cgroups.EXPECT().IsCgroupV2().Return(false).Maybe()
					cgroups.EXPECT().Read("memory", "memory.usage_in_bytes").Return(memUsage, nil).Once()
				})

				It("succeeds and starts allocating memory", func() {
					Expect(inj.Inject()).To(Succeed())

					// clean up background goroutine
					Expect(inj.Clean()).To(Succeed())
				})
			})
		})
	})

	pageSize := os.Getpagesize()

	DescribeTable("page-aligned memory allocation plan",
		func(targetBytes int, rampDuration time.Duration, expectedSteps, expectedChunkSize int) {
			core, logs := observer.New(zap.InfoLevel)
			config.Log = zap.New(core).Sugar()

			manager.EXPECT().ProcessID().Return(processID).Once()
			cgroups.EXPECT().Join(processID).Return(nil).Once()
			cgroups.EXPECT().IsCgroupV2().Return(true)
			cgroups.EXPECT().Read("", "memory.max").Return(strconv.Itoa(targetBytes), nil).Once()
			cgroups.EXPECT().Read("", "memory.current").Return("0", nil).Once()

			inj = NewMemoryStressInjector(config, 100, rampDuration, manager)
			DeferCleanup(inj.Clean)
			Expect(inj.Inject()).To(Succeed())

			plans := logs.FilterMessage("memory allocation plan").All()
			Expect(plans).To(HaveLen(1))
			plan := plans[0].ContextMap()
			AddReportEntry("memory allocation plan", plan)
			Expect(plan[tags.StepsKey]).To(Equal(int64(expectedSteps)))
			Expect(plan[tags.ChunkSizeKey]).To(Equal(int64(expectedChunkSize)))
		},
		Entry("caps steps to target pages", 2*pageSize, 10*time.Second, 2, pageSize),
		Entry("rounds per-step chunks up", 5*pageSize, 2*time.Second, 2, 3*pageSize),
		Entry("handles targets below one page", 1, 10*time.Second, 1, pageSize),
		Entry("rounds immediate allocation up", pageSize+1, time.Duration(0), 1, 2*pageSize),
		Entry("keeps page-aligned allocations", 2*pageSize, 2*time.Second, 2, pageSize),
	)

	Describe("Clean", func() {
		It("succeeds when no injection has occurred", func() {
			Expect(inj.Clean()).To(Succeed())
		})
	})
})
