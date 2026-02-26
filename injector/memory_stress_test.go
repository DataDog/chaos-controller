// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector_test

import (
	"fmt"
	"time"

	"github.com/DataDog/chaos-controller/cgroup"
	. "github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/process"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

	Describe("Clean", func() {
		It("succeeds when no injection has occurred", func() {
			Expect(inj.Clean()).To(Succeed())
		})
	})
})
