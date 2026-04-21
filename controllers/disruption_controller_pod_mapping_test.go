// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package controllers

import (
	"testing"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	chaostypes "github.com/DataDog/chaos-controller/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestMapPodToDisruptionRequests(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		pod      *corev1.Pod
		expected []reconcile.Request
	}{
		{
			name:     "returns empty list for nil pod",
			pod:      nil,
			expected: nil,
		},
		{
			name: "returns empty list for pod without disruption labels",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "target",
					Namespace: "target-namespace",
				},
			},
			expected: nil,
		},
		{
			name: "returns empty list for pod missing disruption namespace label",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "chaos-pod",
					Namespace: "chaos-engineering",
					Labels: map[string]string{
						chaostypes.DisruptionNameLabel: "disruption-a",
					},
				},
			},
			expected: nil,
		},
		{
			name: "maps pod with disruption labels to reconcile request",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "chaos-pod",
					Namespace: "chaos-engineering",
					Labels: map[string]string{
						chaostypes.DisruptionNameLabel:      "disruption-a",
						chaostypes.DisruptionNamespaceLabel: "ns-a",
					},
				},
			},
			expected: []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Name:      "disruption-a",
						Namespace: "ns-a",
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result := mapPodToDisruptionRequests(testCase.pod)

			if len(result) != len(testCase.expected) {
				t.Fatalf("unexpected number of requests: got %d, expected %d", len(result), len(testCase.expected))
			}

			for i := range testCase.expected {
				if result[i] != testCase.expected[i] {
					t.Fatalf("unexpected request at index %d: got %#v, expected %#v", i, result[i], testCase.expected[i])
				}
			}
		})
	}
}

func TestShouldTriggerReconcile(t *testing.T) {
	t.Parallel()

	t.Run("returns true for disruption objects", func(t *testing.T) {
		t.Parallel()
		if !shouldTriggerReconcile(&chaosv1beta1.Disruption{}) {
			t.Fatal("expected disruption object to trigger reconcile")
		}
	})

	t.Run("returns false for pod without disruption labels", func(t *testing.T) {
		t.Parallel()
		if shouldTriggerReconcile(&corev1.Pod{}) {
			t.Fatal("expected pod without disruption labels to not trigger reconcile")
		}
	})

	t.Run("returns true for pod with disruption labels", func(t *testing.T) {
		t.Parallel()
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					chaostypes.DisruptionNameLabel:      "disruption-a",
					chaostypes.DisruptionNamespaceLabel: "ns-a",
				},
			},
		}

		if !shouldTriggerReconcile(pod) {
			t.Fatal("expected pod with disruption labels to trigger reconcile")
		}
	})
}
