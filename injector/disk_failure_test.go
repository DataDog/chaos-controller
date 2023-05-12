// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package injector_test

import (
	"os"

	"github.com/DataDog/chaos-controller/api"
	v1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/container"
	. "github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("Failure", func() {
	var (
		config      DiskFailureInjectorConfig
		level       types.DisruptionLevel
		proc        *os.Process
		inj         Injector
		spec        v1beta1.DiskFailureSpec
		commandMock *BPFDiskFailureCommandMock
		ctr         *container.ContainerMock
	)

	const PID = 1

	BeforeEach(func() {
		proc = &os.Process{Pid: PID}

		ctr = container.NewContainerMock(GinkgoT())

		commandMock = NewBPFDiskFailureCommandMock(GinkgoT())
		commandMock.EXPECT().Run(mock.Anything, mock.Anything).Return(nil)

		config = DiskFailureInjectorConfig{
			Config: Config{
				Log:         log,
				MetricsSink: ms,
				Disruption: api.DisruptionArgs{
					Level: level,
				},
				TargetContainer: ctr,
			},
			Cmd: commandMock,
		}

		spec = v1beta1.DiskFailureSpec{
			Path: "/",
		}
	})

	Describe("injection", func() {
		JustBeforeEach(func() {
			// instantiate lately so config can be updated in BeforeEach
			var err error
			inj, err = NewDiskFailureInjector(spec, config)

			Expect(err).ToNot(HaveOccurred())

			Expect(inj.Inject()).To(Succeed())
		})

		Context("with a pod level", func() {
			BeforeEach(func() {
				config.Disruption.Level = types.DisruptionLevelPod

				ctr.EXPECT().PID().Return(PID).Once()
			})

			It("should start the eBPF Disk failure program", func() {
				commandMock.AssertCalled(GinkgoT(), "Run", proc.Pid, "/")
			})
		})

		Context("with a node level", func() {
			BeforeEach(func() {
				config.Disruption.Level = types.DisruptionLevelNode
			})

			It("should start the eBPF Disk failure program", func() {
				ctr.AssertNumberOfCalls(GinkgoT(), "PID", 0)
				commandMock.AssertCalled(GinkgoT(), "Run", 0, "/")
			})
		})
	})
})
