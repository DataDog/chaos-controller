// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package injector_test

import (
	"github.com/DataDog/chaos-controller/container"
	"github.com/DataDog/chaos-controller/process"
	"os"
	"syscall"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	. "github.com/DataDog/chaos-controller/injector"
)

var _ = Describe("Failure", func() {
	var (
		config  PodFailureInjectorConfig
		manager *process.ManagerMock
		proc    *os.Process
		ctn     *container.ContainerMock
		inj     Injector
		spec    v1beta1.PodFailureSpec
	)

	BeforeEach(func() {
		// process
		const PID = 1
		proc = &os.Process{Pid: PID}

		// container
		ctn = &container.ContainerMock{}
		ctn.On("PID").Return(uint32(PID))

		// manager
		manager = &process.ManagerMock{}
		manager.On("Find", mock.Anything).Return(proc, nil)
		manager.On("Signal", mock.Anything, mock.Anything).Return(nil)

		config = PodFailureInjectorConfig{
			Config: Config{
				Log:         log,
				MetricsSink: ms,
				Container:   ctn,
			},
			ProcessManager: manager,
		}

		spec = v1beta1.PodFailureSpec{}
	})

	JustBeforeEach(func() {
		inj = NewPodFailureInjector(spec, config)
	})

	Describe("injection", func() {
		JustBeforeEach(func() {
			Expect(inj.Inject()).To(BeNil())
		})

		Context("with kill enabled", func() {
			BeforeEach(func() {
				spec.Kill = true
			})

			It("should send the SIGKILL signal to the given process", func() {
				manager.AssertCalled(GinkgoT(), "Signal", proc, syscall.SIGKILL)
			})
		})

		Context("with kill disabled", func() {
			It("should send the SIGINT signal to the given process", func() {
				manager.AssertCalled(GinkgoT(), "Signal", proc, syscall.SIGINT)
			})
		})

	})
})
