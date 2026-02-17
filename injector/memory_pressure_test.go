// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector_test

import (
	"errors"
	"time"

	"github.com/DataDog/chaos-controller/cgroup"
	"github.com/DataDog/chaos-controller/command"
	"github.com/DataDog/chaos-controller/container"
	. "github.com/DataDog/chaos-controller/injector"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("Memory pressure", func() {
	var (
		config     Config
		cgroups    *cgroup.ManagerMock
		ctr        *container.ContainerMock
		factory    *InjectorCmdFactoryMock
		args       *MemoryStressArgsBuilderMock
		background *command.BackgroundCmdMock
	)

	const containerName = "my-container-name"

	BeforeEach(func() {
		cgroups = cgroup.NewManagerMock(GinkgoT())
		ctr = container.NewContainerMock(GinkgoT())
		args = NewMemoryStressArgsBuilderMock(GinkgoT())
		background = command.NewBackgroundCmdMock(GinkgoT())
		factory = NewInjectorCmdFactoryMock(GinkgoT())

		config = Config{
			Log:             log,
			Cgroup:          cgroups,
			TargetContainer: ctr,
		}
	})

	When("Inject is called", func() {
		It("succeeds with valid percentage", func() {
			inj := NewMemoryPressureInjector(config, "76%", time.Duration(0), factory, args)

			seenArgs := []string{"76"}

			ctr.EXPECT().Name().Return(containerName).Once()
			args.EXPECT().GenerateArgs(76, time.Duration(0)).Return(seenArgs).Once()

			background.EXPECT().Start().Return(nil).Once()
			background.EXPECT().KeepAlive().Once()

			factory.EXPECT().NewInjectorBackgroundCmd(config.DisruptionDeadline, config.Disruption, containerName, seenArgs).Return(background, nothingToCancel, nil).Once()

			Expect(inj.Inject()).To(Succeed())
		})

		It("fails with invalid percentage", func() {
			inj := NewMemoryPressureInjector(config, "abc", time.Duration(0), factory, args)

			Expect(inj.Inject()).Should(MatchError(ContainSubstring("unable to parse target percent")))
		})

		It("fails with background manager error", func() {
			inj := NewMemoryPressureInjector(config, "50%", time.Duration(0), factory, args)

			ctr.EXPECT().Name().Return("").Once()
			args.EXPECT().GenerateArgs(50, time.Duration(0)).Return(nil).Once()
			factory.EXPECT().NewInjectorBackgroundCmd(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil, errors.New("background manager error")).Once()

			Expect(inj.Inject()).Should(MatchError("unable to create new process definition for injector: background manager error"))
		})
	})

	When("Clean is called", func() {
		It("succeeds if no background process", func() {
			inj := NewMemoryPressureInjector(config, "", time.Duration(0), factory, args)
			Expect(inj.Clean()).To(Succeed())
		})

		It("succeeds and calls stop after proper inject", func() {
			background.EXPECT().Start().Return(nil).Once()
			background.EXPECT().KeepAlive().Once()
			background.EXPECT().Stop().Return(nil).Once()

			inj := NewMemoryPressureInjector(config, "50%", time.Duration(0), factory, args)

			ctr.EXPECT().Name().Return("").Once()
			args.EXPECT().GenerateArgs(50, time.Duration(0)).Return(nil).Once()
			factory.EXPECT().NewInjectorBackgroundCmd(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(background, nothingToCancel, nil)

			Expect(inj.Inject()).To(Succeed())
			Expect(inj.Clean()).To(Succeed())
		})
	})
})
