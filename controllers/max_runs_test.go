// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package controllers

import (
	"context"
	"errors"
	"testing"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	cLog "github.com/DataDog/chaos-controller/log"
	"github.com/DataDog/chaos-controller/o11y/metrics/noop"
	"github.com/DataDog/chaos-controller/services"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func newTestTargetPod(name, namespace, nodeName, ip string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			NodeName: nodeName,
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: ip,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:        "app",
					ContainerID: "containerd://" + name,
					State:       corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
				},
			},
		},
	}
}

func removeChaosPods(t *testing.T, ctx context.Context, cl client.Client, pods []corev1.Pod) {
	t.Helper()

	for i := range pods {
		pod := pods[i].DeepCopy()
		if len(pod.Finalizers) > 0 {
			pod.Finalizers = nil
			if err := cl.Update(ctx, pod); err != nil {
				t.Fatal(err)
			}
		}

		if err := cl.Delete(ctx, pod); err != nil {
			t.Fatal(err)
		}
	}
}

func TestStartInjectionMaxRunsCountsPerRound(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := chaosv1beta1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	logger := zap.NewNop().Sugar()
	ctx := cLog.WithLogger(context.Background(), logger)
	count := intstr.FromInt(2)
	maxRuns := 1

	disruption := &chaosv1beta1.Disruption{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "max-runs-round",
			Namespace:         "app",
			CreationTimestamp: metav1.Now(),
		},
		Spec: chaosv1beta1.DisruptionSpec{
			Level:    chaostypes.DisruptionLevelPod,
			Count:    &count,
			MaxRuns:  &maxRuns,
			Duration: "1h",
			Network:  &chaosv1beta1.NetworkDisruptionSpec{Drop: 10},
		},
		Status: chaosv1beta1.DisruptionStatus{
			DesiredTargetsCount:  2,
			SelectedTargetsCount: 2,
			TargetInjections: chaosv1beta1.TargetInjections{
				"pod-a": {
					chaostypes.DisruptionKindNetworkDisruption: {
						InjectionStatus: chaostypes.DisruptionTargetInjectionStatusNotInjected,
					},
				},
				"pod-b": {
					chaostypes.DisruptionKindNetworkDisruption: {
						InjectionStatus: chaostypes.DisruptionTargetInjectionStatusNotInjected,
					},
				},
			},
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&chaosv1beta1.Disruption{}).
		WithObjects(
			disruption,
			newTestTargetPod("pod-a", "app", "node-a", "10.0.0.1"),
			newTestTargetPod("pod-b", "app", "node-b", "10.0.0.2"),
			newTestTargetPod("pod-c", "app", "node-c", "10.0.0.3"),
		).
		Build()

	instance := &chaosv1beta1.Disruption{}
	if err := cl.Get(ctx, client.ObjectKeyFromObject(disruption), instance); err != nil {
		t.Fatal(err)
	}

	metricsSink := noop.New(logger)
	service, err := services.NewChaosPodService(services.ChaosPodServiceConfig{
		Client:         cl,
		Reader:         cl,
		ChaosNamespace: "chaos-system",
		MetricsSink:    metricsSink,
	})
	if err != nil {
		t.Fatal(err)
	}

	r := &DisruptionReconciler{
		Client:          cl,
		APIReader:       cl,
		BaseLog:         logger,
		log:             logger,
		Scheme:          scheme,
		Recorder:        record.NewFakeRecorder(20),
		MetricsSink:     metricsSink,
		ChaosPodService: service,
	}

	// Pass 1: both selected targets should be injected in a single round, counting as 1 run.
	if err := r.startInjection(ctx, instance); err != nil {
		t.Fatalf("first startInjection pass failed: %v", err)
	}

	chaosPods, err := service.GetChaosPodsOfDisruption(ctx, instance, nil)
	if err != nil {
		t.Fatal(err)
	}

	injectedTargets := map[string]bool{}
	for _, pod := range chaosPods {
		injectedTargets[pod.Labels[chaostypes.TargetLabel]] = true
	}

	if len(chaosPods) != 2 || !injectedTargets["pod-a"] || !injectedTargets["pod-b"] {
		t.Fatalf("first round injected targets %v (%d pods), want both pod-a and pod-b (2 pods)", injectedTargets, len(chaosPods))
	}

	stored := &chaosv1beta1.Disruption{}
	if err := cl.Get(ctx, client.ObjectKeyFromObject(instance), stored); err != nil {
		t.Fatal(err)
	}

	if stored.Status.RunCount != 1 {
		t.Fatalf("stored RunCount = %d, want 1", stored.Status.RunCount)
	}

	// Remove chaos pods and introduce a new selected target; a subsequent pass must create nothing.
	removeChaosPods(t, ctx, cl, chaosPods)

	remainingPods, err := service.GetChaosPodsOfDisruption(ctx, instance, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(remainingPods) != 0 {
		t.Fatalf("expected all chaos pods to be removed before second pass, got %d", len(remainingPods))
	}

	instance.Status.TargetInjections = chaosv1beta1.TargetInjections{
		"pod-c": {
			chaostypes.DisruptionKindNetworkDisruption: {
				InjectionStatus: chaostypes.DisruptionTargetInjectionStatusNotInjected,
			},
		},
	}

	if err := r.startInjection(ctx, instance); err != nil {
		t.Fatalf("second startInjection pass failed: %v", err)
	}

	chaosPods, err = service.GetChaosPodsOfDisruption(ctx, instance, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(chaosPods) != 0 {
		t.Fatalf("second pass after maxRuns reached created %d chaos pods, want 0", len(chaosPods))
	}

	if err := cl.Get(ctx, client.ObjectKeyFromObject(instance), stored); err != nil {
		t.Fatal(err)
	}

	if stored.Status.RunCount != 1 {
		t.Fatalf("stored RunCount after idle pass = %d, want 1", stored.Status.RunCount)
	}
}

func TestStartInjectionRetriesPartialErrorBeforeIncrementingRunCount(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := chaosv1beta1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	logger := zap.NewNop().Sugar()
	ctx := cLog.WithLogger(context.Background(), logger)
	count := intstr.FromInt(2)
	maxRuns := 1

	disruption := &chaosv1beta1.Disruption{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "max-runs-partial-error",
			Namespace:         "app",
			CreationTimestamp: metav1.Now(),
		},
		Spec: chaosv1beta1.DisruptionSpec{
			Level:    chaostypes.DisruptionLevelPod,
			Count:    &count,
			MaxRuns:  &maxRuns,
			Duration: "1h",
			Network:  &chaosv1beta1.NetworkDisruptionSpec{Drop: 10},
		},
		Status: chaosv1beta1.DisruptionStatus{
			DesiredTargetsCount:  2,
			SelectedTargetsCount: 2,
			TargetInjections: chaosv1beta1.TargetInjections{
				"pod-a": {
					chaostypes.DisruptionKindNetworkDisruption: {
						InjectionStatus: chaostypes.DisruptionTargetInjectionStatusNotInjected,
					},
				},
				"pod-b": {
					chaostypes.DisruptionKindNetworkDisruption: {
						InjectionStatus: chaostypes.DisruptionTargetInjectionStatusNotInjected,
					},
				},
			},
		},
	}

	createCalls := 0
	failSecondCreate := true
	injectedErr := errors.New("simulated pod creation failure")
	var createdTargets []string
	var failedTarget string

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&chaosv1beta1.Disruption{}).
		WithObjects(
			disruption,
			newTestTargetPod("pod-a", "app", "node-a", "10.0.0.1"),
			newTestTargetPod("pod-b", "app", "node-b", "10.0.0.2"),
		).
		WithInterceptorFuncs(interceptor.Funcs{
			Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				if pod, ok := obj.(*corev1.Pod); ok {
					createCalls++
					target := pod.Labels[chaostypes.TargetLabel]
					if failSecondCreate && createCalls == 2 {
						failedTarget = target

						return injectedErr
					}

					createdTargets = append(createdTargets, target)
				}

				return c.Create(ctx, obj, opts...)
			},
		}).
		Build()

	instance := &chaosv1beta1.Disruption{}
	if err := cl.Get(ctx, client.ObjectKeyFromObject(disruption), instance); err != nil {
		t.Fatal(err)
	}

	metricsSink := noop.New(logger)
	service, err := services.NewChaosPodService(services.ChaosPodServiceConfig{
		Client:         cl,
		Reader:         cl,
		ChaosNamespace: "chaos-system",
		MetricsSink:    metricsSink,
	})
	if err != nil {
		t.Fatal(err)
	}

	r := &DisruptionReconciler{
		Client:          cl,
		APIReader:       cl,
		BaseLog:         logger,
		log:             logger,
		Scheme:          scheme,
		Recorder:        record.NewFakeRecorder(20),
		MetricsSink:     metricsSink,
		ChaosPodService: service,
	}

	// Pass 1: first target succeeds, second target fails; error is returned and RunCount stays 0.
	err = r.startInjection(ctx, instance)
	if !errors.Is(err, injectedErr) {
		t.Fatalf("first startInjection error = %v, want wrapped %v", err, injectedErr)
	}

	stored := &chaosv1beta1.Disruption{}
	if err := cl.Get(ctx, client.ObjectKeyFromObject(instance), stored); err != nil {
		t.Fatal(err)
	}

	if stored.Status.RunCount != 0 {
		t.Fatalf("stored RunCount after failed pass = %d, want 0", stored.Status.RunCount)
	}

	chaosPods, err := service.GetChaosPodsOfDisruption(ctx, instance, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(chaosPods) != 1 {
		t.Fatalf("expected 1 chaos pod after partial failure, got %d", len(chaosPods))
	}

	firstTarget := chaosPods[0].Labels[chaostypes.TargetLabel]
	firstPodName := chaosPods[0].Name

	// Pass 2 (retry): only the previously failed target gets a chaos pod (the first target is not duplicated),
	// and RunCount increments once to 1.
	failSecondCreate = false
	createCallsBeforeRetry := createCalls
	createdTargets = nil

	if err := cl.Get(ctx, client.ObjectKeyFromObject(instance), instance); err != nil {
		t.Fatal(err)
	}

	if err := r.startInjection(ctx, instance); err != nil {
		t.Fatalf("retry startInjection failed: %v", err)
	}

	if createCalls-createCallsBeforeRetry != 1 || len(createdTargets) != 1 || createdTargets[0] != failedTarget {
		t.Fatalf("retry created targets %v (%d creates), want only %q (1 create)", createdTargets, createCalls-createCallsBeforeRetry, failedTarget)
	}

	chaosPods, err = service.GetChaosPodsOfDisruption(ctx, instance, nil)
	if err != nil {
		t.Fatal(err)
	}

	podsByTarget := map[string][]string{}
	for _, pod := range chaosPods {
		target := pod.Labels[chaostypes.TargetLabel]
		podsByTarget[target] = append(podsByTarget[target], pod.Name)
	}

	if len(chaosPods) != 2 || len(podsByTarget["pod-a"]) != 1 || len(podsByTarget["pod-b"]) != 1 {
		t.Fatalf("chaos pods after retry = %v (%d pods), want 1 pod each for pod-a and pod-b", podsByTarget, len(chaosPods))
	}
	if podsByTarget[firstTarget][0] != firstPodName {
		t.Fatalf("existing chaos pod for %s changed from %q to %q", firstTarget, firstPodName, podsByTarget[firstTarget][0])
	}

	if err := cl.Get(ctx, client.ObjectKeyFromObject(instance), stored); err != nil {
		t.Fatal(err)
	}

	if stored.Status.RunCount != 1 {
		t.Fatalf("stored RunCount after retry = %d, want 1", stored.Status.RunCount)
	}

	// Pass 3: once chaos pods are gone, maxRuns=1 prevents creating any more chaos pods.
	removeChaosPods(t, ctx, cl, chaosPods)

	remainingPods, err := service.GetChaosPodsOfDisruption(ctx, instance, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(remainingPods) != 0 {
		t.Fatalf("expected all chaos pods to be removed before third pass, got %d", len(remainingPods))
	}

	createCallsBeforeThird := createCalls

	if err := r.startInjection(ctx, instance); err != nil {
		t.Fatalf("third startInjection pass failed: %v", err)
	}

	if createCalls != createCallsBeforeThird {
		t.Fatalf("third pass attempted %d pod creations, want 0", createCalls-createCallsBeforeThird)
	}

	chaosPods, err = service.GetChaosPodsOfDisruption(ctx, instance, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(chaosPods) != 0 {
		t.Fatalf("third pass created %d chaos pods, want 0", len(chaosPods))
	}

	if err := cl.Get(ctx, client.ObjectKeyFromObject(instance), stored); err != nil {
		t.Fatal(err)
	}

	if stored.Status.RunCount != 1 {
		t.Fatalf("stored RunCount after third pass = %d, want 1", stored.Status.RunCount)
	}
}
