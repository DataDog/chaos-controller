// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package watchers_test

import (
	"fmt"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/watchers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
)

var _ = Describe("Chaos Pod watcher", func() {
	Describe("Handler", func() {
		const (
			bufferSize = 10
		)

		var (
			chaosPodHandler    watchers.ChaosPodHandler
			disruption         *v1beta1.Disruption
			eventRecorder      record.EventRecorder
			metricsAdapterMock *watchers.WatcherMetricsAdapterMock
		)

		JustBeforeEach(func() {
			// Arrange
			disruption = &v1beta1.Disruption{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "disruption-name",
					Namespace: "namespace",
				},
			}
			eventRecorder = record.NewFakeRecorder(bufferSize)
			metricsAdapterMock = watchers.NewWatcherMetricsAdapterMock(GinkgoT())
			metricsAdapterMock.EXPECT().OnChange(
				mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			).Maybe()
			chaosPodHandler = watchers.NewChaosPodHandler(eventRecorder, disruption, logger, metricsAdapterMock)
		})

		Describe("on Add", func() {
			var newPod v1.Pod

			JustBeforeEach(func() {
				// Action
				chaosPodHandler.OnAdd(&newPod)
			})

			It("should do nothing", func() {})
		})

		Describe("on Update", func() {
			Context("with pods", func() {
				var oldPod, newPod v1.Pod

				JustBeforeEach(func() {
					// Action
					chaosPodHandler.OnUpdate(&oldPod, &newPod)
				})

				When("the old pod is in a running phase and the new pod is in a failed phase", func() {
					BeforeEach(func() {
						// Arrange
						oldPodStatus := v1.PodStatus{Phase: v1.PodRunning}
						newPodStatus := v1.PodStatus{Phase: v1.PodFailed}
						oldPod, newPod = createPods(oldPodStatus, newPodStatus)
					})

					It("should not send a warning event", func() {
						Consistently(eventRecorder.(*record.FakeRecorder).Events).ShouldNot(Receive())
					})
				})

				When("the old pod is in a pending phase and the new pod is in a failing phase", func() {
					BeforeEach(func() {
						// Arrange
						oldPodStatus := v1.PodStatus{
							Phase: v1.PodPending,
						}
						newPodStatus := v1.PodStatus{
							Phase:  v1.PodFailed,
							Reason: "OutOfpods",
						}
						oldPod, newPod = createPods(oldPodStatus, newPodStatus)
					})

					It("should send a warning event", func() {
						expectEventType := v1beta1.Events[v1beta1.EventChaosPodFailedState].Type
						expectEventMessage := fmt.Sprintf(v1beta1.Events[v1beta1.EventChaosPodFailedState].OnDisruptionTemplateMessage,
							newPod,
							v1.PodFailed,
							"OutOfpods",
						)

						Eventually(eventRecorder.(*record.FakeRecorder).Events).Should(Receive(ContainSubstring(expectEventType), ContainSubstring(expectEventMessage)))
					})

					It("should send the event once", func() {
						Expect(eventRecorder.(*record.FakeRecorder).Events).Should(HaveLen(1))
					})
				})

				When("the old pod and the new pod are both in a failing phase", func() {
					BeforeEach(func() {
						// Arrange
						podStatus := v1.PodStatus{Phase: v1.PodFailed}
						oldPod, newPod = createPods(podStatus, podStatus)
					})

					It("should not send a warning event", func() {
						fakeEventRecorder := eventRecorder.(*record.FakeRecorder)
						Consistently(fakeEventRecorder.Events).ShouldNot(Receive())
					})
				})

				When("the old pod and the new pod are both in a pending phase", func() {
					BeforeEach(func() {
						// Arrange
						podStatus := v1.PodStatus{Phase: v1.PodPending}
						oldPod, newPod = createPods(podStatus, podStatus)
					})

					It("should not send a warning event", func() {
						Consistently(eventRecorder.(*record.FakeRecorder).Events).ShouldNot(Receive())
					})
				})

				When("the old pod is in a running phase  and the new pod is in a Succeed phase", func() {
					BeforeEach(func() {
						// Arrange
						oldPodStatus := v1.PodStatus{Phase: v1.PodRunning}
						newPodStatus := v1.PodStatus{Phase: v1.PodSucceeded}
						oldPod, newPod = createPods(oldPodStatus, newPodStatus)
					})

					It("should not send a warning event", func() {
						Consistently(eventRecorder.(*record.FakeRecorder).Events).ShouldNot(Receive())
					})
				})
			})

			Context("with nodes", func() {
				var oldNode, newNode v1.Node

				BeforeEach(func() {
					chaosPodHandler.OnUpdate(&oldNode, &newNode)
				})

				It("should not send a warning event", func() {
					Consistently(eventRecorder.(*record.FakeRecorder).Events).ShouldNot(Receive())
				})
			})
		})

		Describe("on Delete", func() {
			var deletePod v1.Pod

			JustBeforeEach(func() {
				// Action
				chaosPodHandler.OnDelete(&deletePod)
			})

			It("should do nothing", func() {})
		})
	})
})

func createPods(oldPodStatus, newPodStatus v1.PodStatus) (oldPod, newPod v1.Pod) {
	oldPod = v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "oldPod"},
		Status:     oldPodStatus,
	}
	newPod = v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "newPod"},
		Status:     newPodStatus,
	}
	return
}
