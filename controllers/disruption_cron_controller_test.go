// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package controllers

import (
	"testing"
	"time"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/o11y/metrics/noop"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDisruptionCronIntervalSchedule(t *testing.T) {
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name             string
		schedule         string
		alreadyScheduled bool
		elapsed          time.Duration
		wantMissed       time.Time
		wantNext         time.Time
	}{
		{name: "creation anchor", schedule: "@every 1h", elapsed: 50 * time.Minute, wantNext: start.Add(time.Hour)},
		{name: "last schedule anchor", schedule: "@every 1h", alreadyScheduled: true, elapsed: 50 * time.Minute, wantNext: start.Add(time.Hour)},
		{name: "missed intervals", schedule: "@every 1h", elapsed: 170 * time.Minute, wantMissed: start.Add(2 * time.Hour), wantNext: start.Add(3 * time.Hour)},
		{name: "calendar schedule", schedule: "0 * * * *", elapsed: 50 * time.Minute, wantNext: start.Add(time.Hour)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logger := zap.NewNop().Sugar()
			reconciler := &DisruptionCronReconciler{log: logger, MetricsSink: noop.New(logger)}
			instance := &chaosv1beta1.DisruptionCron{
				ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.NewTime(start)},
				Spec:       chaosv1beta1.DisruptionCronSpec{Schedule: tc.schedule},
			}
			if tc.alreadyScheduled {
				instance.CreationTimestamp = metav1.NewTime(start.Add(-2 * time.Hour))
				instance.Status.LastScheduleTime = &metav1.Time{Time: start}
			}
			missed, next, err := reconciler.getNextSchedule(instance, start.Add(tc.elapsed))
			if err != nil {
				t.Fatal(err)
			}
			if !missed.Equal(tc.wantMissed) {
				t.Errorf("missed = %s, want %s", missed, tc.wantMissed)
			}
			if !next.Equal(tc.wantNext) {
				t.Errorf("next = %s, want %s", next, tc.wantNext)
			}
		})
	}
}
