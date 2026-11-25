// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	cLog "github.com/DataDog/chaos-controller/log"
	"github.com/DataDog/chaos-controller/o11y/metrics"
	"github.com/DataDog/chaos-controller/services"
	chaostypes "github.com/DataDog/chaos-controller/types"
)

func TestOrphanCleanupReturnsAPIErrorsAndCanRetry(t *testing.T) {
	for _, operation := range []string{"get", "update", "delete"} {
		t.Run(operation, func(t *testing.T) {
			ctx := cLog.WithLogger(context.Background(), zap.NewNop().Sugar())
			scheme := runtime.NewScheme()
			if err := corev1.AddToScheme(scheme); err != nil {
				t.Fatal(err)
			}

			newOrphan := func(name, target string) *corev1.Pod {
				return &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: name, Namespace: "chaos",
						Finalizers: []string{chaostypes.ChaosPodFinalizer},
						Labels: map[string]string{
							chaostypes.DisruptionNameLabel:      "gone-disruption",
							chaostypes.DisruptionNamespaceLabel: "workload",
							chaostypes.TargetLabel:              target,
							chaostypes.DisruptionKindLabel:      string(chaostypes.DisruptionKindNetworkDisruption),
						},
					},
					Spec:   corev1.PodSpec{Containers: []corev1.Container{{Name: "injector", Args: []string{"network-disruption", "--level", "pod"}}}},
					Status: corev1.PodStatus{Phase: corev1.PodFailed},
				}
			}

			injectedErr := errors.New("temporary API outage")
			faultDelivered := false
			c := fake.NewClientBuilder().WithScheme(scheme).
				WithObjects(newOrphan("orphan-a", "target-a"), newOrphan("orphan-b", "target-b")).
				WithInterceptorFuncs(interceptor.Funcs{
					Get: func(ctx context.Context, underlying client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
						if operation == "get" && key.Name == "target-a" && !faultDelivered {
							faultDelivered = true
							return injectedErr
						}
						return underlying.Get(ctx, key, obj, opts...)
					},
					Update: func(ctx context.Context, underlying client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
						if operation == "update" && obj.GetName() == "orphan-a" && !faultDelivered {
							faultDelivered = true
							return injectedErr
						}
						return underlying.Update(ctx, obj, opts...)
					},
					Delete: func(ctx context.Context, underlying client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
						if operation == "delete" && obj.GetName() == "orphan-a" && !faultDelivered {
							faultDelivered = true
							return injectedErr
						}
						return underlying.Delete(ctx, obj, opts...)
					},
				}).Build()
			sink := metrics.NewSinkMock(t)
			sink.EXPECT().MetricOrphanFound(mock.Anything).Return(nil)
			svc, err := services.NewChaosPodService(services.ChaosPodServiceConfig{
				Client: c, Reader: c, ChaosNamespace: "chaos", MetricsSink: sink,
			})
			if err != nil {
				t.Fatal(err)
			}
			req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "gone-disruption", Namespace: "workload"}}

			if err := svc.HandleOrphanedChaosPods(ctx, req); !errors.Is(err, injectedErr) {
				t.Fatalf("cleanup must return its API error to request a retry; got %v", err)
			}
			if !faultDelivered {
				t.Fatal("did not reach the requested API failure")
			}
			if err := c.Get(ctx, client.ObjectKey{Name: "orphan-a", Namespace: "chaos"}, &corev1.Pod{}); err != nil {
				t.Fatalf("failed orphan should remain available for retry: %v", err)
			}
			if err := c.Get(ctx, client.ObjectKey{Name: "orphan-b", Namespace: "chaos"}, &corev1.Pod{}); !apierrors.IsNotFound(err) {
				t.Fatalf("a failed orphan must not prevent sibling cleanup: %v", err)
			}
			if err := svc.HandleOrphanedChaosPods(ctx, req); err != nil {
				t.Fatalf("retry should succeed after the API recovers: %v", err)
			}
			if err := c.Get(ctx, client.ObjectKey{Name: "orphan-a", Namespace: "chaos"}, &corev1.Pod{}); !apierrors.IsNotFound(err) {
				t.Fatalf("retry should delete the remaining orphan: %v", err)
			}
		})
	}
}
