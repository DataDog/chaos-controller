// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package controllers

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/DataDog/chaos-controller/targetselector"
	chaostypes "github.com/DataDog/chaos-controller/types"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrl "sigs.k8s.io/controller-runtime"
	k8scache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
)

// DisruptionTargetWatcherHandler struct used to manage what to do when changes occur on the watched objects in the cache
type DisruptionTargetWatcherHandler struct {
	reconciler *DisruptionReconciler
	disruption *chaosv1beta1.Disruption
}

// On new target
func (h DisruptionTargetWatcherHandler) OnAdd(obj interface{}) {
	pod, okPod := obj.(*corev1.Pod)
	node, okNode := obj.(*corev1.Node)

	h.OnChangeHandleMetricsSink(pod, node, okPod, okNode)
}

// On target deletion
func (h DisruptionTargetWatcherHandler) OnDelete(obj interface{}) {
	pod, okPod := obj.(*corev1.Pod)
	node, okNode := obj.(*corev1.Node)

	h.OnChangeHandleMetricsSink(pod, node, okPod, okNode)
}

// On updated target
func (h DisruptionTargetWatcherHandler) OnUpdate(oldObj, newObj interface{}) {
	oldPod, okOldPod := oldObj.(*corev1.Pod)
	newPod, okNewPod := newObj.(*corev1.Pod)
	oldNode, okOldNode := oldObj.(*corev1.Node)
	newNode, okNewNode := newObj.(*corev1.Node)

	h.OnChangeHandleMetricsSink(newPod, newNode, okNewPod, okNewNode)

	if h.reconciler.EnableObserver {
		h.OnChangeHandleNotifierSink(oldPod, newPod, oldNode, newNode, okOldPod, okNewPod, okOldNode, okNewNode)
	}
}

// OnChangeHandleMetricsSink Trigger Metric Sink on changes in the targets
func (h DisruptionTargetWatcherHandler) OnChangeHandleMetricsSink(pod *corev1.Pod, node *corev1.Node, okPod, okNode bool) {
	switch {
	case okPod:
		h.reconciler.handleMetricSinkError(h.reconciler.MetricsSink.MetricSelectorCacheTriggered([]string{"disruptionName:" + h.disruption.Name, "namespace:" + h.disruption.Namespace, "event:add", "targetKind:pod", "target:" + pod.Name}))
	case okNode:
		h.reconciler.handleMetricSinkError(h.reconciler.MetricsSink.MetricSelectorCacheTriggered([]string{"disruptionName:" + h.disruption.Name, "namespace:" + h.disruption.Namespace, "event:add", "targetKind:node", "target:" + node.Name}))
	default:
		h.reconciler.handleMetricSinkError(h.reconciler.MetricsSink.MetricSelectorCacheTriggered([]string{"disruptionName:" + h.disruption.Name, "namespace:" + h.disruption.Namespace, "event:add", "targetKind:object"}))
	}
}

