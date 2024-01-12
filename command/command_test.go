// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.
package command

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/DataDog/chaos-controller/process"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

var _ = Describe("Factory", func() {
	var (
		binaryToLaunch string
		argsToUse      []string
	)

	BeforeEach(func() {
		binaryToLaunch = "/usr/bin/this-is-magic"
		argsToUse = []string{"this", "is", "magic", "time", "--args"}
	})

	DescribeTable(
		"NewCmd creation define expected fields",
		func(ctx SpecContext, dryRun bool) {
			magicCmd := NewFactory(dryRun).NewCmd(ctx, binaryToLaunch, argsToUse)

			cmd, ok := magicCmd.(*cmd)
			Expect(ok).To(BeTrue())
			Expect(cmd).ToNot(BeNil())

			Expect(cmd.Stdout).To(Equal(os.Stdout))
			Expect(cmd.Stderr).To(Equal(os.Stderr))
			Expect(cmd.Path).To(Equal(binaryToLaunch))
			Expect(cmd.Args[1:]).To(Equal(argsToUse))
			Expect(cmd.DryRun()).To(Equal(dryRun))
		},
		Entry("when dryRun is true", true),
		Entry("when dryRun is false", true),
	)
})

var _ = Describe("Cmd", func() {
	var newCmd *cmd

	Specify("PID()", func() {
		By("Nil cmd")
		Expect(newCmd.PID()).To(Equal(process.NotFoundProcessPID))

		By("Setting up cmd")
		newCmd = &cmd{}
		Expect(newCmd.PID()).To(Equal(process.NotFoundProcessPID))

		By("Setting up exec.Cmd")
		newCmd.Cmd = &exec.Cmd{}
		Expect(newCmd.PID()).To(Equal(process.NotFoundProcessPID))

		By("Setting up process, returns process Pid")
		newCmd.Cmd.Process = &os.Process{
			Pid: 45,
		}
		Expect(newCmd.PID()).To(Equal(newCmd.Cmd.Process.Pid))
	})

	Specify("ExitCode()", func() {
		By("Nil cmd")
		Expect(newCmd.ExitCode()).To(Equal(NotFoundProcessExitCode))

		By("Setting up cmd")
		newCmd = &cmd{}
		Expect(newCmd.ExitCode()).To(Equal(NotFoundProcessExitCode))

		By("Setting up exec.Cmd")
		newCmd.Cmd = &exec.Cmd{}
		Expect(newCmd.ExitCode()).To(Equal(NotFoundProcessExitCode))

		By("Setting up process state, returns it's exit code")
		newCmd.Cmd.ProcessState = &os.ProcessState{}
		Expect(newCmd.ExitCode()).To(Equal(0))
	})
})

