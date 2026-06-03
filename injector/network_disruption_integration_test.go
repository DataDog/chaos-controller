// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

//go:build integration

package injector_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("network latency disruption", func() {
	var (
		ctx        context.Context
		cancel     context.CancelFunc
		netName    string
		netCleanup func()
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 2*time.Minute)
		DeferCleanup(cancel)

		netName, netCleanup = startIsolatedNetwork(ctx)
		DeferCleanup(netCleanup)
	})

	Describe("structural assertions", func() {
		It("applies netem delay rule to target netns and cleans it up", func() {
			target, _ := startTargetContainer(ctx, netName)
			DeferCleanup(func() { _ = target.Terminate(ctx) })

			inj, targetPID := buildNetworkInjector(ctx, latencySpec(200), containerID(target))

			By("injecting 200ms delay")
			Expect(inj.Inject()).To(Succeed())

			By("asserting netem delay rule exists in target netns")
			assertTCQdisc(targetPID, "netem")
			assertTCQdisc(targetPID, "delay 200ms")

			By("cleaning up")
			Expect(inj.Clean()).To(Succeed())

			By("asserting netem rule is removed after Clean")
			Eventually(func() string {
				return tcQdiscShow(targetPID)
			}, 5*time.Second, 500*time.Millisecond).ShouldNot(ContainSubstring("netem"))
		})
	})
})
