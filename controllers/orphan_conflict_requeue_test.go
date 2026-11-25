// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package controllers_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	chaos "github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/controllers"
	"github.com/DataDog/chaos-controller/o11y/metrics"
	"github.com/DataDog/chaos-controller/services"
)

type orphanConflictService struct {
	services.ChaosPodService
	err error
}

func (s orphanConflictService) HandleOrphanedChaosPods(context.Context, ctrl.Request) error {
	return s.err
}

func TestOrphanCleanupConflictStillRequestsReconciliation(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := chaos.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	key := types.NamespacedName{Name: "disruption", Namespace: "workload"}
	sink := metrics.NewSinkMock(t)
	sink.EXPECT().MetricReconcile().Return(nil).Once()
	sink.EXPECT().MetricReconcileDuration(mock.Anything, mock.Anything).Return(nil).Once()
	conflict := apierrors.NewConflict(schema.GroupResource{Resource: "pods"}, "orphan",
		errors.New("the object has been modified; please apply your changes to the latest version and try again"))

	// A stale cached NotFound dispatches to orphan cleanup. The uncached reader
	// still sees the owner, avoiding unrelated watcher teardown in this regression.
	r := controllers.DisruptionReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).Build(),
		APIReader: fake.NewClientBuilder().WithScheme(scheme).WithObjects(&chaos.Disruption{
			ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: key.Namespace},
		}).Build(),
		BaseLog: zap.NewNop().Sugar(), MetricsSink: sink,
		ChaosPodService: orphanConflictService{err: conflict},
	}
	result, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: key})
	if err != nil && !apierrors.IsConflict(err) {
		t.Fatalf("unexpected reconciliation error: %v", err)
	}
	if err == nil && !result.Requeue && result.RequeueAfter <= 0 {
		t.Fatal("conflict was suppressed without scheduling another cleanup attempt")
	}
}
