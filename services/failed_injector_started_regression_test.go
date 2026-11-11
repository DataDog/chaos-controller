// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package services_test

import (
	"context"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/services"
	"github.com/DataDog/chaos-controller/targetselector"
	chaostypes "github.com/DataDog/chaos-controller/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Failed injector Started regression", func() {
	DescribeTable("does not interpret Started=false as proof that injection never ran",
		func(state, previousState corev1.ContainerState, expectedStuck bool) {
			started := false
			now := metav1.Now()
			disruption := &chaosv1beta1.Disruption{
				ObjectMeta: metav1.ObjectMeta{Name: "network", Namespace: "app"},
				Spec:       chaosv1beta1.DisruptionSpec{Level: chaostypes.DisruptionLevelPod, Network: &chaosv1beta1.NetworkDisruptionSpec{}},
			}
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "failed-injector", Namespace: "chaos", DeletionTimestamp: &now,
					Finalizers: []string{chaostypes.ChaosPodFinalizer},
					Labels:     map[string]string{chaostypes.TargetLabel: "target", chaostypes.DisruptionKindLabel: chaostypes.DisruptionKindNetworkDisruption},
				},
				Status: corev1.PodStatus{
					Phase:             corev1.PodFailed,
					ContainerStatuses: []corev1.ContainerStatus{{Name: "injector", Started: &started, State: state, LastTerminationState: previousState}},
				},
			}
			k8sClient := fake.NewClientBuilder().WithObjects(pod).Build()
			targetSelector := targetselector.NewTargetSelectorMock(GinkgoT())
			targetSelector.EXPECT().TargetIsHealthy("target", k8sClient, disruption).Return(nil).Once()
			podService, err := services.NewChaosPodService(services.ChaosPodServiceConfig{Client: k8sClient, TargetSelector: targetSelector})
			Expect(err).NotTo(HaveOccurred())

			stuck, err := podService.HandleChaosPodTermination(context.Background(), disruption, pod)
			Expect(err).NotTo(HaveOccurred())
			Expect(stuck).To(Equal(expectedStuck))
			if expectedStuck {
				Expect(pod.Finalizers).To(ContainElement(chaostypes.ChaosPodFinalizer))
			} else {
				Expect(pod.Finalizers).NotTo(ContainElement(chaostypes.ChaosPodFinalizer))
			}
		},
		Entry("retains failed cleanup", corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 1, Reason: "Error"}}, corev1.ContainerState{}, true),
		Entry("retains unknown termination", corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 137, Reason: "ContainerStatusUnknown"}}, corev1.ContainerState{}, true),
		Entry("retains an injector that is still running", corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}, corev1.ContainerState{}, true),
		Entry("retains a waiting injector with an earlier termination", corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}}, corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 1}}, true),
		Entry("removes an injector that could not be created", corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CreateContainerError"}}, corev1.ContainerState{}, false),
	)
})