// OnChangeHandleNotifierSink Trigger Notifier Sink on changes in the targets
func (h DisruptionTargetWatcherHandler) OnChangeHandleNotifierSink(oldPod, newPod *corev1.Pod, oldNode, newNode *corev1.Node, okOldPod, okNewPod, okOldNode, okNewNode bool) {
	var objectToNotify runtime.Object

	eventsToSend := make(map[string]bool)
	name := ""

	switch {
	case okNewPod && okOldPod:
		objectToNotify, name = newPod, newPod.Name

		disruptionEvents, err := h.getEventsFromCurrentDisruption("Pod", newPod.ObjectMeta, h.disruption.CreationTimestamp.Time)
		if err != nil {
			h.reconciler.log.Warnf("couldn't get the list of events from the target. Might not be able to notify on error changes: %s", err.Error())
		}

		// we detect and compute the error / warning events, status changes, conditions of the updated pod
		eventsToSend = h.buildPodEventsToSend(*oldPod, *newPod, disruptionEvents)
	case okNewNode && okOldNode:
		objectToNotify, name = newNode, newNode.Name

		disruptionEvents, err := h.getEventsFromCurrentDisruption("Node", newNode.ObjectMeta, h.disruption.CreationTimestamp.Time)
		if err != nil {
			h.reconciler.log.Warnf("couldn't get the list of events from the target. Might not be able to notify on error changes: %s", err.Error())
		}

		// we detect and compute the error / warning events, status changes, conditions of the updated node
		eventsToSend = h.buildNodeEventsToSend(*oldNode, *newNode, disruptionEvents)
	default:
		h.reconciler.log.Debugw("target observer couldn't detect what type of changes happened on the targets")
	}

	// Send events to notifier / to disruption
	lastEvents, _ := h.getEventsFromCurrentDisruption("Disruption", h.disruption.ObjectMeta, h.disruption.CreationTimestamp.Time)

	for eventReason, toSend := range eventsToSend {
		if !toSend {
			continue
		}

		eventType := chaosv1beta1.Events[eventReason].Type

		if eventReason == chaosv1beta1.EventNodeRecoveredState || eventReason == chaosv1beta1.EventPodRecoveredState {
			eventType = corev1.EventTypeNormal
		}

		// Send to updated target
		h.reconciler.Recorder.Event(objectToNotify, eventType, eventReason, fmt.Sprintf(chaosv1beta1.Events[eventReason].OnTargetTemplateMessage, h.disruption.Name))

		// Send to disruption, broadcast to notifiers
		for _, event := range lastEvents {
			if event.Type == eventReason {
				h.reconciler.Recorder.Event(h.disruption, eventType, eventReason, fmt.Sprintf(chaosv1beta1.Events[eventReason].OnDisruptionTemplateMessage, name))

				return
			}
		}

		h.reconciler.Recorder.Event(h.disruption, eventType, eventReason, chaosv1beta1.Events[eventReason].OnDisruptionTemplateAggMessage)
	}
}

func (h DisruptionTargetWatcherHandler) getEventsFromCurrentDisruption(kind string, objectMeta v1.ObjectMeta, disruptionStateTime time.Time) ([]corev1.Event, error) {
	eventList := &corev1.EventList{}
	fieldSelector := fields.Set{
		"involvedObject.kind": kind,
		"involvedObject.name": objectMeta.Name,
	}

	err := h.reconciler.Reader.List(context.Background(), eventList, &client.ListOptions{
		FieldSelector: fieldSelector.AsSelector(),
		Namespace:     objectMeta.GetNamespace(),
	})
	if err != nil {
		return nil, err
	}

	// Sort by last timestamp
	sort.Slice(eventList.Items, func(i, j int) bool {
		return eventList.Items[i].LastTimestamp.After(eventList.Items[j].LastTimestamp.Time)
	})

	// Keep events sent during the disruption only, no need to filter events coming from the disruption itself
	if kind != "Disruption" {
		for i, event := range eventList.Items {
			if event.Type == corev1.EventTypeWarning && event.Reason == chaosv1beta1.EventDisrupted || event.LastTimestamp.Time.Before(disruptionStateTime) {
				if i == 0 {
					return []corev1.Event{}, nil
				}

				return eventList.Items[:(i - 1)], nil
			}
		}
	}

	return eventList.Items, nil
}

func getContainerState(containerStatus corev1.ContainerStatus) (string, string) {
	var state, reason string

	switch {
	case containerStatus.State.Running != nil:
		state = "Running"
	case containerStatus.State.Waiting != nil:
		state = "Waiting"
		reason = containerStatus.State.Waiting.Reason
	case containerStatus.State.Terminated != nil:
		state = "Terminated"
		reason = containerStatus.State.Terminated.Reason
	}

	return state, reason
}

