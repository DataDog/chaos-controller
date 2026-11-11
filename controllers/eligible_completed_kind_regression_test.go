// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package controllers

import (
	"context"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/services"
	chaostypes "github.com/DataDog/chaos-controller/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Completed kind eligibility regression", func() {
	DescribeTable("intersects only kinds whose injectors have not completed successfully",
		func(allowDisruptedTargets, cpuCompleted, expectedEligible bool) {
			instance := &chaosv1beta1.Disruption{
				ObjectMeta: metav1.ObjectMeta{Name: "new-cpu", Namespace: "app"},
				Spec: chaosv1beta1.DisruptionSpec{
					Level:                 chaostypes.DisruptionLevelPod,
					AllowDisruptedTargets: allowDisruptedTargets,
					CPUPressure:           &chaosv1beta1.CPUPressureSpec{},
				},
			}
			cpuPod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "old-cpu", Namespace: "chaos", Labels: map[string]string{
					chaostypes.TargetLabel:              "target",
					chaostypes.DisruptionNamespaceLabel: "app",
					chaostypes.DisruptionKindLabel:      chaostypes.DisruptionKindCPUPressure,
				}},
				Status: corev1.PodStatus{Phase: corev1.PodRunning, ContainerStatuses: []corev1.ContainerStatus{{Name: "injector", State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}}}},
			}
			if cpuCompleted {
				cpuPod.Status.Phase = corev1.PodSucceeded
				cpuPod.Status.ContainerStatuses[0].State = corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 0}}
			}
			networkPod := cpuPod.DeepCopy()
			networkPod.Name = "active-network"
			networkPod.Labels[chaostypes.DisruptionKindLabel] = chaostypes.DisruptionKindNetworkDisruption
			networkPod.Status.Phase = corev1.PodRunning
			networkPod.Status.ContainerStatuses[0].State = corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}
			k8sClient := fake.NewClientBuilder().WithObjects(cpuPod, networkPod).Build()
			podService, err := services.NewChaosPodService(services.ChaosPodServiceConfig{Client: k8sClient, ChaosNamespace: "chaos"})
			Expect(err).NotTo(HaveOccurred())
			reconciler := &DisruptionReconciler{Client: k8sClient, ChaosPodService: podService, log: zap.NewNop().Sugar()}

			eligible, err := reconciler.getEligibleTargets(context.Background(), instance, []string{"target"})
			Expect(err).NotTo(HaveOccurred())
			if expectedEligible {
				Expect(eligible).To(HaveKey("target"))
			} else {
				Expect(eligible).To(BeEmpty())
			}
		},
		Entry("allows completed CPU plus active network when requesting CPU", true, true, true),
		Entry("still rejects an actively injected overlapping kind", true, false, false),
		Entry("still rejects active network when sharing is disabled", false, true, false),
	)
})
