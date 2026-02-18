// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package controllers

import (
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	chaostypes "github.com/DataDog/chaos-controller/types"
)

var _ = Describe("Memory Pressure", func() {
	var (
		memoryStress chaosv1beta1.Disruption
		targetPod    corev1.Pod
	)

	BeforeEach(func() {
		memoryStress = chaosv1beta1.Disruption{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   namespace,
				Annotations: map[string]string{chaosv1beta1.SafemodeEnvironmentAnnotation: "lima"},
			},
			Spec: chaosv1beta1.DisruptionSpec{
				Duration: "1m",
				Count:    &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
				Unsafemode: &chaosv1beta1.UnsafemodeSpec{
					DisableAll: true,
				},
				MemoryPressure: &chaosv1beta1.MemoryPressureSpec{
					TargetPercent: "50%",
				},
			},
		}
	})

	JustBeforeEach(func(ctx SpecContext) {
		memoryStress, targetPod, _ = InjectPodsAndDisruption(ctx, memoryStress, true)
	})

	It("should inject memory pressure with correct chaos pod args", func(ctx SpecContext) {
		ExpectDisruptionStatus(ctx, memoryStress, chaostypes.DisruptionInjectionStatusInjected)

		By("Ensuring chaos pod is created and carries memory-pressure args")
		Eventually(func(g Gomega, ctx SpecContext) {
			chaosPods, err := listChaosPods(ctx, memoryStress)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(chaosPods.Items).To(HaveLen(1))

			args := strings.Join(chaosPods.Items[0].Spec.Containers[0].Args, " ")
			g.Expect(args).To(ContainSubstring("memory-pressure"))
			g.Expect(args).To(ContainSubstring("--target-percent"))
		}).WithContext(ctx).Within(calcDisruptionGoneTimeout(memoryStress)).ProbeEvery(disruptionPotentialChangesEvery).Should(Succeed())
	})

	Context("immediate allocation (no ramp)", func() {
		BeforeEach(func() {
			memoryStress.Spec.Duration = shortDisruptionDuration
			memoryStress.Spec.MemoryPressure.TargetPercent = "30%"
		})

		It("should allocate memory immediately and expire naturally", func(ctx SpecContext) {
			ExpectDisruptionStatus(ctx, memoryStress, chaostypes.DisruptionInjectionStatusInjected)

			By("Ensuring chaos pod does not carry a ramp-duration argument")
			Eventually(func(g Gomega, ctx SpecContext) {
				chaosPods, err := listChaosPods(ctx, memoryStress)
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(chaosPods.Items).To(HaveLen(1))

				args := strings.Join(chaosPods.Items[0].Spec.Containers[0].Args, " ")
				g.Expect(args).To(ContainSubstring("memory-pressure"))
				g.Expect(args).ToNot(ContainSubstring("--ramp-duration"))
			}).WithContext(ctx).Within(calcDisruptionGoneTimeout(memoryStress)).ProbeEvery(disruptionPotentialChangesEvery).Should(Succeed())

			Concurrently{
				func(ctx SpecContext) {
					By("Waiting for the disruption to expire naturally")
					ExpectChaosPods(ctx, memoryStress, 0)
				},
				func(ctx SpecContext) {
					By("Waiting for the disruption to reach PreviouslyInjected")
					ExpectDisruptionStatus(ctx, memoryStress, chaostypes.DisruptionInjectionStatusPreviouslyInjected)
				},
			}.DoAndWait(ctx)
		})
	})

	Context("with ramp duration", func() {
		BeforeEach(func() {
			memoryStress.Spec.MemoryPressure.RampDuration = chaosv1beta1.DisruptionDuration("30s")
		})

		It("should inject memory pressure with ramp-duration argument", func(ctx SpecContext) {
			ExpectDisruptionStatus(ctx, memoryStress, chaostypes.DisruptionInjectionStatusInjected)

			By("Ensuring chaos pod carries the ramp-duration argument")
			Eventually(func(g Gomega, ctx SpecContext) {
				chaosPods, err := listChaosPods(ctx, memoryStress)
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(chaosPods.Items).To(HaveLen(1))

				args := strings.Join(chaosPods.Items[0].Spec.Containers[0].Args, " ")
				g.Expect(args).To(ContainSubstring("memory-pressure"))
				g.Expect(args).To(ContainSubstring("--ramp-duration"))
			}).WithContext(ctx).Within(calcDisruptionGoneTimeout(memoryStress)).ProbeEvery(disruptionPotentialChangesEvery).Should(Succeed())
		})
	})

	Context("combined with CPU pressure", func() {
		BeforeEach(func() {
			memoryStress.Spec.CPUPressure = &chaosv1beta1.CPUPressureSpec{}
		})

		It("should create chaos pods for both disruption kinds", func(ctx SpecContext) {
			ExpectDisruptionStatus(ctx, memoryStress, chaostypes.DisruptionInjectionStatusInjected)

			By("Ensuring that two chaos pods are created (one per disruption kind)")
			ExpectChaosPods(ctx, memoryStress, 2)

			By("Verifying chaos pods carry both memory-pressure and cpu-pressure args")
			Concurrently{
				func(ctx SpecContext) {
					Eventually(expectChaosPodWithArgs(memoryStress, "memory-pressure")).
						WithContext(ctx).
						Within(calcDisruptionGoneTimeout(memoryStress)).
						ProbeEvery(disruptionPotentialChangesEvery).
						Should(Succeed())
				},
				func(ctx SpecContext) {
					Eventually(expectChaosPodWithArgs(memoryStress, "cpu-pressure")).
						WithContext(ctx).
						Within(calcDisruptionGoneTimeout(memoryStress)).
						ProbeEvery(disruptionPotentialChangesEvery).
						Should(Succeed())
				},
			}.DoAndWait(ctx)
		})
	})

	Context("pulse mode", func() {
		BeforeEach(func() {
			memoryStress.Spec.Duration = "2m"
			memoryStress.Spec.Pulse = &chaosv1beta1.DisruptionPulse{
				ActiveDuration:  chaosv1beta1.DisruptionDuration("15s"),
				DormantDuration: chaosv1beta1.DisruptionDuration("10s"),
			}
		})

		It("should inject with pulse arguments and cycle through active/dormant phases", func(ctx SpecContext) {
			ExpectDisruptionStatus(ctx, memoryStress, chaostypes.DisruptionInjectionStatusInjected)

			By("Ensuring chaos pod is created")
			ExpectChaosPods(ctx, memoryStress, 1)

			By("Verifying chaos pod has pulse arguments")
			Eventually(func(g Gomega, ctx SpecContext) {
				chaosPods, err := listChaosPods(ctx, memoryStress)
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(chaosPods.Items).To(HaveLen(1))

				args := strings.Join(chaosPods.Items[0].Spec.Containers[0].Args, " ")
				g.Expect(args).To(ContainSubstring("memory-pressure"))
				g.Expect(args).To(ContainSubstring("--pulse-active-duration"))
				g.Expect(args).To(ContainSubstring("--pulse-dormant-duration"))
			}).WithContext(ctx).Within(calcDisruptionGoneTimeout(memoryStress)).ProbeEvery(disruptionPotentialChangesEvery).Should(Succeed())

			By("Ensuring disruption stays healthy throughout pulse cycle")
			ExpectDisruptionStatusUntilExpired(ctx, memoryStress, chaostypes.DisruptionInjectionStatusInjected)
		})
	})

	DescribeTable("targeted container is stopped", func(ctx SpecContext, forced bool) {
		ExpectDisruptionStatus(ctx, memoryStress, chaostypes.DisruptionInjectionStatusInjected)

		stopTargetedContainer := chaosv1beta1.Disruption{
			ObjectMeta: metav1.ObjectMeta{
				Name:        targetPod.Name + "-term",
				Namespace:   namespace,
				Annotations: map[string]string{chaosv1beta1.SafemodeEnvironmentAnnotation: "lima"},
			},
			Spec: chaosv1beta1.DisruptionSpec{
				AllowDisruptedTargets: true,
				StaticTargeting:       true,
				Duration:              "15s",
				Count:                 &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
				Containers:            []string{"ctn1"},
				ContainerFailure: &chaosv1beta1.ContainerFailureSpec{
					Forced: forced,
				},
			},
		}

		disruptionKey := types.NamespacedName{Namespace: memoryStress.Namespace, Name: memoryStress.Name}

		Concurrently{
			func(ctx SpecContext) {
				Consistently(func(g Gomega, ctx SpecContext) {
					var dis chaosv1beta1.Disruption
					if err := k8sClient.Get(ctx, disruptionKey, &dis); apierrors.IsNotFound(err) {
						return
					} else {
						g.Expect(err).ToNot(HaveOccurred())
					}

					g.Expect(dis.Status.InjectionStatus).To(BeElementOf(
						chaostypes.DisruptionInjectionStatusInjected,
						chaostypes.DisruptionInjectionStatusPausedInjected,
						chaostypes.DisruptionInjectionStatusPreviouslyInjected,
					))
				}).WithContext(ctx).Within(calcDisruptionGoneTimeout(memoryStress)).ProbeEvery(disruptionPotentialChangesEvery).Should(Succeed())
			},
			func(ctx SpecContext) {
				GinkgoHelper()

				CreateDisruption(ctx, stopTargetedContainer, targetPod)

				ExpectDisruptionStatus(ctx, stopTargetedContainer, chaostypes.DisruptionInjectionStatusInjected)
				ExpectDisruptionStatus(ctx, stopTargetedContainer, chaostypes.DisruptionInjectionStatusPreviouslyInjected)

				Eventually(func(g Gomega, ctx SpecContext) {
					var freshDisruption chaosv1beta1.Disruption

					err := k8sClient.Get(ctx, disruptionKey, &freshDisruption)
					if apierrors.IsNotFound(err) {
						return
					}
					g.Expect(err).ToNot(HaveOccurred())

					chaosPods, err := listChaosPods(ctx, freshDisruption)
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(chaosPods.Items).To(HaveLen(1))

					pod := chaosPods.Items[0]
					AddReportEntry("Checking chaos pod is running after container restart", pod.Name)
					g.Expect(allContainersAreRunning(ctx, pod)).To(BeTrue())
				}).WithContext(ctx).Within(calcDisruptionGoneTimeout(memoryStress)).ProbeEvery(disruptionPotentialChangesEvery).Should(Succeed())
			},
		}.DoAndWait(ctx)
	},
		Entry("by a SIGTERM", false),
		Entry("by a SIGKILL", true),
	)
})

// expectChaosPodWithArgs returns an Eventually-compatible function that verifies at least one
// chaos pod's container args contain the given substring (e.g. "memory-pressure", "cpu-pressure").
func expectChaosPodWithArgs(disruption chaosv1beta1.Disruption, argsSubstring string) func(ctx SpecContext) error {
	return func(ctx SpecContext) error {
		chaosPods, err := listChaosPods(ctx, disruption)
		if err != nil {
			return fmt.Errorf("listing chaos pods: %w", err)
		}

		for i := range chaosPods.Items {
			args := strings.Join(chaosPods.Items[i].Spec.Containers[0].Args, " ")
			if strings.Contains(args, argsSubstring) {
				AddReportEntry(fmt.Sprintf("Found chaos pod %s with args containing %q", chaosPods.Items[i].Name, argsSubstring))

				return nil
			}
		}

		return fmt.Errorf("no chaos pod found with args containing %q", argsSubstring)
	}
}
