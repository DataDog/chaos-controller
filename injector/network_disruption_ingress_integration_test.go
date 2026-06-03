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

// ifbNameForTest is the IFB device name used in integration tests.
// The injector derives it from DisruptionUID (first 8 chars); DisruptionUID
// is empty in buildNetworkInjector, so the name is "ifb-".
const ifbNameForTest = "ifb-"

var _ = Describe("network ingress latency disruption", func() {
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
		It("creates IFB device with netem delay rule and removes it on Clean", func() {
			target, _ := startTargetContainer(ctx, netName)
			DeferCleanup(func() { _ = target.Terminate(ctx) })

			inj, targetPID := buildNetworkInjector(ctx, ingressLatencySpec(200), containerID(target))

			By("injecting 200ms ingress delay")
			Expect(inj.Inject()).To(Succeed())

			By("asserting IFB device created in target netns")
			Expect(listNetDevices(targetPID)).To(ContainSubstring(ifbNameForTest))

			By("asserting netem delay rule on IFB device")
			ifbQdisc := tcQdiscShowDev(targetPID, ifbNameForTest)
			Expect(ifbQdisc).To(ContainSubstring("netem"))
			Expect(ifbQdisc).To(ContainSubstring("delay 200ms"))

			By("cleaning up")
			Expect(inj.Clean()).To(Succeed())

			By("asserting IFB device removed after Clean")
			Eventually(func() string {
				return listNetDevices(targetPID)
			}, 5*time.Second, 500*time.Millisecond).ShouldNot(ContainSubstring(ifbNameForTest))
		})
	})

	Describe("behavioral assertions", func() {
		It("causes measurable HTTP latency increase on sender requests to target", func() {
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

			inj, targetPID := buildNetworkInjector(ctx, ingressLatencySpec(200), containerID(target))
			DeferCleanup(func() { _ = inj.Clean() })

			By("injecting 200ms ingress delay and activating matchall redirect to IFB")
			injectAndActivateIngress(inj, targetPID, ifbNameForTest)

			By("asserting HTTP latency increases to > 150ms (sender requests delayed at target ingress)")
			Eventually(func() int {
				return measureHTTPLatencyMS(ctx, sender, targetIP)
			}, 15*time.Second, 1*time.Second).Should(BeNumerically(">", 150))
		})
	})
})

var _ = Describe("network ingress drop-only disruption", func() {
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
		It("attaches BPF ingress filter without IFB device and cleans up", func() {
			target, _ := startTargetContainer(ctx, netName)
			DeferCleanup(func() { _ = target.Terminate(ctx) })

			inj, targetPID := buildNetworkInjector(ctx, ingressDropSpec(50), containerID(target))

			By("injecting 50% ingress drop")
			Expect(inj.Inject()).To(Succeed())

			By("asserting no IFB device created (ActionDrop path needs no shaping device)")
			Expect(listNetDevices(targetPID)).NotTo(ContainSubstring(ifbNameForTest))

			By("asserting BPF filter attached on eth0 ingress")
			Expect(tcFilterShowIngress(targetPID)).To(ContainSubstring("bpf"))

			By("cleaning up")
			Expect(inj.Clean()).To(Succeed())

			By("asserting BPF ingress filter removed after Clean")
			Eventually(func() string {
				return tcFilterShowIngress(targetPID)
			}, 5*time.Second, 500*time.Millisecond).ShouldNot(ContainSubstring("bpf"))
		})
	})

	Describe("behavioral assertions", func() {
		It("causes measurable ICMP packet loss on sender pings to target", func() {
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

			inj, targetPID := buildNetworkInjector(ctx, ingressDropSpec(50), containerID(target))
			DeferCleanup(func() { _ = inj.Clean() })

			By("injecting 50% ingress drop and activating matchall drop as fallback")
			injectAndActivateIngressDrop(inj, targetPID)

			By("asserting >20% ping loss (sender pings dropped at target ingress)")
			Eventually(func() float64 {
				return measurePingLoss(ctx, sender, targetIP, 10)
			}, 15*time.Second, 2*time.Second).Should(BeNumerically(">", 0.20))
		})
	})
})