// https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/events/event.go
// In case of new events sent from kubelet, we can determine any error event in the node to propagate it
func (h DisruptionTargetWatcherHandler) findNotifiableEvents(eventsToSend map[string]bool, eventsFromTarget []corev1.Event, recoverTimestamp *time.Time, targetName string) (map[string]bool, bool) {
	cannotRecoverYet := true

	for _, event := range eventsFromTarget {
		if event.Source.Component == chaosv1beta1.SourceDisruptionComponent {
			// if the target has not sent any warnings, we can't recover it as there is nothing to recover
			if chaosv1beta1.IsTargetEvent(event) && event.Type == corev1.EventTypeWarning {
				cannotRecoverYet = false
			}

			break
		}
	}

	// ordered by last timestamp
	for _, event := range eventsFromTarget {
		// We stop at the last event sent by us
		if event.Source.Component == chaosv1beta1.SourceDisruptionComponent || (recoverTimestamp != nil && recoverTimestamp.After(event.LastTimestamp.Time)) {
			break
		}

		// if warning event has been sent after target recovering
		switch event.InvolvedObject.Kind {
		case "Pod":
			if event.Type == corev1.EventTypeWarning {
				if event.Reason == "Unhealthy" || event.Reason == "ProbeWarning" {
					lowerCasedMessage := strings.ToLower(event.Message)

					switch {
					case strings.Contains(lowerCasedMessage, "liveness probe"):
						eventsToSend[chaosv1beta1.EventLivenessProbeChange] = true
					case strings.Contains(lowerCasedMessage, "readiness probe"):
						if h.disruption.Status.HasTarget(event.InvolvedObject.Name) {
							eventsToSend[chaosv1beta1.EventReadinessProbeChangeDuringDisruption] = true
						} else {
							eventsToSend[chaosv1beta1.EventReadinessProbeChangeBeforeDisruption] = true
						}
					default:
						eventsToSend[chaosv1beta1.EventPodWarningState] = true
					}
				} else {
					eventsToSend[chaosv1beta1.EventPodWarningState] = true
				}

				h.reconciler.log.Debugw("warning event detected on target",
					"target", targetName,
					"reason", event.Reason,
					"message", event.Message,
					"timestamp", event.LastTimestamp.Time.Unix(),
				)
			} else if event.Reason == "Started" {
				if recoverTimestamp == nil {
					recoverTimestamp = &event.LastTimestamp.Time
				}

				eventsToSend[chaosv1beta1.EventPodRecoveredState] = true

				h.reconciler.log.Infow("recovering event detected on target",
					"target", targetName,
					"reason", event.Reason,
					"message", event.Message,
					"timestamp", event.LastTimestamp.Time.Unix(),
				)
			}
		case "Node":
			if event.Type == corev1.EventTypeWarning {
				eventsToSend[chaosv1beta1.EventNodeWarningState] = true

				h.reconciler.log.Debugw("warning event detected on target",
					"target", targetName,
					"reason", event.Reason,
					"message", event.Message,
					"timestamp", event.LastTimestamp.Time.Unix(),
				)
			} else if event.Reason == "NodeReady" {
				if recoverTimestamp == nil {
					recoverTimestamp = &event.LastTimestamp.Time
				}

				eventsToSend[chaosv1beta1.EventNodeRecoveredState] = true

				h.reconciler.log.Infow("recovering event detected on target",
					"target", targetName,
					"reason", event.Reason,
					"message", event.Message,
					"timestamp", event.LastTimestamp.Time.Unix(),
				)
			}
		}
	}

	return eventsToSend, cannotRecoverYet
}

