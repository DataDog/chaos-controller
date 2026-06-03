// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package watchers

import (
	"context"
	"fmt"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/o11y/tags"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SharedChaosPodHandler is a single shared event handler registered on the manager's pod
// informer (chaos namespace only). It replaces per-disruption ChaosPodWatcher instances,
// eliminating one informer cache per active disruption.
//
// The disruption is identified at event time from the pod's labels rather than being
// pre-stored at construction time.
type SharedChaosPodHandler struct {
	reader         client.Reader
	recorder       record.EventRecorder
	log            *zap.SugaredLogger
	metricsAdapter WatcherMetricsAdapter
}

// NewSharedChaosPodHandler creates a SharedChaosPodHandler.
func NewSharedChaosPodHandler(
	reader client.Reader,
	recorder record.EventRecorder,
	log *zap.SugaredLogger,
	metricsAdapter WatcherMetricsAdapter,
) *SharedChaosPodHandler {
	return &SharedChaosPodHandler{
		reader:         reader,
		recorder:       recorder,
		log:            log,
		metricsAdapter: metricsAdapter,
	}
}

// OnAdd is a no-op — chaos pod creation does not require notification.
func (h *SharedChaosPodHandler) OnAdd(_ interface{}, _ bool) {}

// OnUpdate emits metrics and sends a Kubernetes event to the parent Disruption when
// a chaos pod unexpectedly transitions to a failed phase.
func (h *SharedChaosPodHandler) OnUpdate(oldObj, newObj interface{}) {
	oldPod, okOld := oldObj.(*corev1.Pod)
	newPod, okNew := newObj.(*corev1.Pod)

	if !okOld || !okNew {
		return
	}

	disruptionName := newPod.Labels[chaostypes.DisruptionNameLabel]
	disruptionNamespace := newPod.Labels[chaostypes.DisruptionNamespaceLabel]

	if disruptionName == "" || disruptionNamespace == "" {
		return
	}

	// Build a minimal stub for metrics — only name/namespace are needed there.
	stub := &v1beta1.Disruption{ObjectMeta: metav1.ObjectMeta{Name: disruptionName, Namespace: disruptionNamespace}}
	h.metricsAdapter.OnChange(stub, ChaosPodHandlerName, newPod, nil, true, false, WatcherUpdateEvent)

	if oldPod.Status.Phase == newPod.Status.Phase {
		return
	}

	// Running→Failed is expected (injection completed), do not notify.
	if oldPod.Status.Phase == corev1.PodRunning && newPod.Status.Phase == corev1.PodFailed {
		return
	}

	if newPod.Status.Phase == corev1.PodFailed {
		h.sendEvent(newPod, disruptionName, disruptionNamespace)
	}
}

// OnDelete is a no-op.
func (h *SharedChaosPodHandler) OnDelete(_ interface{}) {}

func (h *SharedChaosPodHandler) sendEvent(pod *corev1.Pod, disruptionName, disruptionNamespace string) {
	disruption := &v1beta1.Disruption{}
	if err := h.reader.Get(context.Background(), types.NamespacedName{Name: disruptionName, Namespace: disruptionNamespace}, disruption); err != nil {
		h.log.Warnw("SharedChaosPodHandler: could not fetch disruption to send event",
			tags.ErrorKey, err,
			tags.DisruptionNameKey, disruptionName,
			tags.DisruptionNamespaceKey, disruptionNamespace,
		)

		return
	}

	eventReason := v1beta1.EventChaosPodFailedState
	eventType := v1beta1.Events[eventReason].Type
	eventMessage := fmt.Sprintf(v1beta1.Events[eventReason].OnDisruptionTemplateMessage,
		pod.Name,
		pod.Status.Phase,
		pod.Status.Reason,
	)

	h.recorder.Event(disruption, eventType, string(eventReason), eventMessage)

	h.log.Debugw("SharedChaosPodHandler UPDATE - Send event",
		tags.DisruptionKey, eventMessage,
		tags.EventTypeKey, eventType,
		tags.DisruptionNameKey, disruptionName,
		tags.DisruptionNamespaceKey, disruptionNamespace,
		tags.ChaosPodNameKey, pod.Name,
	)
}
