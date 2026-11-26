// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package eventbroadcaster

import (
	"context"
	"encoding/json"
	"testing"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	notifTypes "github.com/DataDog/chaos-controller/eventnotifier/types"
)

type uidRegressionNotifier struct {
	calls int
}

func (*uidRegressionNotifier) GetNotifierName() string { return "uid-regression" }

func (n *uidRegressionNotifier) Notify(context.Context, client.Object, corev1.Event, notifTypes.NotificationType) error {
	n.calls++
	return nil
}

func TestNotifierSinkDoesNotNotifyReplacementObject(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := v1beta1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	for _, kind := range []string{v1beta1.DisruptionKind, v1beta1.DisruptionCronKind} {
		for _, scenario := range []struct {
			name      string
			eventUID  k8stypes.UID
			snapshot  bool
			wantCalls int
		}{
			{name: "old-event-new-object", eventUID: "old-uid", wantCalls: 0},
			{name: "matching-object", eventUID: "new-uid", wantCalls: 1},
			{name: "original-object-snapshot", eventUID: "old-uid", snapshot: true, wantCalls: 1},
		} {
			t.Run(kind+"/"+scenario.name, func(t *testing.T) {
				meta := metav1.ObjectMeta{Name: "reused-name", Namespace: "default", UID: "new-uid"}
				var obj client.Object
				annotation := v1beta1.EventDisruptionAnnotation
				if kind == v1beta1.DisruptionKind {
					obj = &v1beta1.Disruption{ObjectMeta: meta}
				} else {
					obj = &v1beta1.DisruptionCron{ObjectMeta: meta}
					annotation = v1beta1.EventDisruptionCronAnnotation
				}

				k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(obj).Build()
				notifier := &uidRegressionNotifier{}
				sink := NotifierSink{client: k8sClient, notifier: notifier, logger: zap.NewNop().Sugar()}
				event := &corev1.Event{
					InvolvedObject: corev1.ObjectReference{
						Kind: kind, Namespace: meta.Namespace, Name: meta.Name, UID: scenario.eventUID,
					},
					Type: corev1.EventTypeWarning,
				}

				if scenario.snapshot {
					original := obj.DeepCopyObject().(client.Object)
					original.SetUID(scenario.eventUID)
					encoded, err := json.Marshal(original)
					if err != nil {
						t.Fatal(err)
					}
					event.Annotations = map[string]string{annotation: string(encoded)}
				}

				if _, err := sink.Create(event); err != nil {
					t.Fatal(err)
				}
				if notifier.calls != scenario.wantCalls {
					t.Fatalf("notifications = %d, want %d (event UID %q)", notifier.calls, scenario.wantCalls, scenario.eventUID)
				}
			})
		}
	}
}