func (h DisruptionTargetWatcherHandler) buildPodEventsToSend(oldPod corev1.Pod, newPod corev1.Pod, disruptionEvents []corev1.Event) map[string]bool {
	var recoverTimestamp *time.Time // keep track of the timestamp of a recovering event / state

	eventsToSend := make(map[string]bool)
	runningState := "Running"

	// compare statuses between old and new pod to detect changes
	for _, container := range newPod.Status.ContainerStatuses {
		for _, oldContainer := range oldPod.Status.ContainerStatuses {
			if container.Name != oldContainer.Name { // When restarting, container can change of ID, so we need to verify by name
				continue
			}

			// Warning events
			if container.RestartCount > (oldContainer.RestartCount + 2) {
				h.reconciler.log.Infow("container restart detected on target",
					"target", fmt.Sprintf("%s/%s", newPod.Namespace, newPod.Name),
					"container", container.Name,
					"restarts", container.RestartCount,
				)

				eventsToSend[chaosv1beta1.EventTooManyRestarts] = true
			}

			lastState, lastReason := getContainerState(oldContainer)
			newState, newReason := getContainerState(container)

			if lastState != newState {
				h.reconciler.log.Infow("container state change detected on target",
					"target", fmt.Sprintf("%s/%s", newPod.Namespace, newPod.Name),
					"container", container.Name,
					"lastState", lastState,
					"newState", newState,
				)

				switch {
				case newReason == "Completed": // if pod is terminated in a normal way
					continue
				case newState != runningState && newReason != "ContainerCreating": // if pod is in Waiting or Terminated state with warning reasons
					eventsToSend[chaosv1beta1.EventContainerWarningState] = true
				case lastReason != "ContainerCreating" && newState == runningState: // if pod is spawned normally, it was not in a warning state before
					if recoverTimestamp == nil {
						recoverTimestamp = &container.State.Running.StartedAt.Time
					}

					eventsToSend[chaosv1beta1.EventPodRecoveredState] = true
				}
			}

			break
		}
	}

	// remove recovering event if pod has another warning container state
	if eventsToSend[chaosv1beta1.EventPodRecoveredState] && len(eventsToSend) > 1 {
		eventsToSend[chaosv1beta1.EventPodRecoveredState] = false
		recoverTimestamp = nil
	}

	eventsToSend, cannotRecoverYet := h.findNotifiableEvents(eventsToSend, disruptionEvents, recoverTimestamp, fmt.Sprintf("%s/%s", newPod.Namespace, newPod.Name))

	// if other warning events has been detected, the target hasn't recovered
	if cannotRecoverYet || (eventsToSend[chaosv1beta1.EventPodRecoveredState] && len(eventsToSend) > 1) {
		eventsToSend[chaosv1beta1.EventPodRecoveredState] = false
	}

	return eventsToSend
}

