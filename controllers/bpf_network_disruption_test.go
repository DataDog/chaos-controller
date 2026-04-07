// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package controllers

import (
	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	chaostypes "github.com/DataDog/chaos-controller/types"
	. "github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// These E2E tests verify that the BPF disruption engine correctly creates chaos pods
// for various network disruption scenarios including ingress, egress, UDP, and per-host targeting.
// Tests use DryRun mode — they verify chaos pod creation and injection status,
// not actual packet behavior (which requires a live traffic generator).
var _ = Describe("BPF Network Disruption E2E", func() {
	var (
		targetPod, anotherTargetPod corev1.Pod
		disruption                  chaosv1beta1.Disruption
		skipSecondPod               bool
		expectedDisruptionStatus    chaostypes.DisruptionInjectionStatus
	)

	BeforeEach(func(ctx SpecContext) {
		disruption = chaosv1beta1.Disruption{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   namespace,
				Annotations: map[string]string{chaosv1beta1.SafemodeEnvironmentAnnotation: "lima"},
			},
			Spec: chaosv1beta1.DisruptionSpec{
				DryRun: true,
				Count:  &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
				Unsafemode: &chaosv1beta1.UnsafemodeSpec{
					DisableAll: true,
				},
				Containers: []string{"ctn1"},
			},
		}

		skipSecondPod = false
		expectedDisruptionStatus = chaostypes.DisruptionInjectionStatusInjected
	})

	JustBeforeEach(func(ctx SpecContext) {
		By("Creating disruption resource and waiting for injection to be done")
		disruption, targetPod, anotherTargetPod = InjectPodsAndDisruption(ctx, disruption, skipSecondPod)
		ExpectDisruptionStatus(ctx, disruption, expectedDisruptionStatus)
	})

	// --- Egress disruption tests (BPF classifier at parent 1:0) ---

	Context("egress drop with host targeting", func() {
		BeforeEach(func() {
			disruption.Spec.Network = &chaosv1beta1.NetworkDisruptionSpec{
				Hosts: []chaosv1beta1.NetworkDisruptionHostSpec{
					{
						Host:     "10.0.0.1",
						Port:     80,
						Protocol: "tcp",
						Flow:     chaosv1beta1.FlowEgress,
					},
				},
				Drop: 100,
			}
		})

		It("should create chaos pods for egress drop with BPF classification", func(ctx SpecContext) {
			By("Ensuring that the chaos pods have been created")
			ExpectChaosPods(ctx, disruption, 3)
		})
	})

	Context("egress delay and bandwidth limit", func() {
		BeforeEach(func() {
			disruption.Spec.Network = &chaosv1beta1.NetworkDisruptionSpec{
				Hosts: []chaosv1beta1.NetworkDisruptionHostSpec{
					{
						Host:     "10.0.0.1",
						Port:     443,
						Protocol: "tcp",
						Flow:     chaosv1beta1.FlowEgress,
					},
				},
				Delay:          500,
				BandwidthLimit: 10000,
			}
		})

		It("should create chaos pods for egress delay+bandwidth with BPF classification", func(ctx SpecContext) {
			By("Ensuring that the chaos pods have been created")
			ExpectChaosPods(ctx, disruption, 3)
		})
	})

	Context("egress UDP disruption", func() {
		BeforeEach(func() {
			disruption.Spec.Network = &chaosv1beta1.NetworkDisruptionSpec{
				Hosts: []chaosv1beta1.NetworkDisruptionHostSpec{
					{
						Host:     "8.8.8.8",
						Port:     53,
						Protocol: "udp",
						Flow:     chaosv1beta1.FlowEgress,
					},
				},
				Drop: 100,
			}
		})

		It("should create chaos pods for UDP egress drop (previously unreliable with net_cls)", func(ctx SpecContext) {
			By("Ensuring that the chaos pods have been created")
			ExpectChaosPods(ctx, disruption, 3)
		})
	})

	// --- Ingress disruption tests (BPF DirectAction on clsact) ---

	Context("ingress drop only (BPF TC_ACT_SHOT)", func() {
		BeforeEach(func() {
			disruption.Spec.Network = &chaosv1beta1.NetworkDisruptionSpec{
				Hosts: []chaosv1beta1.NetworkDisruptionHostSpec{
					{
						Host: "10.0.0.0/8",
						Flow: chaosv1beta1.FlowIngress,
					},
				},
				Drop: 50,
			}
		})

		It("should create chaos pods for ingress drop with BPF DirectAction", func(ctx SpecContext) {
			By("Ensuring that the chaos pods have been created")
			ExpectChaosPods(ctx, disruption, 3)
		})
	})

	Context("ingress delay via IFB redirect", func() {
		BeforeEach(func() {
			disruption.Spec.Network = &chaosv1beta1.NetworkDisruptionSpec{
				Hosts: []chaosv1beta1.NetworkDisruptionHostSpec{
					{
						Host: "10.0.0.0/8",
						Flow: chaosv1beta1.FlowIngress,
					},
				},
				Delay: 200,
				Drop:  50,
			}
		})

		It("should create chaos pods for ingress shaping via IFB device", func(ctx SpecContext) {
			By("Ensuring that the chaos pods have been created")
			ExpectChaosPods(ctx, disruption, 3)
		})
	})

	Context("ingress UDP drop (previously broken with egress ACK trick)", func() {
		BeforeEach(func() {
			disruption.Spec.Network = &chaosv1beta1.NetworkDisruptionSpec{
				Hosts: []chaosv1beta1.NetworkDisruptionHostSpec{
					{
						Host:     "10.0.0.5",
						Port:     5353,
						Protocol: "udp",
						Flow:     chaosv1beta1.FlowIngress,
					},
				},
				Drop: 100,
			}
		})

		It("should create chaos pods for UDP ingress drop with BPF", func(ctx SpecContext) {
			By("Ensuring that the chaos pods have been created")
			ExpectChaosPods(ctx, disruption, 3)
		})
	})

	// --- Per-host ingress targeting (BPF matches real source IP) ---

	Context("ingress per-host targeting with multiple sources", func() {
		BeforeEach(func() {
			disruption.Spec.Network = &chaosv1beta1.NetworkDisruptionSpec{
				Hosts: []chaosv1beta1.NetworkDisruptionHostSpec{
					{
						Host:     "10.0.1.1",
						Port:     80,
						Protocol: "tcp",
						Flow:     chaosv1beta1.FlowIngress,
					},
					{
						Host:     "10.0.2.0/24",
						Protocol: "tcp",
						Flow:     chaosv1beta1.FlowIngress,
					},
				},
				Drop: 100,
			}
		})

		It("should create chaos pods for multi-host ingress targeting via BPF LPM trie", func(ctx SpecContext) {
			By("Ensuring that the chaos pods have been created")
			ExpectChaosPods(ctx, disruption, 3)
		})
	})

	// --- Mixed egress + ingress ---

	Context("mixed egress and ingress hosts", func() {
		BeforeEach(func() {
			disruption.Spec.Network = &chaosv1beta1.NetworkDisruptionSpec{
				Hosts: []chaosv1beta1.NetworkDisruptionHostSpec{
					{
						Host:     "10.0.0.1",
						Port:     80,
						Protocol: "tcp",
						Flow:     chaosv1beta1.FlowEgress,
					},
					{
						Host: "10.0.0.0/8",
						Flow: chaosv1beta1.FlowIngress,
					},
				},
				Drop:  50,
				Delay: 100,
			}
		})

		It("should create chaos pods for mixed direction disruption", func(ctx SpecContext) {
			By("Ensuring that the chaos pods have been created")
			ExpectChaosPods(ctx, disruption, 3)
		})
	})

	// --- Match-all (no hosts specified) ---

	Context("match-all egress with no hosts", func() {
		BeforeEach(func() {
			disruption.Spec.Network = &chaosv1beta1.NetworkDisruptionSpec{
				Drop: 100,
			}
		})

		It("should create chaos pods for match-all egress via BPF 0.0.0.0/0 rule", func(ctx SpecContext) {
			By("Ensuring that the chaos pods have been created")
			ExpectChaosPods(ctx, disruption, 3)
		})
	})

	// --- Allowed hosts (BPF ALLOW rules) ---

	Context("allowed hosts exclude specific IPs from disruption", func() {
		BeforeEach(func() {
			disruption.Spec.Network = &chaosv1beta1.NetworkDisruptionSpec{
				Drop: 100,
				AllowedHosts: []chaosv1beta1.NetworkDisruptionHostSpec{
					{
						Host:     "8.8.8.8",
						Port:     53,
						Protocol: "udp",
					},
				},
			}
		})

		It("should create chaos pods with allowed hosts as BPF ALLOW rules", func(ctx SpecContext) {
			By("Ensuring that the chaos pods have been created")
			ExpectChaosPods(ctx, disruption, 3)
		})
	})

	// --- ConnState warning (graceful degradation) ---

	Context("connState specified (warn and skip in BPF mode)", func() {
		BeforeEach(func() {
			disruption.Spec.Network = &chaosv1beta1.NetworkDisruptionSpec{
				Hosts: []chaosv1beta1.NetworkDisruptionHostSpec{
					{
						Host:      "10.0.0.1",
						Port:      80,
						Protocol:  "tcp",
						Flow:      chaosv1beta1.FlowEgress,
						ConnState: "new",
					},
				},
				Drop: 50,
			}
		})

		It("should create chaos pods even with connState (BPF ignores it with a warning)", func(ctx SpecContext) {
			By("Ensuring that the chaos pods have been created despite connState being unsupported in BPF")
			ExpectChaosPods(ctx, disruption, 3)
		})
	})

	// --- All disruption types on ingress ---

	Context("ingress corruption via IFB", func() {
		BeforeEach(func() {
			disruption.Spec.Network = &chaosv1beta1.NetworkDisruptionSpec{
				Hosts: []chaosv1beta1.NetworkDisruptionHostSpec{
					{
						Host: "10.0.0.0/8",
						Flow: chaosv1beta1.FlowIngress,
					},
				},
				Corrupt: 50,
			}
		})

		It("should create chaos pods for ingress corruption via IFB shaping", func(ctx SpecContext) {
			By("Ensuring that the chaos pods have been created")
			ExpectChaosPods(ctx, disruption, 3)
		})
	})

	Context("ingress bandwidth limit via IFB", func() {
		BeforeEach(func() {
			disruption.Spec.Network = &chaosv1beta1.NetworkDisruptionSpec{
				Hosts: []chaosv1beta1.NetworkDisruptionHostSpec{
					{
						Host: "10.0.0.0/8",
						Flow: chaosv1beta1.FlowIngress,
					},
				},
				BandwidthLimit: 5000,
			}
		})

		It("should create chaos pods for ingress bandwidth limiting via IFB shaping", func(ctx SpecContext) {
			By("Ensuring that the chaos pods have been created")
			ExpectChaosPods(ctx, disruption, 3)
		})
	})

	_ = targetPod
	_ = anotherTargetPod
})
