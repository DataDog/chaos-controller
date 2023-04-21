// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package stress_test

import (
	"github.com/DataDog/chaos-controller/cpuset"
	"github.com/DataDog/chaos-controller/mocks"
	. "github.com/DataDog/chaos-controller/stress"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zaptest"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("StresserManager Test", func() {
	var (
		stresserManager StresserManager
		cgroupManager   *mocks.CGroupManagerMock
	)

	BeforeEach(func() {
		stresserManager = NewCPUStresserManager(zaptest.NewLogger(GinkgoT()).Sugar())
	})

	When("IsCoreAlreadyStress", func() {
		Context("StresserManager is freshly initialized", func() {
			It("does not have any core stressed", func() {
				Expect(stresserManager.IsCoreAlreadyStressed(1)).To(BeFalse())
				Expect(stresserManager.StresserPIDs()).To(BeEmpty())
			})
		})

		Context("StresserManager previously tracked the same core", func() {
			It("should return true for core already stressed", func() {
				stresserManager.TrackCoreAlreadyStressed(0, 123)

				Expect(stresserManager.IsCoreAlreadyStressed(0)).To(BeTrue())
			})
		})
	})

	When("TrackInjectorCores", func() {
		BeforeEach(func() {
			cgroupManager = mocks.NewCGroupManagerMock(GinkgoT())
			cgroupManager.EXPECT().IsCgroupV2().Return(false)
			cgroupManager.EXPECT().Read("cpuset", "cpuset.effective_cpus").Return("0-1", nil)
		})

		Context("StresserManager is freshly initialized", func() {
			It("should track injector cores and return new cores to apply stress", func() {
				userRequestedCount := intstr.FromInt(2)
				cores, err := stresserManager.TrackInjectorCores(cgroupManager, &userRequestedCount)

				Expect(err).To(BeNil())
				Expect(cores).To(Equal(cpuset.NewCPUSet(0, 1)))
			})
		})

		Context("user request to target a specific number of cores", func() {
			It("should discount core according to user request", func() {
				userRequestedCount := intstr.FromInt(1)
				cores, err := stresserManager.TrackInjectorCores(cgroupManager, &userRequestedCount)

				Expect(err).To(BeNil())
				Expect(cores).To(Equal(cpuset.NewCPUSet(0)))
			})

			It("should discount core, percentage round up, according to user request", func() {
				userRequestedCount := intstr.FromString("30%")
				cores, err := stresserManager.TrackInjectorCores(cgroupManager, &userRequestedCount)

				Expect(err).To(BeNil())
				Expect(cores).To(Equal(cpuset.NewCPUSet(0)))
			})
		})
	})

	When("calling TrackCoreAlreadyStressed", func() {
		It("should record the stresser processID by coreID", func() {
			stresserManager.TrackCoreAlreadyStressed(0, 123)

			Expect(stresserManager.StresserPIDs()).To(HaveKeyWithValue(0, 123))
		})
	})
})
