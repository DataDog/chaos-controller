// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.
package injector_test

import (
	"errors"
	"strconv"

	"github.com/DataDog/chaos-controller/cgroup"
	"github.com/DataDog/chaos-controller/command"
	"github.com/DataDog/chaos-controller/container"
	"github.com/DataDog/chaos-controller/cpuset"
	. "github.com/DataDog/chaos-controller/injector"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var nothingToCancel = func() {}

var _ = Describe("CPU pressure", func() {
	var (
		config     Config
		cgroups    *cgroup.ManagerMock
		ctr        *container.ContainerMock
		factory    *InjectorCmdFactoryMock
		args       *CPUStressArgsBuilderMock
		background *command.BackgroundCmdMock
	)

	noCPUs := cpuset.CPUSet{}
	threeCPUs := cpuset.NewCPUSet(0, 1, 2)

	const containerName = "my-container-name"

	BeforeEach(func() {
		cgroups = cgroup.NewManagerMock(GinkgoT())
		ctr = container.NewContainerMock(GinkgoT())
		args = NewCPUStressArgsBuilderMock(GinkgoT())
		background = command.NewBackgroundCmdMock(GinkgoT())
		factory = NewInjectorCmdFactoryMock(GinkgoT())

		config = Config{
			Log:             log,
			Cgroup:          cgroups,
			TargetContainer: ctr,
		}
	})

	When("Inject is called", func() {
		DescribeTable("succeed with valid user requests",
			func(count string, stressExpected int, cpus cpuset.CPUSet) {
				inj := NewCPUPressureInjector(config, count, factory, args)

				seenArgs := []string{strconv.Itoa(stressExpected)}

				cgroups.EXPECT().ReadCPUSet().Return(cpus, nil).Maybe() // Only called when Int, let's be simple, externally we should not know
				ctr.EXPECT().Name().Return(containerName).Once()

				args.EXPECT().GenerateArgs(stressExpected).Return(seenArgs).Once()

				background.EXPECT().Start().Return(nil).Once()
				background.EXPECT().KeepAlive().Once()

				factory.EXPECT().NewInjectorBackgroundCmd(config.DisruptionDeadline, config.Disruption, containerName, seenArgs).Return(background, nothingToCancel, nil).Once()

				Expect(inj.Inject()).To(Succeed())
			},
			Entry("all the cores", "100%", 100, noCPUs),
			Entry("half the cores", "50%", 50, noCPUs),
			Entry("nothing", "0%", 0, noCPUs),
			Entry("too much", "1000%", 100, noCPUs),
			Entry("negative", "-1000%", 0, noCPUs),
			Entry("1 core out of 3", "1", 33, threeCPUs),
			Entry("3 core out of 3", "3", 100, threeCPUs),
			Entry("-1 core out of 3", "-1", 0, threeCPUs),
			Entry("6 core out of 3", "6", 100, threeCPUs),
		)

		Context("fails", func() {
			var inj Injector
			ExpectInjectError := func(expectedError string) {
				GinkgoHelper()

				Expect(inj.Inject()).Should(MatchError(expectedError))
			}

			It("when count is empty", func() {
				inj = NewCPUPressureInjector(config, "", factory, args)

				ExpectInjectError("unable to calculate stress percentage for '': invalid value for IntOrString: invalid type: string is not a percentage")
			})

			It("with cgroup manager error", func() {
				inj = NewCPUPressureInjector(config, "2", factory, args)
				cgroups.EXPECT().ReadCPUSet().Return(noCPUs, errors.New("cgroup manager error")).Once()

				ExpectInjectError("unable to read CPUSet for current container: cgroup manager error")
			})

			It("with background manager error", func() {
				inj = NewCPUPressureInjector(config, "100%", factory, args)

				ctr.EXPECT().Name().Return("").Once()
				args.EXPECT().GenerateArgs(100).Return(nil).Once()
				factory.EXPECT().NewInjectorBackgroundCmd(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil, errors.New("background manager error")).Once()

				ExpectInjectError("unable to create new process definition for injector: background manager error")
			})
		})
	})

	When("Clean is called", func() {
		It("succeed if no background process", func() {
			inj := NewCPUPressureInjector(config, "", factory, args)
			Expect(inj.Clean()).To(Succeed())
		})

		It("succeed and call stop after proper inject", func() {
			background.EXPECT().Start().Return(nil).Once()
			background.EXPECT().KeepAlive().Once()
			background.EXPECT().Stop().Return(nil).Once()

			inj := NewCPUPressureInjector(config, "100%", factory, args)

			ctr.EXPECT().Name().Return("").Once()
			args.EXPECT().GenerateArgs(100).Return(nil).Once()
			factory.EXPECT().NewInjectorBackgroundCmd(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(background, nothingToCancel, nil)

			Expect(inj.Inject()).To(Succeed()) // we need to first call inject to store the background process
			Expect(inj.Clean()).To(Succeed())
		})
	})
})
