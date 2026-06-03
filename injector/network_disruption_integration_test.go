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

			By("asserting netem rule removed after Clean")
			Eventually(func() string {
				return tcQdiscShow(targetPID)
			}, 5*time.Second, 500*time.Millisecond).ShouldNot(ContainSubstring("netem"))
		})
	})

	Describe("behavioral assertions", func() {
		It("causes measurable HTTP latency increase", func() {
			target, targetIP := startTargetContainer(ctx, netName)
			sender, _ := startSenderContainer(ctx, netName)
			DeferCleanup(func() {
				_ = target.Terminate(ctx)
				_ = sender.Terminate(ctx)
			})

			By("measuring baseline latency (must be < 100ms)")
			baseline := measureHTTPLatencyMS(ctx, sender, targetIP)
			Expect(baseline).To(BeNumerically("<", 100),
				"baseline latency %dms too high — test environment issue", baseline)

			inj, targetPID := buildNetworkInjector(ctx, latencySpec(200), containerID(target))
			DeferCleanup(func() { _ = inj.Clean() })

			By("injecting 200ms delay and activating matchall classifier")
			injectAndActivate(inj, targetPID)

			By("asserting HTTP latency increases to > 150ms")
			Eventually(func() int {
				return measureHTTPLatencyMS(ctx, sender, targetIP)
			}, 15*time.Second, 1*time.Second).Should(BeNumerically(">", 150))
		})
	})
})

var _ = Describe("network packet loss disruption", func() {
	var (
		ctx        context.Context
		cancel     context.CancelFunc
		netName    string
		netCleanup func()
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 3*time.Minute)
		DeferCleanup(cancel)
		netName, netCleanup = startIsolatedNetwork(ctx)
		DeferCleanup(netCleanup)
	})

	Describe("structural assertions", func() {
		It("applies netem loss rule to target netns and cleans it up", func() {
			target, _ := startTargetContainer(ctx, netName)
			DeferCleanup(func() { _ = target.Terminate(ctx) })

			inj, targetPID := buildNetworkInjector(ctx, packetLossSpec(50), containerID(target))

			By("injecting 50% packet loss")
			Expect(inj.Inject()).To(Succeed())

			By("asserting netem loss rule exists in target netns")
			assertTCQdisc(targetPID, "netem")
			assertTCQdisc(targetPID, "loss 50%")

			By("cleaning up")
			Expect(inj.Clean()).To(Succeed())

			By("asserting netem rule removed after Clean")
			Eventually(func() string {
				return tcQdiscShow(targetPID)
			}, 5*time.Second, 500*time.Millisecond).ShouldNot(ContainSubstring("netem"))
		})
	})

	Describe("behavioral assertions", func() {
		It("causes measurable ICMP packet loss", func() {
			target, targetIP := startTargetContainer(ctx, netName)
			sender, _ := startSenderContainer(ctx, netName)
			DeferCleanup(func() {
				_ = target.Terminate(ctx)
				_ = sender.Terminate(ctx)
			})

			By("asserting baseline 0% ping loss")
			baselineLoss := measurePingLoss(ctx, sender, targetIP, 10)
			Expect(baselineLoss).To(BeNumerically("==", 0.0),
				"baseline ping loss must be 0%% (got %.0f%%)", baselineLoss*100)

			inj, targetPID := buildNetworkInjector(ctx, packetLossSpec(50), containerID(target))
			DeferCleanup(func() { _ = inj.Clean() })

			By("injecting 50% packet loss and activating matchall classifier")
			injectAndActivate(inj, targetPID)

			By("asserting >20% ping loss (conservative: 50% packet drop)")
			Eventually(func() float64 {
				return measurePingLoss(ctx, sender, targetIP, 10)
			}, 15*time.Second, 2*time.Second).Should(BeNumerically(">", 0.20))
		})
	})
})

var _ = Describe("network disruption clean-state verification", func() {
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

	It("removes all tc rules after Clean and HTTP traffic recovers", func() {
		target, targetIP := startTargetContainer(ctx, netName)
		sender, _ := startSenderContainer(ctx, netName)
		DeferCleanup(func() {
			_ = target.Terminate(ctx)
			_ = sender.Terminate(ctx)
		})

		inj, targetPID := buildNetworkInjector(ctx, latencySpec(200), containerID(target))

		By("injecting 200ms delay and activating matchall classifier")
		injectAndActivate(inj, targetPID)
		assertTCQdisc(targetPID, "netem")

		By("asserting degraded latency > 150ms")
		degraded := measureHTTPLatencyMS(ctx, sender, targetIP)
		Expect(degraded).To(BeNumerically(">", 150),
			"expected degraded latency >150ms, got %dms", degraded)

		By("cleaning up")
		Expect(inj.Clean()).To(Succeed())

		By("asserting no netem rule in target netns")
		Eventually(func() string {
			return tcQdiscShow(targetPID)
		}, 5*time.Second, 500*time.Millisecond).ShouldNot(ContainSubstring("netem"))

		By("asserting HTTP latency recovers to < 100ms")
		Eventually(func() int {
			return measureHTTPLatencyMS(ctx, sender, targetIP)
		}, 10*time.Second, 1*time.Second).Should(BeNumerically("<", 100))
	})
})
