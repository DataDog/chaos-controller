// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package injector_test

import (
	v1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/container"
	. "github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/process"
	"github.com/DataDog/chaos-controller/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"os"
	"time"
)

var _ = Describe("Failure", func() {
	var (
		config  DiskFailureInjectorConfig
		level   types.DisruptionLevel
		manager *process.ManagerMock
		cmdMock BPFDiskFailureCommandMock
		proc    *os.Process
		ctn     *container.ContainerMock
		inj     Injector
		spec    v1beta1.DiskFailureSpec
	)

	JustBeforeEach(func() {
		const PID = 1
		proc = &os.Process{Pid: PID}

		// container
		ctn = &container.ContainerMock{}
		ctn.On("PID").Return(uint32(PID))

		// manager
		manager = &process.ManagerMock{}
		manager.On("Find", mock.Anything).Return(proc, nil)
		manager.On("Signal", mock.Anything, mock.Anything).Return(nil)

		// BPF Disk failure command
		cmdMock = BPFDiskFailureCommandMock{}
		cmdMock.On("Run", mock.Anything, mock.Anything).Return(nil)
		cmdMock.On("GetProcess").Return(proc)

		config = DiskFailureInjectorConfig{
			Config: Config{
				Log:             log,
				MetricsSink:     ms,
				Level:           level,
				TargetContainer: ctn,
			},
			Process: nil, Cmd: &cmdMock,
		}

		spec = v1beta1.DiskFailureSpec{
			Path: "/",
		}

		var err error
		inj, err = NewDiskFailureInjector(spec, config)

		Expect(err).To(BeNil())
	})

	Describe("injection", func() {
		JustBeforeEach(func() {
			Expect(inj.Inject()).To(BeNil())
			time.Sleep(time.Second * 1)
		})

		Context("with a pod level", func() {
			BeforeEach(func() {
				level = types.DisruptionLevelPod
			})

			It("should start the eBPF Disk failure program", func() {
				cmdMock.AssertCalled(GinkgoT(), "Run", proc.Pid, "/")
			})

			It("should get the pid of the ebpf program", func() {
				cmdMock.AssertCalled(GinkgoT(), "GetProcess")
			})
		})

		Context("with a node level", func() {
			BeforeEach(func() {
				level = types.DisruptionLevelNode
			})

			It("should start the eBPF Disk failure program", func() {
				cmdMock.AssertCalled(GinkgoT(), "Run", 0, "/")
			})

			It("should get the pid of the ebpf program", func() {
				cmdMock.AssertCalled(GinkgoT(), "GetProcess")
			})
		})
	})
})
