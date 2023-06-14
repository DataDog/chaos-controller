// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.
package injector_test

import (
	context "context"
	"strconv"
	"time"

	"github.com/DataDog/chaos-controller/api"
	"github.com/DataDog/chaos-controller/command"
	"github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/process"
	"github.com/DataDog/chaos-controller/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("InjectorCmdFactory Create", func() {
	var (
		disruptionArgs api.DisruptionArgs
		args           []string
		target         string
		deadline       time.Time
		factory        *command.FactoryMock
		manager        *process.ManagerMock
		sut            injector.InjectorCmdFactory
	)

	const (
		deadlineDuration = 100 * time.Millisecond
	)

	BeforeEach(func() {
		disruptionArgs = api.DisruptionArgs{}
		target = ""
		args = []string{}
		deadline = time.Now().Add(deadlineDuration)

		factory = command.NewFactoryMock(GinkgoT())
		manager = process.NewManagerMock(GinkgoT())

		sut = injector.NewInjectorCmdFactory(log, manager, factory)
	})

	Describe("pod level", func() {
		BeforeEach(func() {
			// Default disruption level is pod
			// Here we are not going through the unmarshalling layer hence we set explicitely
			disruptionArgs.Level = types.DisruptionLevelPod
			target = "container-name"
		})

		Specify("fails if targeted container does not exists in disruption args", func() {
			Expect(sut.NewInjectorBackgroundCmd(deadline, disruptionArgs, target, args)).Error().To(MatchError("targeted container does not exists: container-name"))
		})

		Describe("with several containers", func() {
			processID := 42

			BeforeEach(func() {
				disruptionArgs.TargetContainers = map[string]string{
					target:                 "container-id",
					"other-container-name": "other-container-id",
				}

				manager.EXPECT().ProcessID().Return(processID).Once()
			})

			Specify("succeed to create process", func() {
				factory.EXPECT().
					NewCmd(mock.Anything, injector.ChaosInjectorBinaryLocation, mock.Anything).
					Run(func(ctx context.Context, name string, args []string) {
						parentIDIndex, deadlineIndex := -1, -1
						for index, arg := range args {
							if arg == injector.ParentPIDFlag.String() {
								parentIDIndex = index
							} else if arg == injector.DeadlineFlag.String() {
								deadlineIndex = index
							}

							if parentIDIndex != -1 && deadlineIndex != -1 {
								break
							}
						}

						// we expect to find flags
						Expect(parentIDIndex).ToNot(BeNumerically("==", -1))
						Expect(deadlineIndex).ToNot(BeNumerically("==", -1))

						// we expect each flag to have a defined value
						Expect(parentIDIndex + 1).To(BeNumerically("<", len(args)))
						Expect(deadlineIndex + 1).To(BeNumerically("<", len(args)))

						// we expect each value to be appropriate
						Expect(args[parentIDIndex+1]).To(Equal(strconv.Itoa(processID)))
						Expect(args[deadlineIndex+1]).To(Equal(deadline.Format(time.RFC3339)))
					}).Return(nil).Once()

				Expect(sut.NewInjectorBackgroundCmd(deadline, disruptionArgs, target, args)).Error().To(Succeed())
			})

			Specify("process duration is soon after deadline", func(outCtx SpecContext) {
				start := time.Now()

				factory.EXPECT().NewCmd(mock.Anything, injector.ChaosInjectorBinaryLocation, mock.Anything).Run(
					func(ctx context.Context, name string, args []string) {
						// here we assume we are running a forever loop and expect context to be canceled at some point
						// we also monitor outer context in case the implementation is incorrect and does not provide a context with deadline
						for {
							select {
							case <-ctx.Done():
								return
							case <-outCtx.Done():
								return
							}
						}
					}).Return(nil).Once()

				Expect(sut.NewInjectorBackgroundCmd(deadline, disruptionArgs, target, args)).Error().To(Succeed())
				// We expect the program duration to last at least deadlineDuration and up to 1% more
				Expect(time.Since(start)).To(BeNumerically("~", deadlineDuration, float64(deadlineDuration)*1.01))
			}, SpecTimeout(2*deadlineDuration))
		})
	})
})