func (h DisruptionTargetWatcherHandler) buildNodeEventsToSend(oldNode corev1.Node, newNode corev1.Node, targetEvents []corev1.Event) map[string]bool {
	var recoverTimestamp *time.Time // keep track of the timestamp of a recovering event / condition / phase

	eventsToSend := make(map[string]bool)

	// Evaluate the need to send a warning event on node condition changes
	for _, newCondition := range newNode.Status.Conditions {
		for _, oldCondition := range oldNode.Status.Conditions {
			if newCondition.Type != oldCondition.Type || !newCondition.LastTransitionTime.After(oldCondition.LastTransitionTime.Time) {
				continue
			}

			if newCondition.Status != oldCondition.Status {
				h.reconciler.log.Debugw("condition changed on target node",
					"target", newNode.Name,
					"conditionType", newCondition.Type,
					"newStatus", newCondition.Status,
					"oldStatus", oldCondition.Status,
					"timestamp", newCondition.LastTransitionTime.Unix())
			}

			if newCondition.Status == corev1.ConditionUnknown && oldCondition.Status != corev1.ConditionUnknown {
				eventsToSend[chaosv1beta1.EventNodeWarningState] = true
			}

			switch newCondition.Type {
			case corev1.NodeReady:
				if newCondition.Status == corev1.ConditionFalse && oldCondition.Status == corev1.ConditionTrue {
					eventsToSend[chaosv1beta1.EventNodeWarningState] = true
				} else if newCondition.Status == corev1.ConditionTrue && oldCondition.Status == corev1.ConditionFalse {
					if recoverTimestamp == nil {
						recoverTimestamp = &newCondition.LastTransitionTime.Time
					}

					eventsToSend[chaosv1beta1.EventNodeRecoveredState] = true
				}
			case corev1.NodeDiskPressure:
				if newCondition.Status == corev1.ConditionTrue && oldCondition.Status == corev1.ConditionFalse {
					eventsToSend[chaosv1beta1.EventNodeDiskPressureState] = true
				}
			case corev1.NodePIDPressure:
				if newCondition.Status == corev1.ConditionTrue && oldCondition.Status == corev1.ConditionFalse {
					eventsToSend[chaosv1beta1.EventNodeWarningState] = true
				}
			case corev1.NodeMemoryPressure:
				if newCondition.Status == corev1.ConditionTrue && oldCondition.Status == corev1.ConditionFalse {
					eventsToSend[chaosv1beta1.EventNodeMemPressureState] = true
				}
			case corev1.NodeNetworkUnavailable:
				if newCondition.Status == corev1.ConditionTrue && oldCondition.Status == corev1.ConditionFalse {
					eventsToSend[chaosv1beta1.EventNodeUnavailableNetworkState] = true
				}
			}

			break
		}
	}

	if newNode.Status.Phase != oldNode.Status.Phase {
		h.reconciler.log.Debugw("condition changed on target node",
			"target", newNode.Name,
			"newPhase", newNode.Status.Phase,
			"oldPhase", oldNode.Status.Phase,
		)

		switch newNode.Status.Phase {
		case corev1.NodeRunning:
			eventsToSend[chaosv1beta1.EventNodeRecoveredState] = true
		case corev1.NodePending, corev1.NodeTerminated:
			if oldNode.Status.Phase == corev1.NodeRunning {
				eventsToSend[chaosv1beta1.EventNodeWarningState] = true
			}
		}
	}

	// remove recovering event if node has another warning condition / phase
	if eventsToSend[chaosv1beta1.EventNodeRecoveredState] && len(eventsToSend) > 1 {
		eventsToSend[chaosv1beta1.EventNodeRecoveredState] = false
		recoverTimestamp = nil
	}

	eventsToSend, cannotRecoverYet := h.findNotifiableEvents(eventsToSend, targetEvents, recoverTimestamp, newNode.Name)

	// if other warning events has been detected, the target hasn't recovered
	if cannotRecoverYet || (eventsToSend[chaosv1beta1.EventNodeRecoveredState] && len(eventsToSend) > 1) {
		eventsToSend[chaosv1beta1.EventNodeRecoveredState] = false
	}

	return eventsToSend
}

// Cache life handler

