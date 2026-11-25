// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package targetselector_test

import (
	"context"
	"time"

	chaos "github.com/DataDog/chaos-controller/api/v1beta1"
	clog "github.com/DataDog/chaos-controller/log"
	"github.com/DataDog/chaos-controller/services"
	"github.com/DataDog/chaos-controller/targetselector"
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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ = Describe("On-init target cleanup health", func() {
	var (
		scheme     *runtime.Scheme
		selector   targetselector.TargetSelector
		disruption *chaos.Disruption
		pod        *corev1.Pod
	)

	BeforeEach(func() {
		scheme = runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		selector = targetselector.NewRunningTargetSelector(false, "controller-node")
		count := intstr.FromInt(1)
		disruption = &chaos.Disruption{
			ObjectMeta: metav1.ObjectMeta{Name: "on-init", Namespace: "ns"},
			Spec: chaos.DisruptionSpec{
				Count: &count, Level: chaostypes.DisruptionLevelPod, OnInit: true, Duration: "1h",
				Selector: map[string]string{"app": "test"},
				Network:  &chaos.NetworkDisruptionSpec{Drop: 100},
			},
			Status: chaos.DisruptionStatus{TargetInjections: chaos.TargetInjections{
				"target": {chaostypes.DisruptionKindNetworkDisruption: {InjectionStatus: chaostypes.DisruptionTargetInjectionStatusInjected}},
			}},
		}
		pod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "target", Namespace: "ns",
				Labels: map[string]string{"app": "test", chaostypes.DisruptOnInitLabel: "true"},
			},
			Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{{Name: "chaos-handler", Image: "handler:v1"}, {Name: "bootstrap", Image: "app:v1"}},
				Containers:     []corev1.Container{{Name: "app", Image: "app:v1"}},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodPending,
				InitContainerStatuses: []corev1.ContainerStatus{{
					Name: "chaos-handler", State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
				}},
			},
		}
	})

	DescribeTable("does not confuse an active on-init target with a dead pod", func(onInit bool, state corev1.ContainerState, wantHealthy bool) {
		disruption.Spec.OnInit = onInit
		pod.Status.InitContainerStatuses[0].State = state
		cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pod).Build()
		if wantHealthy {
			matching, _, err := selector.GetMatchingPodsOverTotalPods(cl, disruption)
			Expect(err).NotTo(HaveOccurred())
			Expect(matching.Items).To(HaveLen(1))
			Expect(selector.TargetIsHealthy(pod.Name, cl, disruption)).To(Succeed())
		} else {
			Expect(selector.TargetIsHealthy(pod.Name, cl, disruption)).NotTo(Succeed())
		}
	},
		Entry("running chaos handler", true, corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}, true),
		Entry("completed handler before later init containers finish", true, corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "Completed", ExitCode: 0}}, true),
		Entry("ordinary pending pod", false, corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}, false),
		Entry("failed chaos handler", true, corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "Error", ExitCode: 1}}, false),
	)

	It("retains the cleanup finalizer after an injector failure while its on-init target is still pending", func() {
		now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
		started := true
		injector := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "injector", Namespace: "chaos",
				DeletionTimestamp: &metav1.Time{Time: now},
				Finalizers:        []string{chaostypes.ChaosPodFinalizer},
				Labels: map[string]string{
					chaostypes.TargetLabel:         pod.Name,
					chaostypes.DisruptionKindLabel: chaostypes.DisruptionKindNetworkDisruption,
				},
			},
			Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "injector", Image: "chaos:v1"}}},
			Status: corev1.PodStatus{
				Phase: corev1.PodFailed,
				ContainerStatuses: []corev1.ContainerStatus{{
					Name: "injector", Started: &started,
					State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{
						ExitCode: 1, Reason: "Error", StartedAt: metav1.NewTime(now.Add(-time.Minute)), FinishedAt: metav1.NewTime(now),
					}},
				}},
			},
		}
		cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pod, injector).Build()
		ctx := clog.WithLogger(context.Background(), zap.NewNop().Sugar())
		current := &corev1.Pod{}
		Expect(cl.Get(ctx, client.ObjectKeyFromObject(injector), current)).To(Succeed())
		service, err := services.NewChaosPodService(services.ChaosPodServiceConfig{
			Client: cl, Reader: cl, ChaosNamespace: "chaos", TargetSelector: selector,
		})
		Expect(err).NotTo(HaveOccurred())
		stuck, err := service.HandleChaosPodTermination(ctx, disruption, current)
		Expect(err).NotTo(HaveOccurred())
		Expect(stuck).To(BeTrue())
		Expect(controllerutil.ContainsFinalizer(current, chaostypes.ChaosPodFinalizer)).To(BeTrue())
	})
})
