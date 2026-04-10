// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package controllers

import (
	"testing"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/mocks"
	"github.com/DataDog/chaos-controller/targetselector"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetSelectorMatchingTargetsNoTargetsEventDeduplication(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		status      chaostypes.DisruptionInjectionStatus
		expectEvent bool
	}{
		{
			name:        "emits event when disruption is initial",
			status:      chaostypes.DisruptionInjectionStatusInitial,
			expectEvent: true,
		},
		{
			name:        "does not emit event when disruption is not injected",
			status:      chaostypes.DisruptionInjectionStatusNotInjected,
			expectEvent: false,
		},
		{
			name:        "does not emit event when disruption is paused injected",
			status:      chaostypes.DisruptionInjectionStatusPausedInjected,
			expectEvent: false,
		},
		{
			name:        "does not emit event when disruption is paused partially injected",
			status:      chaostypes.DisruptionInjectionStatusPausedPartiallyInjected,
			expectEvent: false,
		},
		{
			name:        "emits event when disruption is injected",
			status:      chaostypes.DisruptionInjectionStatusInjected,
			expectEvent: true,
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			disruption := &chaosv1beta1.Disruption{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "disruption",
					Namespace: "default",
				},
				Spec: chaosv1beta1.DisruptionSpec{
					Level:    chaostypes.DisruptionLevelPod,
					Selector: map[string]string{"app": "demo"},
				},
				Status: chaosv1beta1.DisruptionStatus{
					InjectionStatus: tc.status,
				},
			}

			recorderMock := mocks.NewEventRecorderMock(t)
			targetSelectorMock := targetselector.NewTargetSelectorMock(t)
			targetSelectorMock.EXPECT().GetMatchingPodsOverTotalPods(mock.Anything, disruption).Return(&corev1.PodList{}, 0, nil).Once()

			if tc.expectEvent {
				recorderMock.EXPECT().Event(disruption, mock.Anything, mock.Anything, mock.Anything).Once()
			}

			reconciler := &DisruptionReconciler{
				Recorder:       recorderMock,
				TargetSelector: targetSelectorMock,
				log:            zap.NewNop().Sugar(),
			}

			targets, totalCount, err := reconciler.getSelectorMatchingTargets(disruption)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if targets != nil {
				t.Fatalf("expected nil targets, got %v", targets)
			}

			if totalCount != 0 {
				t.Fatalf("expected total count to be 0, got %d", totalCount)
			}

			if !tc.expectEvent {
				recorderMock.AssertNotCalled(t, "Event", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			}
		})
	}
}
