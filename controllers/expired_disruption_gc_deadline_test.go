// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package controllers

import (
	"context"
	"testing"
	"time"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	metricsnoop "github.com/DataDog/chaos-controller/o11y/metrics/noop"
	tracernoop "github.com/DataDog/chaos-controller/o11y/tracer/noop"
	"github.com/DataDog/chaos-controller/services"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"github.com/DataDog/chaos-controller/watchers"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestExpiredDisruptionRequeuesAtOriginalGCDeadline(t *testing.T) {
	for _, testCase := range []struct {
		name        string
		creationAge time.Duration
		triggers    *chaosv1beta1.DisruptionTriggers
	}{
		{name: "without_triggers", creationAge: time.Hour},
		{
			name:        "with_injection_offset",
			creationAge: 90 * time.Minute,
			triggers: &chaosv1beta1.DisruptionTriggers{
				Inject: chaosv1beta1.DisruptionTrigger{Offset: "30m"},
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			if err := chaosv1beta1.AddToScheme(scheme); err != nil {
				t.Fatal(err)
			}
			if err := corev1.AddToScheme(scheme); err != nil {
				t.Fatal(err)
			}

			count := intstr.FromInt(1)
			disruption := &chaosv1beta1.Disruption{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "expired-disruption",
					Namespace:         "default",
					CreationTimestamp: metav1.NewTime(time.Now().Add(-testCase.creationAge)),
				},
				Spec: chaosv1beta1.DisruptionSpec{
					Count:       &count,
					Selector:    map[string]string{"app": "target"},
					Level:       chaostypes.DisruptionLevelPod,
					Duration:    "1m",
					Triggers:    testCase.triggers,
					CPUPressure: &chaosv1beta1.CPUPressureSpec{},
					Unsafemode:  &chaosv1beta1.UnsafemodeSpec{DisableAll: true},
				},
			}
			gcDelay := time.Hour
			gcDeadline := disruption.TimeToInject().Add(disruption.Spec.Duration.Duration()).Add(gcDelay)
			cl := fake.NewClientBuilder().WithScheme(scheme).
				WithStatusSubresource(&chaosv1beta1.Disruption{}).
				WithObjects(disruption).Build()
			podService, err := services.NewChaosPodService(services.ChaosPodServiceConfig{
				Client: cl, Reader: cl, ChaosNamespace: "chaos-engineering",
			})
			if err != nil {
				t.Fatal(err)
			}
			watcherManager := watchers.NewDisruptionsWatchersManagerMock(t)
			watcherManager.On("CreateAllWatchers", mock.Anything, mock.Anything, nil, nil).Return(nil).Once()
			logger := zap.NewNop().Sugar()
			reconciler := &DisruptionReconciler{
				Client:                     cl,
				APIReader:                  cl,
				Scheme:                     scheme,
				BaseLog:                    logger,
				Recorder:                   record.NewFakeRecorder(10),
				MetricsSink:                metricsnoop.New(logger),
				TracerSink:                 tracernoop.New(logger),
				ChaosPodService:            podService,
				DisruptionsWatchersManager: watcherManager,
				ExpiredDisruptionGCDelay:   &gcDelay,
			}

			result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
				NamespacedName: types.NamespacedName{Name: disruption.Name, Namespace: disruption.Namespace},
			})
			if err != nil {
				t.Fatal(err)
			}
			remaining := time.Until(gcDeadline)
			if result.RequeueAfter <= 0 || result.RequeueAfter < remaining-time.Second || result.RequeueAfter > remaining+time.Second {
				t.Fatalf("RequeueAfter = %s; want remaining time until original GC deadline (approximately %s), not a fresh %s", result.RequeueAfter, remaining, gcDelay)
			}
		})
	}
}
