// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package controllers

import (
	"context"
	"fmt"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/services"
	chaostypes "github.com/DataDog/chaos-controller/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Current readiness accounting regressions", func() {
	DescribeTable("counts targets only when every disruption kind is currently ready",
		func(readiness [][]bool, previouslyInjected bool, expectedCount int, expectedStatus chaostypes.DisruptionInjectionStatus) {
			scheme := runtime.NewScheme()
			Expect(corev1.AddToScheme(scheme)).To(Succeed())
			Expect(chaosv1beta1.AddToScheme(scheme)).To(Succeed())
			count := intstr.FromInt(len(readiness))
			now := metav1.Now()
			instance := &chaosv1beta1.Disruption{
				ObjectMeta: metav1.ObjectMeta{Name: "readiness", Namespace: "app", CreationTimestamp: now},
				Spec: chaosv1beta1.DisruptionSpec{
					Count: &count, Duration: "1h", Level: chaostypes.DisruptionLevelPod,
					CPUPressure: &chaosv1beta1.CPUPressureSpec{},
					Network:     &chaosv1beta1.NetworkDisruptionSpec{},
				},
				Status: chaosv1beta1.DisruptionStatus{
					DesiredTargetsCount: len(readiness),
					TargetInjections:    make(chaosv1beta1.TargetInjections),
				},
			}
			initialStatus := chaostypes.DisruptionTargetInjectionStatusNotInjected
			if previouslyInjected {
				initialStatus = chaostypes.DisruptionTargetInjectionStatusInjected
				instance.Status.InjectionStatus = chaostypes.DisruptionInjectionStatusInjected
			}
			objects := []client.Object{instance}
			kinds := []chaostypes.DisruptionKindName{chaostypes.DisruptionKindCPUPressure, chaostypes.DisruptionKindNetworkDisruption}
			for targetIndex, readyKinds := range readiness {
				target := fmt.Sprintf("target-%d", targetIndex)
				instance.Status.TargetInjections[target] = make(chaosv1beta1.TargetInjectorMap)
				for kindIndex, ready := range readyKinds {
					kind := kinds[kindIndex]
					podName := fmt.Sprintf("injector-%d-%d", targetIndex, kindIndex)
					instance.Status.TargetInjections[target][kind] = chaosv1beta1.TargetInjection{InjectorPodName: podName, InjectionStatus: initialStatus}
					condition := corev1.ConditionFalse
					if ready {
						condition = corev1.ConditionTrue
					}
					objects = append(objects, &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{Name: podName, Namespace: "chaos", Labels: map[string]string{
							chaostypes.DisruptionNameLabel:      instance.Name,
							chaostypes.DisruptionNamespaceLabel: instance.Namespace,
							chaostypes.TargetLabel:              target,
							chaostypes.DisruptionKindLabel:      string(kind),
						}},
						Status: corev1.PodStatus{
							Phase:             corev1.PodRunning,
							ContainerStatuses: []corev1.ContainerStatus{{Name: "injector", State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{StartedAt: now}}}},
							Conditions:        []corev1.PodCondition{{Type: corev1.PodReady, Status: condition, LastTransitionTime: now}},
						},
					})
				}
			}
			k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&chaosv1beta1.Disruption{}).WithObjects(objects...).Build()
			Expect(k8sClient.Get(context.Background(), client.ObjectKeyFromObject(instance), instance)).To(Succeed())
			podService, err := services.NewChaosPodService(services.ChaosPodServiceConfig{Client: k8sClient, ChaosNamespace: "chaos"})
			Expect(err).NotTo(HaveOccurred())
			reconciler := &DisruptionReconciler{Client: k8sClient, ChaosPodService: podService, log: zap.NewNop().Sugar()}

			Expect(reconciler.updateInjectionStatus(context.Background(), instance)).To(Succeed())
			Expect(instance.Status.InjectedTargetsCount).To(Equal(expectedCount))
			Expect(instance.Status.InjectionStatus).To(Equal(expectedStatus))
		},
		Entry("does not combine different targets' incomplete injections", [][]bool{{true, false}, {false, true}}, false, 0, chaostypes.DisruptionInjectionStatusPartiallyInjected),
		Entry("does not retain full injection after one injector becomes unready", [][]bool{{true, false}}, true, 0, chaostypes.DisruptionInjectionStatusPartiallyInjected),
		Entry("counts one complete target beside an incomplete target", [][]bool{{true, true}, {true, false}}, false, 1, chaostypes.DisruptionInjectionStatusPartiallyInjected),
		Entry("recognizes all kinds ready on every target", [][]bool{{true, true}}, false, 1, chaostypes.DisruptionInjectionStatusInjected),
	)
})
