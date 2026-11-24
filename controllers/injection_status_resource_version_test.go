// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package controllers

import (
	"context"
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
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCreateChaosPodsRetainsStatusResourceVersion(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := chaosv1beta1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	logger := zap.NewNop().Sugar()
	ctx := cLog.WithLogger(context.Background(), logger)
	disruption := &chaosv1beta1.Disruption{
		ObjectMeta: metav1.ObjectMeta{Name: "network", Namespace: "app", CreationTimestamp: metav1.Now()},
		Spec: chaosv1beta1.DisruptionSpec{
			Level:    chaostypes.DisruptionLevelNode,
			Duration: "1h",
			Network:  &chaosv1beta1.NetworkDisruptionSpec{Drop: 10},
		},
	}
	cl := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(&chaosv1beta1.Disruption{}).
		WithObjects(disruption,
			&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-a"}},
			&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-b"}},
		).Build()
	instance := &chaosv1beta1.Disruption{}
	if err := cl.Get(ctx, client.ObjectKeyFromObject(disruption), instance); err != nil {
		t.Fatal(err)
	}
	// Model the transient cloud expansion that must survive status writes.
	instance.Spec.Network.Hosts = []chaosv1beta1.NetworkDisruptionHostSpec{{Host: "192.0.2.0/24"}}
	metricsSink := noop.New(logger)
	service, err := services.NewChaosPodService(services.ChaosPodServiceConfig{
		Client: cl, Reader: cl, ChaosNamespace: "chaos-system", MetricsSink: metricsSink,
	})
	if err != nil {
		t.Fatal(err)
	}
	r := &DisruptionReconciler{
		Client: cl, APIReader: cl, BaseLog: logger, log: logger, Scheme: scheme,
		Recorder: record.NewFakeRecorder(20), MetricsSink: metricsSink, ChaosPodService: service,
	}
	for _, target := range []string{"node-a", "node-b"} {
		if err := r.createChaosPods(ctx, instance, target); err != nil {
			t.Fatalf("creating injectors for %s with the same reconciliation instance: %v", target, err)
		}
	}
	if len(instance.Spec.Network.Hosts) != 1 || instance.Spec.Network.Hosts[0].Host != "192.0.2.0/24" {
		t.Fatal("status write discarded transient cloud host expansion")
	}
	stored := &chaosv1beta1.Disruption{}
	if err := cl.Get(ctx, client.ObjectKeyFromObject(instance), stored); err != nil {
		t.Fatal(err)
	}
	if stored.Status.RunCount != 2 {
		t.Fatalf("lost successful creation accounting: RunCount = %d, want 2", stored.Status.RunCount)
	}
	if instance.ResourceVersion != stored.ResourceVersion {
		t.Fatalf("working resourceVersion %q differs from persisted %q", instance.ResourceVersion, stored.ResourceVersion)
	}
	if err := cl.Status().Update(ctx, instance); err != nil {
		t.Fatalf("later status update conflicts with this reconciliation's own write: %v", err)
	}
}
