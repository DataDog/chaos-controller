// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package watchers

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Target event object identity", func() {
	It("does not attribute a replaced pod's events to the new pod with the same name", func() {
		start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
		scheme := runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		oldEvent := &corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: "old-warning", Namespace: "ns"},
			InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "worker-0", Namespace: "ns", UID: "old-pod"},
			Type:           corev1.EventTypeWarning, Reason: "Unhealthy",
			Source:        corev1.EventSource{Component: "kubelet"},
			LastTimestamp: metav1.NewTime(start.Add(time.Minute)),
		}
		newEvent := oldEvent.DeepCopy()
		newEvent.Name = "new-warning"
		newEvent.InvolvedObject.UID = "new-pod"
		newEvent.LastTimestamp = metav1.NewTime(start.Add(2 * time.Minute))
		cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(oldEvent, newEvent).
			WithIndex(&corev1.Event{}, "involvedObject.kind", func(obj client.Object) []string {
				return []string{obj.(*corev1.Event).InvolvedObject.Kind}
			}).
			WithIndex(&corev1.Event{}, "involvedObject.name", func(obj client.Object) []string {
				return []string{obj.(*corev1.Event).InvolvedObject.Name}
			}).
			WithIndex(&corev1.Event{}, "involvedObject.uid", func(obj client.Object) []string {
				return []string{string(obj.(*corev1.Event).InvolvedObject.UID)}
			}).Build()
		handler := DisruptionTargetHandler{reader: cl, log: zap.NewNop().Sugar()}
		events, err := handler.getEventsFromCurrentDisruption("Pod", metav1.ObjectMeta{
			Name: "worker-0", Namespace: "ns", UID: "new-pod",
		}, start)
		Expect(err).NotTo(HaveOccurred())
		Expect(events).To(HaveLen(1))
		Expect(events[0].InvolvedObject.UID).To(Equal(newEvent.InvolvedObject.UID))
	})
})
