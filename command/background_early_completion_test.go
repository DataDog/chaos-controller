// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package command

import (
	"time"

	"github.com/DataDog/chaos-controller/process"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

var _ = Describe("Background command completion before bootstrap", func() {
	It("closes Done when Wait succeeds before the bootstrap timer fires", func() {
		oldDuration := cmdBootstrapAllowedDuration
		cmdBootstrapAllowedDuration = time.Hour
		DeferCleanup(func() { cmdBootstrapAllowedDuration = oldDuration })

		cmd := NewCmdMock(GinkgoT())
		cmd.EXPECT().DryRun().Return(false)
		cmd.EXPECT().Start().Return(nil).Once()
		cmd.EXPECT().Wait().Return(nil).Once()
		cmd.EXPECT().PID().Return(42).Once()
		background := NewBackgroundCmd(cmd, zap.NewNop().Sugar(), process.NewManager(false)).(*backgroundCmd)
		DeferCleanup(func() {
			// Unblock the lost-completion monitor after a baseline assertion failure.
			select {
			case <-background.Done():
			default:
				select {
				case background.err <- nil:
				default:
				}
			}
		})

		Expect(background.Start()).To(Succeed())
		Eventually(background.Done()).WithTimeout(time.Second).Should(BeClosed())
	})
})
