// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package controllers

import (
	"context"
	"testing"
	"time"

	chaos "github.com/DataDog/chaos-controller/api/v1beta1"
	clog "github.com/DataDog/chaos-controller/log"
	"github.com/DataDog/chaos-controller/o11y/metrics/noop"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestScheduledExpiryDoesNotDeleteConcurrentlyRetargetedParent(t *testing.T) {
	for _, kind := range []string{"cron", "rollout"} {
		t.Run(kind, func(t *testing.T) {
			scheme := runtime.NewScheme()
			if err := chaos.AddToScheme(scheme); err != nil {
				t.Fatal(err)
			}
			if err := appsv1.AddToScheme(scheme); err != nil {
				t.Fatal(err)
			}

			logger := zap.NewNop().Sugar()
			ctx := clog.WithLogger(context.Background(), logger)
			missingSince := metav1.NewTime(time.Now().Add(-2 * time.Hour))
			metadata := metav1.ObjectMeta{
				Name: "schedule", Namespace: "ns", UID: types.UID("schedule-uid"),
			}
			oldTarget := chaos.TargetResourceSpec{Kind: "deployment", Name: "absent"}
			healthy := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "healthy", Namespace: metadata.Namespace},
			}

			var parent client.Object
			if kind == "cron" {
				metadata.Finalizers = []string{chaostypes.DisruptionCronFinalizer}
				parent = &chaos.DisruptionCron{
					ObjectMeta: metadata,
					Spec: chaos.DisruptionCronSpec{
						Schedule: "@every 24h", Paused: true, TargetResource: oldTarget,
					},
					Status: chaos.DisruptionCronStatus{TargetResourcePreviouslyMissing: &missingSince},
				}
			} else {
				parent = &chaos.DisruptionRollout{
					ObjectMeta: metadata,
					Spec:       chaos.DisruptionRolloutSpec{TargetResource: oldTarget},
					Status:     chaos.DisruptionRolloutStatus{TargetResourcePreviouslyMissing: &missingSince},
				}
			}

			deleteCalls := 0
			cl := fake.NewClientBuilder().WithScheme(scheme).
				WithStatusSubresource(&chaos.DisruptionCron{}, &chaos.DisruptionRollout{}).
				WithObjects(parent, healthy).
				WithInterceptorFuncs(interceptor.Funcs{
					Delete: func(ctx context.Context, underlying client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
						deleteCalls++
						// Commit a user retargeting update after the reconciler has checked
						// the missing old target, but before its delete reaches storage.
						current := parent.DeepCopyObject().(client.Object)
						if err := underlying.Get(ctx, client.ObjectKeyFromObject(parent), current); err != nil {
							return err
						}
						switch p := current.(type) {
						case *chaos.DisruptionCron:
							p.Spec.TargetResource.Name = healthy.Name
						case *chaos.DisruptionRollout:
							p.Spec.TargetResource.Name = healthy.Name
						}
						if err := underlying.Update(ctx, current); err != nil {
							return err
						}
						return underlying.Delete(ctx, obj, opts...)
					},
				}).Build()

			var reconcile func(context.Context, ctrl.Request) (ctrl.Result, error)
			if kind == "cron" {
				r := &DisruptionCronReconciler{
					Client: cl, Scheme: scheme, BaseLog: logger,
					Recorder: record.NewFakeRecorder(10), MetricsSink: noop.New(logger),
					TargetResourceMissingThreshold: time.Hour,
				}
				reconcile = r.Reconcile
			} else {
				r := &DisruptionRolloutReconciler{
					Client: cl, Scheme: scheme, BaseLog: logger, MetricsSink: noop.New(logger),
					TargetResourceMissingThreshold: time.Hour,
				}
				reconcile = r.Reconcile
			}

			req := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(parent)}
			if _, err := reconcile(ctx, req); !apierrors.IsConflict(err) {
				t.Fatalf("expiry delete must conflict with the committed retarget, got %v", err)
			}
			if deleteCalls != 1 {
				t.Fatalf("expected one expiry delete attempt, got %d", deleteCalls)
			}

			current := parent.DeepCopyObject().(client.Object)
			if err := cl.Get(ctx, req.NamespacedName, current); err != nil {
				t.Fatal(err)
			}
			if !current.GetDeletionTimestamp().IsZero() {
				t.Fatal("concurrently retargeted parent was marked for deletion")
			}

			// The normal error retry must observe the new healthy target, retain
			// the parent, and clear the old missing-target timestamp.
			if _, err := reconcile(ctx, req); err != nil {
				t.Fatal(err)
			}
			if err := cl.Get(ctx, req.NamespacedName, current); err != nil {
				t.Fatal(err)
			}
			switch p := current.(type) {
			case *chaos.DisruptionCron:
				if p.Spec.TargetResource.Name != healthy.Name || p.Status.TargetResourcePreviouslyMissing != nil {
					t.Fatalf("cron did not recover to the healthy target: %+v", p)
				}
			case *chaos.DisruptionRollout:
				if p.Spec.TargetResource.Name != healthy.Name || p.Status.TargetResourcePreviouslyMissing != nil {
					t.Fatalf("rollout did not recover to the healthy target: %+v", p)
				}
			}
			if deleteCalls != 1 {
				t.Fatalf("healthy parent received another delete attempt: %d", deleteCalls)
			}
		})
	}
}
