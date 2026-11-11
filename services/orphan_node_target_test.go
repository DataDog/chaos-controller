// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package services_test

import (
	"context"
	"testing"

	cLog "github.com/DataDog/chaos-controller/log"
	"github.com/DataDog/chaos-controller/o11y/metrics/noop"
	"github.com/DataDog/chaos-controller/services"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func TestOrphanCleanupLooksUpTheActualTargetKind(t *testing.T) {
	for _, tc := range []struct {
		name         string
		level        chaostypes.DisruptionLevel
		nodeExists   bool
		podExists    bool
		wantRetained bool
	}{
		{"live node without namesake pod", chaostypes.DisruptionLevelNode, true, false, true},
		{"missing node with unrelated namesake pod", chaostypes.DisruptionLevelNode, false, true, false},
		{"live pod", chaostypes.DisruptionLevelPod, false, true, true},
		{"missing pod with namesake node", chaostypes.DisruptionLevelPod, true, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			if err := corev1.AddToScheme(scheme); err != nil {
				t.Fatal(err)
			}
			logger := zap.NewNop().Sugar()
			ctx := cLog.WithLogger(context.Background(), logger)
			injector := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "orphan", Namespace: "chaos-system",
					Finalizers: []string{chaostypes.ChaosPodFinalizer},
					Labels: map[string]string{
						chaostypes.DisruptionNameLabel:      "gone",
						chaostypes.DisruptionNamespaceLabel: "app",
						chaostypes.TargetLabel:              "target",
						chaostypes.DisruptionKindLabel:      string(chaostypes.DisruptionKindNetworkDisruption),
					},
				},
				Spec: corev1.PodSpec{
					NodeName:   "target",
					Containers: []corev1.Container{{Name: "injector", Args: []string{"network-disruption", "--level", string(tc.level)}}},
				},
				Status: corev1.PodStatus{Phase: corev1.PodFailed},
			}
			objects := []client.Object{injector}
			if tc.nodeExists {
				objects = append(objects, &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "target"}})
			}
			if tc.podExists {
				objects = append(objects, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "target", Namespace: "app"}})
			}
			cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
			service, err := services.NewChaosPodService(services.ChaosPodServiceConfig{
				Client: cl, Reader: cl, ChaosNamespace: "chaos-system", MetricsSink: noop.New(logger),
			})
			if err != nil {
				t.Fatal(err)
			}
			if err := service.HandleOrphanedChaosPods(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: "gone", Namespace: "app"}}); err != nil {
				t.Fatal(err)
			}
			var got corev1.Pod
			err = cl.Get(ctx, client.ObjectKeyFromObject(injector), &got)
			if tc.wantRetained {
				if err != nil {
					t.Fatalf("removed orphan despite a live %s target: %v", tc.level, err)
				}
				if !controllerutil.ContainsFinalizer(&got, chaostypes.ChaosPodFinalizer) {
					t.Fatal("removed cleanup finalizer despite a live target")
				}
			} else if !apierrors.IsNotFound(err) {
				t.Fatalf("missing target should permit orphan removal; got %v", err)
			}
		})
	}
}
