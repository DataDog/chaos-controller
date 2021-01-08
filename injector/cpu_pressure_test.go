// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.
package injector_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/cgroup"
	"github.com/DataDog/chaos-controller/container"
	. "github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/process"
	"github.com/DataDog/chaos-controller/stress"
)

var _ = Describe("Failure", func() {
	var (
		config        CPUPressureInjectorConfig
		cgroupManager *cgroup.ManagerMock
		ctn           *container.ContainerMock
		stresser      *stress.StresserMock
		stresserExit  chan struct{}
		manager       *process.ManagerMock
		inj           Injector
		spec          v1beta1.CPUPressureSpec
	)

	BeforeEach(func() {
		// cgroup
		cgroupManager = &cgroup.ManagerMock{}
		cgroupManager.On("Join", mock.Anything, mock.Anything).Return(nil)

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

		//config
		config = CPUPressureInjectorConfig{
			Config: Config{
				Cgroup:      cgroupManager,
				Container:   ctn,
				Log:         log,
				MetricsSink: ms,
			},
			Stresser:       stresser,
			StresserExit:   stresserExit,
			ProcessManager: manager,
		}

		// spec
		spec = v1beta1.CPUPressureSpec{}
	})

	JustBeforeEach(func() {
		inj = NewCPUPressureInjector(spec, config)
	})

	Describe("injection", func() {
		JustBeforeEach(func() {
			// because the cleaning phase is blocking, we start it in a goroutine
			// and send a signal to the stresser exit handler
			Expect(inj.Inject()).To(BeNil())

			go func() {
				Expect(inj.Clean()).To(BeNil())
			}()

			stresserExit <- struct{}{}
		})

		It("should join the CPU cgroup", func() {
			cgroupManager.AssertCalled(GinkgoT(), "Join", "cpu", mock.Anything)
		})

		It("should prioritize the current process", func() {
			manager.AssertCalled(GinkgoT(), "Prioritize")
		})

		It("should run the stress routines", func() {
			stresser.AssertCalled(GinkgoT(), "Stress")
		})
	})
})
