// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package injector_test

import (
	"os"
	"syscall"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/container"
	. "github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/process"
)

var _ = Describe("Failure", func() {
	var (
		config  ContainerFailureInjectorConfig
		manager *process.ManagerMock
		proc    *os.Process
		ctn     *container.ContainerMock
		inj     Injector
		spec    v1beta1.ContainerFailureSpec
	)

	BeforeEach(func() {
		// process
		const PID = 1
		proc = &os.Process{Pid: PID}

		// container
		ctn = container.NewContainerMock(GinkgoT())
		ctn.EXPECT().PID().Return(PID)

		// manager
		manager = process.NewManagerMock(GinkgoT())
		manager.EXPECT().Find(mock.Anything).Return(proc, nil)
		manager.EXPECT().Signal(mock.Anything, mock.Anything).Return(nil)

		config = ContainerFailureInjectorConfig{
			Config: Config{
				Log:             log,
				MetricsSink:     ms,
				TargetContainer: ctn,
			},
			ProcessManager: manager,
		}

		spec = v1beta1.ContainerFailureSpec{}
	})

	JustBeforeEach(func() {
		inj = NewContainerFailureInjector(spec, config)
	})

	Describe("injection", func() {
		JustBeforeEach(func() {
			Expect(inj.Inject()).To(BeNil())
		})

		Context("with forced enabled", func() {
			BeforeEach(func() {
				spec.Forced = true
			})

			It("should send the SIGKILL signal to the given process", func() {
				manager.AssertCalled(GinkgoT(), "Signal", proc, syscall.SIGKILL)
			})
		})

		Context("with forced disabled", func() {
			It("should send the SIGTERM signal to the given process", func() {
				manager.AssertCalled(GinkgoT(), "Signal", proc, syscall.SIGTERM)
			})
		})
	})
})
