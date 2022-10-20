// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.
package injector_test

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/cgroup"
	"github.com/DataDog/chaos-controller/container"
	"github.com/DataDog/chaos-controller/cpuset"
	. "github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/process"
	"github.com/DataDog/chaos-controller/stress"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("Failure", func() {
	var (
		config          CPUPressureInjectorConfig
		cgroupManager   *cgroup.ManagerMock
		ctn             *container.ContainerMock
		stresser        *stress.StresserMock
		stresserExit    chan struct{}
		manager         *process.ManagerMock
		inj             Injector
		spec            v1beta1.CPUPressureSpec
		stresserManager *StresserManagerMock
	)

	BeforeEach(func() {
		// cgroup
		cgroupManager = &cgroup.ManagerMock{}
		cgroupManager.On("Join", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		// container
		ctn = &container.ContainerMock{}

		// stresser
		stresser = &stress.StresserMock{}
		stresser.On("Stress", mock.Anything).Return()

		// stresser exit chan, used to sync the stress goroutine with the test
		stresserExit = make(chan struct{})

		// manager
		manager = &process.ManagerMock{}
		manager.On("Prioritize").Return(nil)
		manager.On("ThreadID").Return(666)

		stresserManager = &StresserManagerMock{}
		stresserManager.On("TrackInjectorCores", mock.Anything).Return(cpuset.NewCPUSet(0, 1), nil)
		stresserManager.On("TrackCoreAlreadyStressed", mock.Anything, mock.Anything).Return(nil)
		stresserManager.On("StresserPIDs").Return(map[int]int{0: 666})
		stresserManager.On("IsCoreAlreadyStressed", 0).Return(true)
		stresserManager.On("IsCoreAlreadyStressed", 1).Return(false)

		//config
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

		var err error
		inj, err = NewCPUPressureInjector(spec, config)
		Expect(err).To(BeNil())

		// because the cleaning phase is blocking, we start it in a goroutine
		// and send a signal to the stresser exit handler
		Expect(inj.Inject()).To(BeNil())

		go func(inj Injector) {
			Expect(inj.Clean()).To(BeNil())
		}(inj)

		stresserExit <- struct{}{}
	})

	AfterEach(func() {
		manager.AssertExpectations(GinkgoT())
		stresser.AssertExpectations(GinkgoT())
		stresserManager.AssertExpectations(GinkgoT())
		cgroupManager.AssertExpectations(GinkgoT())
		ctn.AssertExpectations(GinkgoT())
	})

	Describe("injection", func() {

		It("should join the cpu and cpuset cgroups for the unstressed core", func() {
			cgroupManager.AssertCalled(GinkgoT(), "Join", "cpu", 666, false)
			cgroupManager.AssertCalled(GinkgoT(), "Join", "cpuset", 666, false)
			cgroupManager.AssertNumberOfCalls(GinkgoT(), "Join", 2)
		})

		It("should prioritize the current process", func() {
			manager.AssertCalled(GinkgoT(), "Prioritize")
		})

		It("should run the stress on one core", func() {
			stresser.AssertNumberOfCalls(GinkgoT(), "Stress", 1)
		})

		It("should record core and StresserPID in StresserManager", func() {
			stresserManager.AssertCalled(GinkgoT(), "TrackCoreAlreadyStressed", 1, 666)
		})

		It("should skip a target core that was already stress", func() {
			stresserManager.AssertNotCalled(GinkgoT(), "TrackCoreAlreadyStressed", 0, mock.Anything)
		})
	})
})
