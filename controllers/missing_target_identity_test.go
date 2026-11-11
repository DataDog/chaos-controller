// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package controllers

import (
	"context"
	"fmt"
	"testing"
	"time"

	chaos "github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/o11y/metrics/noop"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestMissingTargetClockBelongsToTargetResource(t *testing.T) {
	for _, parentKind := range []string{"cron", "rollout"} {
		for _, change := range []string{"name", "kind", "unchanged"} {
			t.Run(fmt.Sprintf("%s/%s", parentKind, change), func(t *testing.T) {
				ctx := context.Background()
				scheme := runtime.NewScheme()
				if err := chaos.AddToScheme(scheme); err != nil {
					t.Fatal(err)
				}
				if err := appsv1.AddToScheme(scheme); err != nil {
					t.Fatal(err)
				}
				count := intstr.FromInt(1)
				template := chaos.DisruptionSpec{Count: &count, Level: "pod", Duration: "1m", Network: &chaos.NetworkDisruptionSpec{Drop: 100}}
				oldTarget := chaos.TargetResourceSpec{Kind: "deployment", Name: "old"}
				metadata := metav1.ObjectMeta{Name: "scheduled", Namespace: "ns"}
				var owner client.Object
				if parentKind == "cron" {
					owner = &chaos.DisruptionCron{ObjectMeta: metadata, Spec: chaos.DisruptionCronSpec{Schedule: "@every 24h", TargetResource: oldTarget, DisruptionTemplate: template}}
				} else {
					owner = &chaos.DisruptionRollout{ObjectMeta: metadata, Spec: chaos.DisruptionRolloutSpec{TargetResource: oldTarget, DisruptionTemplate: template}}
				}
				cl := fake.NewClientBuilder().WithScheme(scheme).
					WithStatusSubresource(&chaos.DisruptionCron{}, &chaos.DisruptionRollout{}).
					WithObjects(owner).Build()
				logger := zap.NewNop().Sugar()
				cronReconciler := &DisruptionCronReconciler{Client: cl, log: logger, MetricsSink: noop.New(logger), Recorder: record.NewFakeRecorder(10), TargetResourceMissingThreshold: time.Hour}
				rolloutReconciler := &DisruptionRolloutReconciler{Client: cl, log: logger, MetricsSink: noop.New(logger), TargetResourceMissingThreshold: time.Hour}
				updateMissing := func() (bool, bool, error) {
					if cron, ok := owner.(*chaos.DisruptionCron); ok {
						return cronReconciler.updateTargetResourcePreviouslyMissing(ctx, cron)
					}
					return rolloutReconciler.updateTargetResourcePreviouslyMissing(ctx, owner.(*chaos.DisruptionRollout))
				}
				if _, deleted, err := updateMissing(); err != nil || deleted {
					t.Fatalf("first missing observation: deleted=%t err=%v", deleted, err)
				}
				expired := metav1.NewTime(time.Now().Add(-2 * time.Hour))
				switch obj := owner.(type) {
				case *chaos.DisruptionCron:
					obj.Status.TargetResourcePreviouslyMissing = &expired
				case *chaos.DisruptionRollout:
					obj.Status.TargetResourcePreviouslyMissing = &expired
				}
				if err := cl.Status().Update(ctx, owner); err != nil {
					t.Fatal(err)
				}
				newTarget := oldTarget
				if change == "name" {
					newTarget.Name = "new"
				} else if change == "kind" {
					newTarget.Kind = "statefulset"
				}
				switch obj := owner.(type) {
				case *chaos.DisruptionCron:
					obj.Spec.TargetResource = newTarget
				case *chaos.DisruptionRollout:
					obj.Spec.TargetResource = newTarget
				}
				if err := cl.Update(ctx, owner); err != nil {
					t.Fatal(err)
				}
				if err := cl.Get(ctx, client.ObjectKeyFromObject(owner), owner); err != nil {
					t.Fatal(err)
				}
				found, deleted, err := updateMissing()
				if err != nil || found {
					t.Fatalf("missing-target update: found=%t err=%v", found, err)
				}
				if wantDeleted := change == "unchanged"; deleted != wantDeleted {
					t.Fatalf("target change=%s: deleted=%t, want %t; old missing duration must not apply to a different target", change, deleted, wantDeleted)
				}
				if !deleted {
					if err := cl.Get(ctx, client.ObjectKeyFromObject(owner), owner); err != nil {
						t.Fatal(err)
					}
					var since *metav1.Time
					switch obj := owner.(type) {
					case *chaos.DisruptionCron:
						since = obj.Status.TargetResourcePreviouslyMissing
					case *chaos.DisruptionRollout:
						since = obj.Status.TargetResourcePreviouslyMissing
					}
					if since == nil || !since.After(expired.Time) {
						t.Fatal("new target did not receive a fresh missing-target clock")
					}
				}
			})
		}
	}
}
