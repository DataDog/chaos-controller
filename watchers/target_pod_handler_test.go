// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package watchers

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zaptest"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/DataDog/chaos-controller/api/v1beta1"
)

var _ = Describe("DisruptionTargetHandler notifier sink", func() {
	var (
		handler    DisruptionTargetHandler
		recorder   *record.FakeRecorder
		disruption *v1beta1.Disruption
	)

	BeforeEach(func() {
		disruption = &v1beta1.Disruption{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "disruption-name",
				Namespace:         "ns",
				CreationTimestamp: metav1.NewTime(time.Now().Add(-time.Minute)),
			},
		}

		recorder = record.NewFakeRecorder(10)

		handler = DisruptionTargetHandler{
			recorder:       recorder,
			enableObserver: true,
			disruption:     disruption,
			log:            zaptest.NewLogger(GinkgoT()).Sugar(),
		}
	})

	When("the disruption already recorded an event with the target warning reason", func() {
		It("sends the single-target notification instead of the aggregated one", func() {
			// a real-shaped event previously sent by this controller: the involved object
			// kind of a Disruption event is "Disruption" and the reason carries the
			// event reason string, not its type
			handler.reader = fakeEventsClient(
				disruptionEvent(v1beta1.EventTargetContainerWarningState),
			)

			oldPod, newPod := healthyPod(), warningPod()

			handler.OnChangeHandleNotifierSink(oldPod, newPod, nil, nil, true, true, false, false)

			// first the warning is recorded on the target itself, then on the disruption
			Expect(recorder.Events).To(Receive(ContainSubstring("containers on pod are failing")))

			// the disruption already carries an event with this reason, so the single-target
			// message is sent instead of the aggregated one
			Expect(recorder.Events).To(Receive(And(
				ContainSubstring(string(v1beta1.EventTargetContainerWarningState)),
				ContainSubstring("target-pod"),
			)))
		})
	})
})

func healthyPod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "target-pod", Namespace: "ns"},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "app",
					State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
				},
			},
		},
	}
}

func warningPod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "target-pod", Namespace: "ns"},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "app",
					State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}},
				},
			},
		},
	}
}

func disruptionEvent(reason v1beta1.EventReason) *corev1.Event {
	return &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "disruption-event",
			Namespace: "ns",
		},
		InvolvedObject: corev1.ObjectReference{
			Kind:      v1beta1.DisruptionKind,
			Name:      "disruption-name",
			Namespace: "ns",
		},
		Type:          v1beta1.Events[reason].Type,
		Reason:        string(reason),
		LastTimestamp: metav1.NewTime(time.Now()),
	}
}

func fakeEventsClient(events ...client.Object) client.Reader {
	return fake.NewClientBuilder().WithScheme(scheme.Scheme).
		WithIndex(&corev1.Event{}, "involvedObject.kind", func(o client.Object) []string {
			return []string{o.(*corev1.Event).InvolvedObject.Kind}
		}).
		WithIndex(&corev1.Event{}, "involvedObject.name", func(o client.Object) []string {
			return []string{o.(*corev1.Event).InvolvedObject.Name}
		}).
		WithObjects(events...).Build()
}
