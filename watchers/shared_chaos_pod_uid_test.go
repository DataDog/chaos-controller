// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package watchers_test

import (
	"strings"
	"testing"

	chaosapi "github.com/DataDog/chaos-controller/api"
	"github.com/DataDog/chaos-controller/api/v1beta1"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"github.com/DataDog/chaos-controller/watchers"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestSharedChaosPodHandlerDoesNotNotifyReplacementDisruption(t *testing.T) {
	for _, tc := range []struct {
		name      string
		podUID    string
		inlineUID bool
		wantEvent bool
	}{
		{name: "old disruption pod", podUID: "old-disruption", wantEvent: false},
		{name: "current disruption pod", podUID: "current-disruption", wantEvent: true},
		{name: "old disruption inline flag", podUID: "old-disruption", inlineUID: true, wantEvent: false},
		{name: "current disruption inline flag", podUID: "current-disruption", inlineUID: true, wantEvent: true},
		{name: "legacy missing UID", wantEvent: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			if err := v1beta1.AddToScheme(scheme); err != nil {
				t.Fatal(err)
			}
			current := &v1beta1.Disruption{ObjectMeta: metav1.ObjectMeta{
				Name: "same-name", Namespace: "app", UID: types.UID("current-disruption"),
			}}
			reader := fake.NewClientBuilder().WithScheme(scheme).WithObjects(current).Build()
			recorder := record.NewFakeRecorder(2)
			metrics := watchers.NewWatcherMetricsAdapterMock(t)
			metrics.EXPECT().OnChange(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once()
			handler := watchers.NewSharedChaosPodHandler(reader, recorder, zap.NewNop().Sugar(), metrics, nil)

			args := chaosapi.DisruptionArgs{
				DisruptionName: current.Name, DisruptionNamespace: current.Namespace, DisruptionUID: tc.podUID,
			}.CreateCmdArgs([]string{"network-disruption"})
			if tc.inlineUID {
				args = []string{"network-disruption", "--log-context-disruption-uid=" + tc.podUID}
			}
			oldPod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "chaos-pod", Namespace: "chaos-engineering",
					Labels: map[string]string{
						chaostypes.DisruptionNameLabel:      current.Name,
						chaostypes.DisruptionNamespaceLabel: current.Namespace,
					},
				},
				Spec:   corev1.PodSpec{Containers: []corev1.Container{{Name: "injector", Args: args}}},
				Status: corev1.PodStatus{Phase: corev1.PodPending},
			}
			newPod := oldPod.DeepCopy()
			newPod.Status.Phase = corev1.PodFailed
			newPod.Status.Reason = "UnexpectedAdmissionError"
			handler.OnUpdate(oldPod, newPod)

			select {
			case event := <-recorder.Events:
				if !tc.wantEvent {
					t.Fatalf("old chaos pod was attributed to the replacement disruption: %s", event)
				}
				if !strings.Contains(event, string(v1beta1.EventChaosPodFailedState)) {
					t.Fatalf("unexpected event: %s", event)
				}
			default:
				if tc.wantEvent {
					t.Fatal("current disruption's failed chaos pod did not produce an event")
				}
			}
		})
	}
}
