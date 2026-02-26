// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package controllers

import (
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

var _ = Describe("CPU Pressure", func() {
	var (
		cpuStress chaosv1beta1.Disruption
		targetPod corev1.Pod
	)

	BeforeEach(func() {
		cpuStress = chaosv1beta1.Disruption{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   namespace,
				Annotations: map[string]string{chaosv1beta1.SafemodeEnvironmentAnnotation: "lima"},
			},
			Spec: chaosv1beta1.DisruptionSpec{
				Duration:   "1m",
				Count:      &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
				Containers: []string{"ctn1"},
				CPUPressure: &chaosv1beta1.CPUPressureSpec{
					Count: &intstr.IntOrString{Type: intstr.String, StrVal: "100%"},
				},
			},
		}
	})

	JustBeforeEach(func(ctx SpecContext) {
		cpuStress, targetPod, _ = InjectPodsAndDisruption(ctx, cpuStress, true)
	})

	DescribeTable("targeted container is stopped", func(ctx SpecContext, forced bool) {
		ExpectDisruptionStatus(ctx, cpuStress, chaostypes.DisruptionInjectionStatusInjected)

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

		disruptionKey := types.NamespacedName{Namespace: cpuStress.Namespace, Name: cpuStress.Name}

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
				}).WithContext(ctx).Within(calcDisruptionGoneTimeout(cpuStress)).ProbeEvery(disruptionPotentialChangesEvery).Should(Succeed())
			},
			func(ctx SpecContext) {
				GinkgoHelper()

				CreateDisruption(ctx, stopTargetedContainer, targetPod)

				// First, we wait to have the container failure injected
				ExpectDisruptionStatus(ctx, stopTargetedContainer, chaostypes.DisruptionInjectionStatusInjected)

				// Then the container failure disruption should disappear so we stop killing the container we want to stress
				ExpectDisruptionStatus(ctx, stopTargetedContainer, chaostypes.DisruptionInjectionStatusPreviouslyInjected)

				// once it's done, we expect to eventually have stressers back before the end of the disruption
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
				}).WithContext(ctx).Within(calcDisruptionGoneTimeout(cpuStress)).ProbeEvery(disruptionPotentialChangesEvery).Should(Succeed())
			},
		}.DoAndWait(ctx)
	},
		Entry("by a SIGTERM", false),
		Entry("by a SIGKILL", true),
	)
})
