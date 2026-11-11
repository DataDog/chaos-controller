// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package watchers

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Target container reason changes", func() {
	DescribeTable("observes waiting-state failures without needing a Kubernetes warning event", func(oldReason, newReason string, wantWarning bool) {
		handler := DisruptionTargetHandler{log: zap.NewNop().Sugar(), disruption: &v1beta1.Disruption{}}
		oldPod := corev1.Pod{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{
			Name: "app", State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: oldReason}},
		}}}}
		newPod := oldPod.DeepCopy()
		newPod.Status.ContainerStatuses[0].State.Waiting.Reason = newReason
		events := handler.buildPodEventsToSend(oldPod, *newPod, nil)
		Expect(events[v1beta1.EventTargetContainerWarningState]).To(Equal(wantWarning))
	},
		Entry("image pull failure after normal creation", "ContainerCreating", "ImagePullBackOff", true),
		Entry("crash loop after normal creation", "ContainerCreating", "CrashLoopBackOff", true),
		Entry("unchanged status during resync", "CrashLoopBackOff", "CrashLoopBackOff", false),
	)
})
