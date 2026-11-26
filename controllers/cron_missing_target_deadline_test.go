// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package controllers

import (
	"context"
	"fmt"
	"testing"
	"time"

	chaos "github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/o11y/metrics/noop"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCronMissingTargetRequeuesByExpiration(t *testing.T) {
	for _, paused := range []bool{false, true} {
		t.Run(fmt.Sprintf("paused=%t", paused), func(t *testing.T) {
			ctx := context.Background()
			scheme := runtime.NewScheme()
			if err := chaos.AddToScheme(scheme); err != nil {
				t.Fatal(err)
			}
			if err := appsv1.AddToScheme(scheme); err != nil {
				t.Fatal(err)
			}
			now := time.Now()
			missingSince := metav1.NewTime(now.Add(-30 * time.Minute))
			count := intstr.FromInt(1)
			cron := &chaos.DisruptionCron{
				ObjectMeta: metav1.ObjectMeta{
					Name: "missing-target", Namespace: "ns",
					CreationTimestamp: metav1.NewTime(now.Add(-time.Hour)),
					Finalizers:        []string{chaostypes.DisruptionCronFinalizer},
				},
				Spec: chaos.DisruptionCronSpec{
					Schedule: "@every 24h", Paused: paused,
					TargetResource: chaos.TargetResourceSpec{Kind: "deployment", Name: "absent"},
					DisruptionTemplate: chaos.DisruptionSpec{
						Count: &count, Level: "pod", Duration: "1m",
						Network: &chaos.NetworkDisruptionSpec{Drop: 100},
					},
				},
				Status: chaos.DisruptionCronStatus{TargetResourcePreviouslyMissing: &missingSince},
			}
			cl := fake.NewClientBuilder().WithScheme(scheme).
				WithStatusSubresource(&chaos.DisruptionCron{}).WithObjects(cron).Build()
			logger := zap.NewNop().Sugar()
			r := &DisruptionCronReconciler{
				Client: cl, Scheme: scheme, BaseLog: logger,
				Recorder: record.NewFakeRecorder(10), MetricsSink: noop.New(logger),
				TargetResourceMissingThreshold: time.Hour,
			}
			req := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(cron)}
			result, err := r.Reconcile(ctx, req)
			if err != nil {
				t.Fatal(err)
			}
			if result.RequeueAfter <= 0 || result.RequeueAfter > 30*time.Minute {
				t.Fatalf("missing target expires within 30m, but RequeueAfter=%s (paused=%t)", result.RequeueAfter, paused)
			}
			current := &chaos.DisruptionCron{}
			if err := cl.Get(ctx, req.NamespacedName, current); err != nil {
				t.Fatal(err)
			}
			if !current.DeletionTimestamp.IsZero() {
				t.Fatal("cron was deleted before the missing-target threshold")
			}
			current.Status.TargetResourcePreviouslyMissing = &metav1.Time{Time: now.Add(-2 * time.Hour)}
			if err := cl.Status().Update(ctx, current); err != nil {
				t.Fatal(err)
			}
			if _, err := r.Reconcile(ctx, req); err != nil {
				t.Fatal(err)
			}
			if err := cl.Get(ctx, req.NamespacedName, current); err != nil {
				t.Fatal(err)
			}
			if current.DeletionTimestamp.IsZero() {
				t.Fatal("cron was not deleted after the missing-target threshold")
			}
		})
	}
}