var _ = Describe("BackgroundCmd", func() {
	var (
		sut     BackgroundCmd
		manager *process.ManagerMock
		cmd     *CmdMock
	)

	BeforeEach(func() {
		manager = process.NewManagerMock(GinkgoT())
		cmd = NewCmdMock(GinkgoT())

		sut = NewBackgroundCmd(cmd, log, manager)
	})

	When("dryRun is true", func() {
		BeforeEach(func() {
			// no mock expectations defined for cmd, hence any call to underlying cmd would lead to a failing test
			cmd.EXPECT().DryRun().Return(true)
		})

		Specify("Start does nothing", func() {
			Expect(sut.Start()).To(Succeed())
		})

		Specify("KeepAlive does nothing", func() {
			sut.KeepAlive()
		})

		Specify("Stop does nothing", func() {
			Expect(sut.Stop()).To(Succeed())
		})
	})

	When("dryRun is false", func() {
		BeforeEach(func() {
			cmd.EXPECT().DryRun().Return(false)
		})

		Describe("Start", func() {
			Specify("cmd.Start fails", func() {
				cmd.EXPECT().Start().Return(errors.New("failed to start"))
				cmd.EXPECT().String().Return("does not matter")

				Expect(sut.Start()).Error().To(MatchError("unable to exec command 'does not matter': failed to start"))
			})

			When("cmd.Start succeeds", func() {
				cmdAfterBootstrapDuration := cmdBootstrapAllowedDuration * 2
				cmdBeforeBootstrapDuration := cmdBootstrapAllowedDuration / 2

				BeforeEach(func() {
					cmd.EXPECT().Start().Return(nil)
				})

				When("initial bootstrap succeed", func() {
					BeforeEach(func() {
						cmd.EXPECT().PID().Return(41).Once()
					})

					Specify("cmd.Wait succeed immediately, Start succeed", func() {
						cmd.EXPECT().Wait().Return(nil)

						Expect(sut.Start()).To(Succeed())
					})

					ExpectStartSuccessInTime := func(cmdDuration time.Duration) {
						GinkgoHelper()

						start := time.Now()

						cmd.EXPECT().Wait().WaitUntil(time.After(cmdDuration)).Return(nil)
						Expect(sut.Start()).To(Succeed())
						Expect(time.Since(start)).To(BeNumerically("~", cmdBootstrapAllowedDuration, float64(cmdBootstrapAllowedDuration)*1.01))
					}

					Specify("cmd.Wait succeed BEFORE initial bootstrap, Start succeed", func() {
						ExpectStartSuccessInTime(cmdBeforeBootstrapDuration)
					})

					Specify("cmd.Wait succeed AFTER initial bootstrap, Start succeed", func() {
						ExpectStartSuccessInTime(cmdAfterBootstrapDuration)
					})

					When("but cmd fails soon after", func() {
						var logs *observer.ObservedLogs

						JustBeforeEach(func() {
							var obs zapcore.Core
							obs, logs = observer.New(zap.InfoLevel)
							z := zap.New(obs)

							bgCmd, ok := sut.(*backgroundCmd)
							Expect(ok).To(BeTrue())
							Expect(bgCmd).ToNot(BeNil())

							// we want to override the logger to track the logs sent AFTER bootstrap
							bgCmd.log = z.Sugar()
						})

						Specify("cmd.Wait fails AFTER initial bootstrap, Start SUCCEED and produce log", func() {
							cmd.EXPECT().Wait().WaitUntil(time.After(cmdAfterBootstrapDuration)).Return(errors.New("wait fails"))

							Expect(sut.Start()).To(Succeed())

							Eventually(func(g Gomega) {
								logEntries := logs.All()
								g.Expect(logEntries).To(HaveLen(1))

								logEntry := logEntries[0]
								g.Expect(logEntry.Level).To(Equal(zapcore.WarnLevel))
								g.Expect(logEntry.Message).To(Equal("background command exited with an error"))
								g.Expect(logEntry.ContextMap()["error"]).To(Equal("wait fails"))
							}).Within(cmdAfterBootstrapDuration * 2).ProbeEvery(cmdBootstrapAllowedDuration / 10).Should(Succeed())
						})
					})
				})

				When("initial bootstrap fails", func() {
					Specify("cmd.Wait fails immediately, Start fails", func() {
						cmd.EXPECT().Wait().Return(errors.New("wait fails"))

						Expect(sut.Start()).To(MatchError("an error occurred during startup of exec command: wait fails"))
					})

					Specify("cmd.Wait fails BEFORE initial bootstrap, Start fails", func() {
						cmd.EXPECT().Wait().WaitUntil(time.After(cmdBootstrapAllowedDuration / 2)).Return(errors.New("wait fails"))

						Expect(sut.Start()).To(MatchError("an error occurred during startup of exec command: wait fails"))
					})
				})
			})
		})

		SetupMockExpect := func(cmd *CmdMock, manager *process.ManagerMock, proc *os.Process, signal os.Signal, findErr, signalErr error, times int) {
			GinkgoHelper()

			manager.EXPECT().Find(mock.Anything).Return(proc, findErr).Times(times)
			if findErr == nil {
				manager.EXPECT().Signal(proc, signal).Return(signalErr).Times(times)
			}
		}

		DescribeTable(
			"KeepAlive",
			func(proc *os.Process, findErr, signalErr error, times int) {
				SetupMockExpect(cmd, manager, proc, syscall.SIGCONT, findErr, signalErr, times)

				sut.KeepAlive()
				sut.KeepAlive() // we call it twice volountarily to ensure a single goroutine is created

				// Wait enough interval to have at least expected calls (times + 20%)
				<-time.After(cmdKeepAliveTickDuration*time.Duration(times) + cmdBootstrapAllowedDuration/5)

				underSut, ok := sut.(*backgroundCmd)
				Expect(ok).To(BeTrue())
				Expect(underSut).ToNot(BeNil())

				// Stop timer ASAP to have realistic amount of calls
				underSut.resetTicker()
			},
			Entry("Send a signal several times", &os.Process{}, nil, nil, 2),
			Entry("Find fails once", nil, errors.New("find fails"), nil, 1),
			Entry("Signal fails once", &os.Process{}, nil, errors.New("signal fails"), 1),
		)

		DescribeTable(
			"Stop",
			func(proc *os.Process, findErr, signalErr error, match types.GomegaMatcher) {
				SetupMockExpect(cmd, manager, proc, syscall.SIGTERM, findErr, signalErr, 1)

				Expect(sut.Stop()).To(match)
			},
			Entry("Released process succeed", &os.Process{Pid: process.NotFoundProcessPID}, nil, nil, Succeed()),
			Entry("Unknown process succeed", &os.Process{}, nil, nil, Succeed()),
			Entry("Send a stop signal succeed", &os.Process{Pid: 42}, nil, nil, Succeed()),
			Entry("Find fails returns the error", nil, errors.New("find fails"), nil, MatchError("an error occurred while finding signal with pid -1: find fails")),
			Entry("Signal fails returns the error", &os.Process{Pid: 1}, nil, errors.New("signal fails"), MatchError("an error occurred while sending SIGTERM signal to process with pid -1: signal fails")),
		)
	})
})
