// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package controllers

import (
	"context"
	"errors"
	"testing"
	"time"

	chaos "github.com/DataDog/chaos-controller/api/v1beta1"
	clog "github.com/DataDog/chaos-controller/log"
	"github.com/DataDog/chaos-controller/o11y/metrics/noop"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestScheduledDisruptionChildListErrorIsReturned(t *testing.T) {
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
			target := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ns"},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "app"}},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "app"}},
						Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "app:v1"}}},
					},
				},
			}
			count := intstr.FromInt(1)
			template := chaos.DisruptionSpec{
				Count: &count, Level: "pod", Duration: "1m",
				Network: &chaos.NetworkDisruptionSpec{Drop: 100},
			}
			targetRef := chaos.TargetResourceSpec{Kind: "deployment", Name: target.Name}
			metadata := metav1.ObjectMeta{Name: "schedule", Namespace: "ns"}
			var owner client.Object
			if kind == "cron" {
				metadata.Finalizers = []string{chaostypes.DisruptionCronFinalizer}
				owner = &chaos.DisruptionCron{ObjectMeta: metadata, Spec: chaos.DisruptionCronSpec{
					Schedule: "@every 24h", TargetResource: targetRef, DisruptionTemplate: template,
				}}
			} else {
				owner = &chaos.DisruptionRollout{ObjectMeta: metadata, Spec: chaos.DisruptionRolloutSpec{
					TargetResource: targetRef, DisruptionTemplate: template,
				}}
			}
			listErr := apierrors.NewServiceUnavailable("child list temporarily unavailable")
			listCalls := 0
			cl := fake.NewClientBuilder().WithScheme(scheme).
				WithStatusSubresource(&chaos.DisruptionCron{}, &chaos.DisruptionRollout{}).
				WithObjects(owner, target).
				WithInterceptorFuncs(interceptor.Funcs{
					List: func(ctx context.Context, underlying client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
						if _, ok := list.(*chaos.DisruptionList); ok {
							listCalls++
							return listErr
						}
						return underlying.List(ctx, list, opts...)
					},
				}).Build()
			var reconcile func(context.Context, ctrl.Request) (ctrl.Result, error)
			if kind == "cron" {
				r := &DisruptionCronReconciler{
					Client: cl, Scheme: scheme, BaseLog: logger, Recorder: record.NewFakeRecorder(10),
					MetricsSink: noop.New(logger), TargetResourceMissingThreshold: time.Hour,
				}
				reconcile = r.Reconcile
			} else {
				r := &DisruptionRolloutReconciler{
					Client: cl, Scheme: scheme, BaseLog: logger,
					MetricsSink: noop.New(logger), TargetResourceMissingThreshold: time.Hour,
				}
				reconcile = r.Reconcile
			}
			result, err := reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(owner)})
			if listCalls != 1 {
				t.Fatalf("expected one child list call, got %d", listCalls)
			}
			if !errors.Is(err, listErr) {
				t.Fatalf("child list failure was swallowed: result=%+v err=%v", result, err)
			}
		})
	}
}
