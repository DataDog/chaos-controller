// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package watchers

import (
	"time"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Target pod recovery requires all containers", func() {
	DescribeTable("does not recover a pod while another container remains in a warning state", func(stillFailing, resync bool) {
		start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
		handler := DisruptionTargetHandler{log: zap.NewNop().Sugar(), disruption: &v1beta1.Disruption{}}
		oldPod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ns"},
			Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{
				{Name: "app", State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}}},
				{Name: "sidecar", State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}}},
			}},
		}
		newPod := oldPod.DeepCopy()
		newPod.Status.ContainerStatuses[0].State = corev1.ContainerState{Running: &corev1.ContainerStateRunning{StartedAt: metav1.NewTime(start.Add(2 * time.Minute))}}
		if !stillFailing {
			newPod.Status.ContainerStatuses[1].State = corev1.ContainerState{Running: &corev1.ContainerStateRunning{StartedAt: metav1.NewTime(start.Add(2 * time.Minute))}}
		}
		history := []corev1.Event{{
			Source:        corev1.EventSource{Component: v1beta1.SourceDisruptionComponent},
			Type:          corev1.EventTypeWarning,
			Reason:        string(v1beta1.EventTargetContainerWarningState),
			LastTimestamp: metav1.NewTime(start.Add(time.Minute)),
		}}
		if resync {
			oldPod = *newPod.DeepCopy()
			history = append([]corev1.Event{{
				InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: newPod.Name},
				Source:         corev1.EventSource{Component: "kubelet"},
				Type:           corev1.EventTypeNormal,
				Reason:         "Started",
				LastTimestamp:  metav1.NewTime(start.Add(2 * time.Minute)),
			}}, history...)
		}
		events := handler.buildPodEventsToSend(oldPod, *newPod, history)
		Expect(events[v1beta1.EventTargetPodRecoveredState]).To(Equal(!stillFailing))
	},
		Entry("another container is still failing", true, false),
		Entry("a Started event during resync does not imply full recovery", true, true),
		Entry("all containers recovered", false, false),
		Entry("a Started event for a healthy pod reports recovery", false, true),
	)
})
