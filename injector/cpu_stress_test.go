// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.
package injector_test

import (
	"fmt"

	"github.com/DataDog/chaos-controller/cgroup"
	"github.com/DataDog/chaos-controller/cpuset"
	. "github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/process"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CPU stress", func() {
	var (
		config  Config
		inj     Injector
		cgroups *cgroup.ManagerMock
		manager *process.ManagerMock
		runtime *process.RuntimeMock
	)

	cpu1, cpu2, cpu3 := 0, 1, 2
	threeCPUs := cpuset.NewCPUSet(cpu1, cpu2, cpu3)

	BeforeEach(func() {
		manager = process.NewManagerMock(GinkgoT())
		runtime = process.NewRuntimeMock(GinkgoT())
		cgroups = cgroup.NewManagerMock(GinkgoT())

		config = Config{
			Log:    log,
			Cgroup: cgroups,
		}
	})

	JustBeforeEach(func() {
		inj = NewCPUStressInjector(config, 10, manager, runtime)
	})

	Specify("invalid CPUSet returns error", func() {
		cgroups.EXPECT().ReadCPUSet().Return(threeCPUs, nil).Return(cpuset.CPUSet{}, fmt.Errorf("error from CPUSet")).Once()

		Expect(inj.Inject()).To(MatchError("unable to read CPUSet: error from CPUSet"))
	})

	Describe("valid CPUSet", func() {
		processID := 42

		BeforeEach(func() {
			cgroups.EXPECT().ReadCPUSet().Return(threeCPUs, nil).Once()
			manager.EXPECT().ProcessID().Return(processID).Once()
		})

		Specify("invalid CGroup.Join returns error", func() {
			cgroups.EXPECT().Join(processID).Return(fmt.Errorf("error from Join")).Once()

			Expect(inj.Inject()).To(MatchError("unable to join cgroup for process '42': error from Join"))
		})

		Describe("valid CGroup.Join", func() {
			BeforeEach(func() {
				cgroups.EXPECT().Join(processID).Return(nil).Once()
			})

			Specify("invalid prioritization returns error", func() {
				manager.EXPECT().Prioritize().Return(fmt.Errorf("error from Prioritize")).Once()

				Expect(inj.Inject()).To(MatchError("unable to prioritize process: error from Prioritize"))
			})

			Describe("valid prioritization", func() {
				cpuCount := threeCPUs.Size()

				BeforeEach(func() {
					By("Setting up expected global calls")
					manager.EXPECT().Prioritize().Return(nil).Once()
					runtime.EXPECT().GOMAXPROCS(cpuCount).Return(0).Once()

					By("Setting expected calls per goroutines")
					runtime.EXPECT().LockOSThread().Times(cpuCount)

					cpus := threeCPUs.ToSlice()
					for i := 0; i < cpuCount; i++ {
						manager.EXPECT().ThreadID().Return(1240 + i).Once() // threadID does not matter, however let's keep it realistic and not the same
						manager.EXPECT().SetAffinity([]int{cpus[i]}).Return(nil).Once()
					}
				})

				Specify("standard flow", func() {
					runtime.EXPECT().UnlockOSThread().Times(cpuCount)

					By("succeeding to inject")
					Expect(inj.Inject()).To(Succeed())

					By("failing to inject when called twice without cleaning")
					Expect(inj.Inject()).To(MatchError("Injector contains 3 unexited stresses, all stresses should be clean before re-injecting"))

					By("succeeding to clean")
					Expect(inj.Clean()).To(Succeed())
				})
			})
		})
	})
})
