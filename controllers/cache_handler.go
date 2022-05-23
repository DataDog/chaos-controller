package controllers

import (
	"context"
	"fmt"
	"log"
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
	h.OnChangeHandleNotifierSink(oldPod, newPod, oldNode, newNode, okOldPod, okNewPod, okOldNode, okNewNode)
}

// OnChangeHandleMetricsSink Trigger Metric Sink on changes in the targets
func (h DisruptionTargetWatcherHandler) OnChangeHandleMetricsSink(pod *corev1.Pod, node *corev1.Node, okPod, okNode bool) {
	switch {
	case okPod:
		h.reconciler.handleMetricSinkError(h.reconciler.MetricsSink.MetricSelectorCacheTriggered([]string{"name:" + h.disruption.Name, "namespace:" + h.disruption.Namespace, "event:add", "targetKind:pod", "target:" + pod.Name}))
	case okNode:
		h.reconciler.handleMetricSinkError(h.reconciler.MetricsSink.MetricSelectorCacheTriggered([]string{"name:" + h.disruption.Name, "namespace:" + h.disruption.Namespace, "event:add", "targetKind:node", "target:" + node.Name}))
	default:
		h.reconciler.handleMetricSinkError(h.reconciler.MetricsSink.MetricSelectorCacheTriggered([]string{"name:" + h.disruption.Name, "namespace:" + h.disruption.Namespace, "event:add", "targetKind:object"}))
	}
}

// OnChangeHandleNotifierSink Trigger Notifier Sink on changes in the targets
func (h DisruptionTargetWatcherHandler) OnChangeHandleNotifierSink(oldPod, newPod *corev1.Pod, oldNode, newNode *corev1.Node, okOldPod, okNewPod, okOldNode, okNewNode bool) {
	eventsToSend := make(map[string]bool)
	var objectToNotify runtime.Object
	var name string

	switch {
	case okNewPod && okOldPod:
		objectToNotify = newPod
		name = newPod.Name

		disruptionEvents, err := h.getEventsFromCurrentDisruption("Pod", newPod.ObjectMeta, h.disruption.CreationTimestamp.Time)
		if err != nil {
			h.reconciler.log.Warnf("couldn't get the list of events from the target. Might not be able to notify on error changes: %s", err.Error())
		}

		// we detect and compute the error / warning events, status changes, conditions of the updated pod
		eventsToSend = h.buildPodEventsToSend(*oldPod, *newPod, disruptionEvents)
	case okNewNode && okOldNode:
		objectToNotify = newNode
		name = newNode.Name
		disruptionEvents, err := h.getEventsFromCurrentDisruption("Node", newNode.ObjectMeta, h.disruption.CreationTimestamp.Time)
		if err != nil {
			h.reconciler.log.Warnf("couldn't get the list of events from the target. Might not be able to notify on error changes: %s", err.Error())
		}

		eventsToSend = h.buildNodeEventsToSend(*oldNode, *newNode, getNonAnalyzedEvents(disruptionEvents, "Warning"))
	default:
		h.reconciler.log.Debugw("target observer couldn't detect what type of changes happened on the targets")
	}

	// Send events to notifier / to disruption
	lastEvents, _ := h.getEventsFromCurrentDisruption("Disruption", h.disruption.ObjectMeta, h.disruption.CreationTimestamp.Time)
	for eventReason, toSend := range eventsToSend {
		if !toSend {
			continue
		}

		eventType := corev1.EventTypeWarning
		if eventReason == chaosv1beta1.DIS_NODE_RECOVERED_STATE || eventReason == chaosv1beta1.DIS_POD_RECOVERED_STATE {
			eventType = corev1.EventTypeNormal
		}

		h.reconciler.Recorder.Event(objectToNotify, eventType, eventReason, fmt.Sprintf(chaosv1beta1.ALL_DISRUPTION_EVENTS[eventReason].OnTargetTemplateMessage, h.disruption.Name))
		if hasEventOfReason(lastEvents, eventReason) {
			h.reconciler.Recorder.Event(h.disruption, eventType, eventReason, fmt.Sprintf(chaosv1beta1.ALL_DISRUPTION_EVENTS[eventReason].OnDisruptionTemplateMessage, name))
		} else {
			h.reconciler.Recorder.Event(h.disruption, eventType, eventReason, chaosv1beta1.ALL_DISRUPTION_EVENTS[eventReason].OnDisruptionTemplateAggMessage)
		}
	}
}

func getNonAnalyzedEvents(events []corev1.Event, eventType string) []corev1.Event {
	idx := -1

	for i, event := range events {
		if event.Source.Component == "disruption-controller" &&
			eventType != "" && event.Type == eventType &&
			event.Reason != "Disrupted" {
			idx = i
			break
		}
	}

	if idx == -1 {
		return events
	}
	return events[:idx]
}

