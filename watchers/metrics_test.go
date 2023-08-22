// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.
package watchers_test

import (
	"fmt"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/o11y/metrics"
	"github.com/DataDog/chaos-controller/watchers"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Metrics", func() {
	Describe("OnChange", func() {
		var (
			disruption = &v1beta1.Disruption{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "disruptionName",
					Namespace: "namespace",
				},
			}
			pod = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "podName",
					Namespace: "namespace",
				},
			}
			node = &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nodeName",
					Namespace: "namespace",
				},
			}
			watcherName = "watcherName"
		)
		DescribeTable("success cases", func(pod *corev1.Pod, node *corev1.Node, okPod, okNode bool, event watchers.WatcherEventType) {
			// Arrange
			expectedTags := []string{
				"disruptionName:" + disruption.Name,
				"namespace:" + disruption.Namespace,
				"event:" + string(event),
				"watcher:" + watcherName,
			}

			if okPod {
				expectedTags = append(expectedTags, "targetKind:pod",
					"targetName:"+pod.Name,
					"targetNamespace:"+pod.Namespace)
			} else if okNode {
				expectedTags = append(expectedTags, "targetKind:node",
					"targetName:"+node.Name,
					"targetNamespace:"+node.Namespace)
			} else {
				expectedTags = append(expectedTags, "targetKind:object")
			}

			metricsSinkMock := metrics.NewSinkMock(GinkgoT())

			By("by increment the watcher calls metric")
			metricsSinkMock.EXPECT().MetricWatcherCalls(expectedTags).Return(nil)
			metricsHandler := watchers.NewWatcherMetricsHandler(metricsSinkMock, logger)

			// Action
			metricsHandler.OnChange(disruption, watcherName, pod, node, okPod, okNode, event)
		},
			Entry("with a pod during a delete action",
				pod, nil, true, false, watchers.WatcherDeleteEvent),
			Entry("with a node during a delete action",
				nil, node, false, true, watchers.WatcherDeleteEvent),
			Entry("with a node during an update action",
				nil, node, false, true, watchers.WatcherUpdateEvent),
			Entry("with a pod during an update action",
				pod, nil, true, false, watchers.WatcherUpdateEvent),
			Entry("with a pod during an add action",
				pod, nil, true, false, watchers.WatcherAddEvent),
			Entry("with a pod during an update action",
				pod, nil, true, false, watchers.WatcherUpdateEvent),
			Entry("without a node and without a pod during an update",
				nil, nil, false, false, watchers.WatcherUpdateEvent),
		)

		When("metricsSink return an error", func() {
			It("should be ignored", func() {
				// Arrange
				metricsSinkMock := metrics.NewSinkMock(GinkgoT())
				metricsSinkMock.EXPECT().MetricWatcherCalls(mock.Anything).Return(fmt.Errorf("an error happened"))
				metricsHandler := watchers.NewWatcherMetricsHandler(metricsSinkMock, logger)

				// Action
				metricsHandler.OnChange(disruption, watcherName, nil, nil, false, false, watchers.WatcherAddEvent)
			})
		})
	})
})
