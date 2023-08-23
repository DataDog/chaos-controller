// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package watchers

import (
	"fmt"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
)

// ChaosPodHandler handles the lifecycle of chaos pods to send notifications if needed
type ChaosPodHandler struct {
	// recorder is used to emit Kubernetes events.
	recorder record.EventRecorder

	// disruption is the current state of the disruption
	disruption *chaosv1beta1.Disruption

	// log is the logger.
	log *zap.SugaredLogger

	// metricsAdapter is used for metric instrumentation
	metricsAdapter WatcherMetricsAdapter
}

const ChaosPodHandlerName = "ChaosPodHandler"

// NewChaosPodHandler creates a new instance of ChaosPodHandler
func NewChaosPodHandler(recorder record.EventRecorder, disruption *chaosv1beta1.Disruption, logger *zap.SugaredLogger, metricsAdapter WatcherMetricsAdapter) ChaosPodHandler {
	return ChaosPodHandler{
		disruption:     disruption,
		log:            logger,
		recorder:       recorder,
		metricsAdapter: metricsAdapter,
	}
}

// OnAdd is a handler function for the add of the chaos pod
func (w ChaosPodHandler) OnAdd(_ interface{}) {
	// Do nothing on add event
}

// OnUpdate is a handler function for the update of the chaos pod
func (w ChaosPodHandler) OnUpdate(oldObj, newObj interface{}) {
	// Convert oldObj and newObj to Pod objects
	oldPod, okOldPod := oldObj.(*corev1.Pod)
	newPod, okNewPod := newObj.(*corev1.Pod)

	// If both old and new are not pod, do nothing
	if !okOldPod || !okNewPod {
		return
	}

	w.metricsAdapter.OnChange(w.disruption, ChaosPodHandlerName, newPod, nil, okNewPod, false, WatcherUpdateEvent)

	// If the old and new phase are the same, do nothing
	if oldPod.Status.Phase == newPod.Status.Phase {
		return
	}

	// If the old pod is running and the new one is failed, do nothing
	// The disruption is fully injected if the pod is in a running phase.
	if oldPod.Status.Phase == corev1.PodRunning && newPod.Status.Phase == corev1.PodFailed {
		return
	}

	// If the new pod has failed, send an event
	if newPod.Status.Phase == corev1.PodFailed {
		w.sendEvent(newPod)
	}
}

// OnDelete is a handler function for the delete of the chaos pod
func (w ChaosPodHandler) OnDelete(_ interface{}) {
	// Do nothing on delete event
}

// sendEvent sends an event to the recorder with the given newPod object.
func (w ChaosPodHandler) sendEvent(newPod *corev1.Pod) {
	// Prepare event fields
	eventReason := chaosv1beta1.EventChaosPodFailedState
	eventType := chaosv1beta1.Events[eventReason].Type
	eventMessage := fmt.Sprintf(chaosv1beta1.Events[eventReason].OnDisruptionTemplateMessage,
		newPod.Name,
		newPod.Status.Phase,
		newPod.Status.Reason,
	)

	// Send event to notify user
	w.recorder.Event(w.disruption, eventType, string(eventReason), eventMessage)

	w.log.Debugw("ChaosPodHandler UPDATE - Send event",
		"eventMessage", eventMessage,
		"eventType", eventType,
		"disruptionName", w.disruption.Name,
		"disruptionNamespace", w.disruption.Namespace,
		"chaosPodName", newPod.Name,
	)
}