func hasEventOfReason(events []corev1.Event, eventReason string) bool {
	for _, event := range events {
		if event.Type == eventReason {
			return true
		}
	}

	return false
}

func (h DisruptionTargetWatcherHandler) getEventsFromCurrentDisruption(kind string, objectMeta v1.ObjectMeta, disruptionStateTime time.Time) ([]corev1.Event, error) {
	fieldSelector := fields.Set{
		"involvedObject.kind": kind,
		"involvedObject.name": objectMeta.Name,
	}

	eventList, err := h.reconciler.DirectClient.CoreV1().Events(objectMeta.Namespace).List(
		context.Background(),
		v1.ListOptions{
			FieldSelector: fieldSelector.AsSelector().String(),
		})
	if err != nil {
		return nil, err
	}

	// Sort by last timestamp
	sort.Slice(eventList.Items, func(i, j int) bool {
		return eventList.Items[i].LastTimestamp.After(eventList.Items[j].LastTimestamp.Time)
	})

	if kind != "Disruption" {
		// Keep events sent during the disruption only
		for i, event := range eventList.Items {
			if event.Type == corev1.EventTypeWarning && event.Reason == "Disrupted" || event.LastTimestamp.Time.Before(disruptionStateTime) {
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
	var state string
	var reason string

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

func (h DisruptionTargetWatcherHandler) buildPodEventsToSend(oldPod corev1.Pod, newPod corev1.Pod, disruptionEvents []corev1.Event) map[string]bool {
	eventToSend := make(map[string]bool)
	cannotRecoverYet := true // know if we can send a recovering event to the disruption

	// https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/events/event.go
	// In case of new events sent from kubelet, we can determine any error event in the pod to propagate it
	for _, event := range disruptionEvents {

		// We stop at the last event sent by us
		if event.Source.Component == "disruption-controller" {
			// if the target has not sent any warnings, we can't recover it as there is nothing to recover
			if event.Type == corev1.EventTypeWarning {
				log.Printf("CAN RECOVER NOW: %s %s %s %s %s", event.LastTimestamp, newPod.Name, event.Type, event.Reason, event.Message)
				cannotRecoverYet = false
			}

			break
		}
		// if there is still warnings in the last events we got, the pod hasn't recovered
		if event.Type == corev1.EventTypeWarning {
			h.reconciler.log.Debugw("warning event detected on target pod", "reason", event.Reason, "message", event.Message, "timestamp", event.LastTimestamp.Time.Unix())

			if event.Reason == "Unhealthy" || event.Reason == "ProbeWarning" {
				lowerCasedMessage := strings.ToLower(event.Message)

				if strings.Contains(lowerCasedMessage, "liveness probe") {
					eventToSend[chaosv1beta1.DIS_LIVENESS_PROBE_CHANGE] = true
				} else if strings.Contains(lowerCasedMessage, "readiness probe") {
					eventToSend[chaosv1beta1.DIS_READINESS_PROBE_CHANGE] = true
				} else {
					eventToSend[chaosv1beta1.DIS_PROBE_CHANGE] = true
				}
			} else {
				eventToSend[chaosv1beta1.DIS_POD_WARNING_STATE] = true
			}
		} else if event.Reason == "Started" {
			log.Printf("< %s - %s %s/%s: %s", event.LastTimestamp, newPod.Name, event.Type, event.Reason, event.Message)
			eventToSend[chaosv1beta1.DIS_POD_RECOVERED_STATE] = true
		}
	}

	// Compare statuses between old and new pod to detect changes
	for _, container := range newPod.Status.ContainerStatuses {
		for _, oldContainer := range oldPod.Status.ContainerStatuses {
			if container.Name != oldContainer.Name { // When restarting, container can change of ID, so we need to verify by name
				continue
			}

			// Warning events
			if container.RestartCount > (oldContainer.RestartCount + 2) {
				h.reconciler.log.Debugw("container restart detected on target pod", "pod", fmt.Sprintf("%s/%s", newPod.Namespace, newPod.Name), "container", container.Name, "restarts", container.RestartCount)

				eventToSend[chaosv1beta1.DIS_TOO_MANY_RESTARTS] = true
			}

			lastState, lastReason := getContainerState(oldContainer)
			newState, newReason := getContainerState(container)

			if lastState != newState {
				h.reconciler.log.Debugw("container state change detected on target pod", "pod", fmt.Sprintf("%s/%s", newPod.Namespace, newPod.Name), "container", container.Name, "lastState", lastState, "newState", newState)

				if newState != "Running" && newReason != "ContainerCreating" && newReason != "KillingContainer" {
					eventToSend[chaosv1beta1.DIS_CONTAINER_WARNING_STATE] = true
				} else {
					log.Printf("< %s - %s :%s/%s %s/%s", newPod.Name, container.Name, lastState, newState, lastReason, newReason)
					eventToSend[chaosv1beta1.DIS_POD_RECOVERED_STATE] = true
				}
			}

			break
		}
	}

	if cannotRecoverYet {
		log.Print("CANT RECOVER")

		eventToSend[chaosv1beta1.DIS_POD_RECOVERED_STATE] = false
	}

	return eventToSend
}

func (h DisruptionTargetWatcherHandler) buildNodeEventsToSend(oldNode corev1.Node, newNode corev1.Node, disruptionEvents []corev1.Event) map[string]bool {
	errorPerDisruptionEvent := make(map[string]bool)

	// https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/events/event.go
	// In case of new events sent from kubelet, we can determine any error event in the node to propagate it
	for _, event := range disruptionEvents {
		if event.Type == "Normal" || event.Source.Component == "disruption-controller" {
			continue
		}

		errorPerDisruptionEvent[chaosv1beta1.DIS_NODE_WARNING_STATE] = true
	}

	// Evaluate the need to send a warning event on node condition changes
	for _, newCondition := range newNode.Status.Conditions {
		for _, oldCondition := range oldNode.Status.Conditions {
			if newCondition.Type != oldCondition.Type {
				continue
			}

			if !newCondition.LastTransitionTime.After(oldCondition.LastTransitionTime.Time) {
				break
			}

			if newCondition.Status == corev1.ConditionUnknown && oldCondition.Status != corev1.ConditionUnknown {
				errorPerDisruptionEvent[chaosv1beta1.DIS_NODE_WARNING_STATE] = true
			}

			switch newCondition.Type {
			case corev1.NodeReady:
				if newCondition.Status == corev1.ConditionFalse && oldCondition.Status == corev1.ConditionTrue {
					errorPerDisruptionEvent[chaosv1beta1.DIS_NODE_WARNING_STATE] = true
				}
			case corev1.NodeDiskPressure:
				if newCondition.Status == corev1.ConditionTrue && oldCondition.Status == corev1.ConditionFalse {
					errorPerDisruptionEvent[chaosv1beta1.DIS_NODE_DISK_PRESSURE_STATE] = true
				}
			case corev1.NodePIDPressure:
				if newCondition.Status == corev1.ConditionTrue && oldCondition.Status == corev1.ConditionFalse {
					errorPerDisruptionEvent[chaosv1beta1.DIS_NODE_WARNING_STATE] = true
				}
			case corev1.NodeMemoryPressure:
				if newCondition.Status == corev1.ConditionTrue && oldCondition.Status == corev1.ConditionFalse {
					errorPerDisruptionEvent[chaosv1beta1.DIS_NODE_MEM_PRESSURE_STATE] = true
				}
			case corev1.NodeNetworkUnavailable:
				if newCondition.Status == corev1.ConditionTrue && oldCondition.Status == corev1.ConditionFalse {
					errorPerDisruptionEvent[chaosv1beta1.DIS_NODE_UNAVAILABLE_NETWORK_STATE] = true
				}
			}
			break
		}
	}

	switch newNode.Status.Phase {
	case corev1.NodePending:
		if oldNode.Status.Phase == corev1.NodeRunning {
			errorPerDisruptionEvent[chaosv1beta1.DIS_NODE_WARNING_STATE] = true
		}
	case corev1.NodeTerminated:
		if oldNode.Status.Phase == corev1.NodeRunning {
			errorPerDisruptionEvent[chaosv1beta1.DIS_NODE_WARNING_STATE] = true
		}
	}

	return errorPerDisruptionEvent
}

// Cache life handler

// createInstanceSelectorCache creates this instance's cache if it doesn't exist and attaches it to the controller
func (r *DisruptionReconciler) manageInstanceSelectorCache(instance *chaosv1beta1.Disruption) error {
	r.clearExpiredCacheContexts()

	// remove check when StaticTargeting is defaulted to false
	if instance.Spec.StaticTargeting == nil {
		r.log.Debugw("StaticTargeting pointer is nil")
	}

	if instance.Spec.StaticTargeting == nil || *instance.Spec.StaticTargeting {
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
		cacheCtx, cacheCancelFunc := context.WithTimeout(context.Background(), instance.Spec.Duration.Duration()+*r.ExpiredDisruptionGCDelay*2)

		go func() { ch <- cache.Start(cacheCtx) }()

		r.CacheContextStore[disCacheHash] = CtxTuple{cacheCtx, cacheCancelFunc}

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

func (r *DisruptionReconciler) clearExpiredCacheContexts() {
	// clean up potential expired cache contexts based on context error return
	deletionList := []string{}

	for key, contextTuple := range r.CacheContextStore {
		if contextTuple.Ctx.Err() != nil {
			deletionList = append(deletionList, key)
		}
	}

	for _, key := range deletionList {
		delete(r.CacheContextStore, key)
	}
}
