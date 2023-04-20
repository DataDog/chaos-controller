// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.
package injector_test

import (
	"time"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/cpuset"
	. "github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("Failure", func() {
	var (
		config          CPUPressureInjectorConfig
		cgroupManager   *mocks.CGroupManagerMock
		ctn             *mocks.ContainerMock
		stresser        *mocks.StresserMock
		stresserExit    chan struct{}
		manager         *mocks.ProcessManagerMock
		inj             Injector
		spec            v1beta1.CPUPressureSpec
		stresserManager *mocks.StresserManagerMock
	)

	BeforeEach(func() {
		// cgroup
		cgroupManager = mocks.NewCGroupManagerMock(GinkgoT())
		cgroupManager.EXPECT().Join(mock.Anything).Return(nil)

		// container
		ctn = mocks.NewContainerMock(GinkgoT())

		// stresser
		stresser = mocks.NewStresserMock(GinkgoT())
		stresser.EXPECT().Stress(mock.Anything).Return()

		// stresser exit chan, used to sync the stress goroutine with the test
		stresserExit = make(chan struct{}, 1)

		// manager
		manager = mocks.NewProcessManagerMock(GinkgoT())
		manager.EXPECT().Prioritize().Return(nil)
		manager.EXPECT().ThreadID().Return(666)
		manager.EXPECT().ProcessID().Return(42)

		stresserManager = mocks.NewStresserManagerMock(GinkgoT())
		stresserManager.EXPECT().TrackCoreAlreadyStressed(mock.Anything, mock.Anything)
		stresserManager.EXPECT().StresserPIDs().Return(map[int]int{0: 666})
		stresserManager.EXPECT().IsCoreAlreadyStressed(0).Return(true)
		stresserManager.EXPECT().IsCoreAlreadyStressed(1).Return(false)

		// config
		config = CPUPressureInjectorConfig{
			Config: Config{
				Cgroup:          cgroupManager,
				TargetContainer: ctn,
				Log:             log,
				MetricsSink:     ms,
			},
			Stresser:        stresser,
			StresserExit:    stresserExit,
			ProcessManager:  manager,
			StresserManager: stresserManager,
		}

		// spec
		spec = v1beta1.CPUPressureSpec{}
	})

	JustBeforeEach(func() {
		var err error
		inj, err = NewCPUPressureInjector(spec, config)
		Expect(err).ShouldNot(HaveOccurred())

		// because the cleaning phase is blocking, we start it in a goroutine
		// and send a signal to the stresser exit handler
		Expect(inj.Inject()).Should(Succeed())

		go func(inj Injector) {
			Expect(inj.Clean()).Should(Succeed())
		}(inj)

		stresserExit <- struct{}{}
	})

	Describe("injection", func() {
		Context("user request to stress all the cores", func() {
			BeforeEach(func() {
				stresserManager.EXPECT().TrackInjectorCores(mock.Anything, mock.Anything).Return(cpuset.NewCPUSet(0, 1), nil)
			})

			It("should call the expected functions and args", func() {
				By("should join target cgroup subsystems from the main process", func() {
					cgroupManager.AssertCalled(GinkgoT(), "Join", 42)
				})

				By("should prioritize the current process", func() {
					manager.AssertCalled(GinkgoT(), "Prioritize")
				})

				By("should run the stress on one core", func() {
					// The Stress happens async, so we need to give it time to guarantee. This sleep will be unnecessary within a month when we have updated cpu_pressure's approach
					time.Sleep(time.Second * 2)
					stresser.AssertNumberOfCalls(GinkgoT(), "Stress", 1)
				})

				By("should record core and StresserPID in StresserManager", func() {
					stresserManager.AssertCalled(GinkgoT(), "TrackCoreAlreadyStressed", 1, 666)
				})

				By("should skip a target core that was already stress", func() {
					stresserManager.AssertNotCalled(GinkgoT(), "TrackCoreAlreadyStressed", 0, mock.Anything)
				})
			})
		})

		Context("user request to stress half of the cores", func() {
			BeforeEach(func() {
				userRequestCount := intstr.FromString("50%")
				spec = v1beta1.CPUPressureSpec{
					Count: &userRequestCount,
				}
				stresserManager.EXPECT().TrackInjectorCores(mock.Anything, &userRequestCount).Return(cpuset.NewCPUSet(0, 1), nil)
			})

			It("should call stresserManager track cores and get new core to apply pressure", func() {
				// left empty as AfterEach 'AssertExpectations' check all this tests expectations
				// TODO what AfterEach was this referring to? Is there an implicit one I don't know about?
			})
		})
	})
})
