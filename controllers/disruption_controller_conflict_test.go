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

	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	metricsnoop "github.com/DataDog/chaos-controller/o11y/metrics/noop"
	tracernoop "github.com/DataDog/chaos-controller/o11y/tracer/noop"
	"github.com/DataDog/chaos-controller/services"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"github.com/DataDog/chaos-controller/watchers"
)

// noChaosPods is a ChaosPodService whose disruption has no chaos pods left.
// Only the lookup is implemented; the reconcile path under test calls nothing else.
type noChaosPods struct{ services.ChaosPodService }

func (noChaosPods) GetChaosPodsOfDisruption(context.Context, *chaosv1beta1.Disruption, labels.Set) ([]corev1.Pod, error) {
	return nil, nil
}

// A deleted disruption whose chaos pods are gone records CleanedAt before its
// finalizer can be removed. When that status update hits a 409, nothing has
// been persisted and no later event is due, so the reconcile must be retried
// rather than reported as done; otherwise the finalizer stays until some
// unrelated event arrives.
func TestReconcileRetriesStatusUpdateConflict(t *testing.T) {
	log := zap.NewNop().Sugar()
	scheme := runtime.NewScheme()
	if err := chaosv1beta1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	count := intstr.FromInt(1)
	deleted := metav1.NewTime(time.Now())
	disruption := &chaosv1beta1.Disruption{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "cpu",
			Namespace:         "team",
			Finalizers:        []string{chaostypes.DisruptionFinalizer},
			DeletionTimestamp: &deleted,
		},
		Spec: chaosv1beta1.DisruptionSpec{
			Count:       &count,
			Level:       chaostypes.DisruptionLevelPod,
			CPUPressure: &chaosv1beta1.CPUPressureSpec{},
		},
	}

	conflicts := 0
	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(disruption).
		WithStatusSubresource(disruption).
		WithInterceptorFuncs(interceptor.Funcs{
			SubResourceUpdate: func(ctx context.Context, c client.Client, subResourceName string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
				conflicts++

				// the API server optimistic-lock message
				return apierrors.NewConflict(chaosv1beta1.GroupVersion.WithResource("disruptions").GroupResource(), obj.GetName(),
					errors.New("the object has been modified; please apply your changes to the latest version and try again"))
			},
		}).
		Build()

	watchersManager := watchers.NewDisruptionsWatchersManagerMock(t)
	watchersManager.EXPECT().CreateAllWatchers(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	reconciler := &DisruptionReconciler{
		Client:                     k8sClient,
		APIReader:                  k8sClient,
		BaseLog:                    log,
		Scheme:                     scheme,
		Recorder:                   record.NewFakeRecorder(10),
		MetricsSink:                metricsnoop.New(log),
		TracerSink:                 tracernoop.New(log),
		ChaosPodService:            noChaosPods{},
		DisruptionsWatchersManager: watchersManager,
		DisruptionsDeletionTimeout: time.Hour,
		FinalizerDeletionDelay:     time.Second,
	}

	res, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(disruption)})

	if conflicts != 1 {
		t.Fatalf("expected the CleanedAt status update to be attempted once, got %d", conflicts)
	}

	if err == nil && res.IsZero() {
		t.Fatal("the conflicting reconcile was reported as done: no error and no requeue")
	}
}
