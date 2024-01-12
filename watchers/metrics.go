// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package watchers

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/o11y/metrics"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
)

// WatcherMetricsAdapter is in charge of watcher metrics.
type WatcherMetricsAdapter interface {
	OnChange(disruption *v1beta1.Disruption, watcherName string, pod *corev1.Pod, node *corev1.Node, okPod bool, okNode bool, event WatcherEventType)
}

type watcherMetricsAdapter struct {
	metricsSink metrics.Sink
	log         *zap.SugaredLogger
}

// NewWatcherMetricsAdapter constructor of WatcherMetricsAdapter
func NewWatcherMetricsAdapter(metricsSink metrics.Sink, log *zap.SugaredLogger) WatcherMetricsAdapter {
	return &watcherMetricsAdapter{
		metricsSink: metricsSink,
		log:         log,
	}
}

// OnChange increment the watcher.calls metrics with
func (m watcherMetricsAdapter) OnChange(disruption *v1beta1.Disruption, watcherName string, pod *corev1.Pod, node *corev1.Node, okPod bool, okNode bool, event WatcherEventType) {
	tags := []string{
		"disruptionName:" + disruption.Name,
		"namespace:" + disruption.Namespace,
		"event:" + string(event),
		"watcher:" + watcherName,
	}

	switch {
	case okPod:
		tags = append(tags, "targetKind:pod",
			"targetName:"+pod.Name,
			"targetNamespace:"+pod.Namespace)
	case okNode:
		tags = append(tags, "targetKind:node",
			"targetName:"+node.Name,
			"targetNamespace:"+node.Namespace)
	default:
		tags = append(tags, "targetKind:object")
	}

	if err := m.metricsSink.MetricWatcherCalls(tags); err != nil {
		m.log.Errorw("error sending a metric", "error", err)
	}
}
