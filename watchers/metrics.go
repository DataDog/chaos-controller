// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package watchers

import (
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	cLog "github.com/DataDog/chaos-controller/log"
	"github.com/DataDog/chaos-controller/o11y/metrics"
	tagutil "github.com/DataDog/chaos-controller/o11y/tags"
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
		tagutil.FormatTag(cLog.DisruptionNameKey, disruption.Name),
		tagutil.FormatTag(cLog.DisruptionNamespaceKey, disruption.Namespace),
		tagutil.FormatTag(cLog.EventKey, string(event)),
		tagutil.FormatTag(cLog.WatcherKey, watcherName),
	}

	switch {
	case okPod:
		tags = append(tags, tagutil.FormatTag(cLog.TargetKindKey, "pod"),
			tagutil.FormatTag(cLog.TargetNameKey, pod.Name),
			tagutil.FormatTag(cLog.TargetNamespaceKey, pod.Namespace))
	case okNode:
		tags = append(tags, tagutil.FormatTag(cLog.TargetKindKey, "node"),
			tagutil.FormatTag(cLog.TargetNameKey, node.Name),
			tagutil.FormatTag(cLog.TargetNamespaceKey, node.Namespace))
	default:
		tags = append(tags, tagutil.FormatTag(cLog.TargetKindKey, "object"))
	}

	if err := m.metricsSink.MetricWatcherCalls(tags); err != nil {
		m.log.Errorw("error sending a metric", cLog.ErrorKey, err)
	}
}
