// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.
package injector_test

import (
	"os"
	"syscall"

	. "github.com/onsi/ginkgo"
	"github.com/stretchr/testify/mock"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/container"
	. "github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/process"
	"github.com/DataDog/chaos-controller/stress"
)

var _ = Describe("Failure", func() {
	var (
		config       CPUPressureInjectorConfig
		cgroup       *container.CgroupMock
		ctn          *container.ContainerMock
		stresser     *stress.StresserMock
		stresserExit chan struct{}
		manager      *process.ManagerMock
		sigHandler   chan os.Signal
		inj          Injector
		spec         v1beta1.CPUPressureSpec
	)

	BeforeEach(func() {
		// cgroup
		cgroup = &container.CgroupMock{}
		cgroup.On("JoinCPU").Return(nil)

		// container
		ctn = &container.ContainerMock{}
		ctn.On("Cgroup").Return(cgroup)

		// stresser
		stresser = &stress.StresserMock{}
		stresser.On("Stress", mock.Anything).Return()

		// stresser exit chan, used to sync the stress goroutine with the test
		stresserExit = make(chan struct{})

		// manager
		manager = &process.ManagerMock{}
		manager.On("Prioritize").Return(nil)

		// signal handler
		sigHandler = make(chan os.Signal)

		//config
		config = CPUPressureInjectorConfig{
			Stresser:       stresser,
			StresserExit:   stresserExit,
			ProcessManager: manager,
			SignalHandler:  sigHandler,
		}

		// spec
		spec = v1beta1.CPUPressureSpec{}
	})

	JustBeforeEach(func() {
		inj = NewCPUPressureInjectorWithConfig("fake", spec, ctn, log, ms, config)
	})

	Describe("injection", func() {
		JustBeforeEach(func() {
			// because the injection is blocking, we start it in a goroutine
			// and send a fake sigterm signal to the signal handler
			// to trigger the end of the injection
			// we also send an event on the stresser exit chan to sync the stress call
			go inj.Inject()
			stresserExit <- struct{}{}
			sigHandler <- syscall.SIGTERM
		})

		It("should join the container CPU cgroup", func() {
			cgroup.AssertCalled(GinkgoT(), "JoinCPU")
		})

		It("should prioritize the current process", func() {
			manager.AssertCalled(GinkgoT(), "Prioritize")
		})

		It("should run the stress routines", func() {
			stresser.AssertCalled(GinkgoT(), "Stress")
		})
	})
})