// createInstanceSelectorCache creates this instance's cache if it doesn't exist and attaches it to the controller
func (r *DisruptionReconciler) manageInstanceSelectorCache(instance *chaosv1beta1.Disruption) error {
	r.clearExpiredCacheContexts()

	if instance.Spec.StaticTargeting {
		return nil
	}

	disNamespacedName := types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}
	disSpecHash, err := instance.Spec.HashNoCount()

	if err != nil {
		return fmt.Errorf("error getting disruption hash")
	}

	disCacheHash := disNamespacedName.String() + disSpecHash
	disCompleteSelector, err := targetselector.GetLabelSelectorFromInstance(instance)

	if err != nil {
		return fmt.Errorf("error getting instance selector: %w", err)
	}
	// if it doesn't exist, create the cache context to re-trigger the disruption
	if _, ok := r.CacheContextStore[disCacheHash]; !ok {
		// create the cache/watcher with its options
		var cacheOptions k8scache.Options
		if instance.Spec.Level == chaostypes.DisruptionLevelNode {
			cacheOptions = k8scache.Options{
				SelectorsByObject: k8scache.SelectorsByObject{
					&corev1.Node{}: {Label: disCompleteSelector},
				},
			}
		} else {
			cacheOptions = k8scache.Options{
				SelectorsByObject: k8scache.SelectorsByObject{
					&corev1.Pod{}: {Label: disCompleteSelector},
				},
				Namespace: instance.Namespace,
			}
		}

		cache, err := k8scache.New(
			ctrl.GetConfigOrDie(),
			cacheOptions,
		)

		if err != nil {
			return fmt.Errorf("cache gen error: %w", err)
		}

		// attach handler to cache in order to monitor the cache activity
		var info k8scache.Informer

		if instance.Spec.Level == chaostypes.DisruptionLevelNode {
			info, err = cache.GetInformer(context.Background(), &corev1.Node{})
		} else {
			info, err = cache.GetInformer(context.Background(), &corev1.Pod{})
		}

		if err != nil {
			return fmt.Errorf("cache gen error: %w", err)
		}

		info.AddEventHandler(DisruptionTargetWatcherHandler{disruption: instance, reconciler: r})

		// start the cache with a cancelable context and duration, and attach it to the controller as a watch source
		ch := make(chan error)

		cacheCtx, cacheCancelFunc := context.WithCancel(context.Background())
		ctxTuple := CtxTuple{cacheCtx, cacheCancelFunc, disNamespacedName}

		r.CacheContextStore[disCacheHash] = ctxTuple

		go func() { ch <- cache.Start(cacheCtx) }()
		go r.cacheDeletionSafety(ctxTuple, disCacheHash)

		var cacheSource source.SyncingSource
		if instance.Spec.Level == chaostypes.DisruptionLevelNode {
			cacheSource = source.NewKindWithCache(&corev1.Node{}, cache)
		} else {
			cacheSource = source.NewKindWithCache(&corev1.Pod{}, cache)
		}

		return r.Controller.Watch(
			cacheSource,
			handler.EnqueueRequestsFromMapFunc(
				func(c client.Object) []reconcile.Request {
					return []reconcile.Request{{NamespacedName: disNamespacedName}}
				}),
		)
	}

	return nil
}

// clearInstanceCache closes the context for the disruption-related cache and cleans the cancelFunc array (if it exists)
func (r *DisruptionReconciler) clearInstanceSelectorCache(instance *chaosv1beta1.Disruption) {
	disNamespacedName := types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}
	disSpecHash, err := instance.Spec.HashNoCount()

	if err != nil {
		r.log.Errorf("error getting disruption hash")
		return
	}

	disCacheHash := disNamespacedName.String() + disSpecHash
	contextTuple, ok := r.CacheContextStore[disCacheHash]

	if ok {
		contextTuple.CancelFunc()
		delete(r.CacheContextStore, disCacheHash)
	}
}

// clearExpiredCacheContexts clean up potential expired cache contexts based on context error return
func (r *DisruptionReconciler) clearExpiredCacheContexts() {
	deletionList := []string{}

	for key, contextTuple := range r.CacheContextStore {
		if contextTuple.Ctx.Err() != nil {
			deletionList = append(deletionList, key)
			continue
		}

		if err := r.Get(contextTuple.Ctx, contextTuple.DisruptionNamespacedName, &chaosv1beta1.Disruption{}); err != nil {
			if client.IgnoreNotFound(err) == nil {
				contextTuple.CancelFunc()

				deletionList = append(deletionList, key)
			}
		}
	}

	for _, key := range deletionList {
		delete(r.CacheContextStore, key)
	}
}

// cacheDeletionSafety is thought to be run in a goroutine to assert a cache is not running without its disruption
// the polling is living on the cache context, meaning if it's deleted elsewhere this function will return early.
func (r *DisruptionReconciler) cacheDeletionSafety(ctxTpl CtxTuple, disHash string) {
	_ = wait.PollInfiniteWithContext(ctxTpl.Ctx, time.Minute, func(context.Context) (bool, error) {
		if err := r.Get(ctxTpl.Ctx, ctxTpl.DisruptionNamespacedName, &chaosv1beta1.Disruption{}); err != nil {
			if client.IgnoreNotFound(err) == nil {
				defer ctxTpl.CancelFunc()
				delete(r.CacheContextStore, disHash)
				return true, nil
			}
		}
		return false, nil
	})
}
