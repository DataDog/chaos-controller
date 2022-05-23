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
	var objectToNotify runtime.Object
	var name string

	eventsToSend := make(map[string]bool)

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

		eventType := corev1.EventTypeWarning
		if eventReason == chaosv1beta1.DisNodeRecoveredState || eventReason == chaosv1beta1.DisPodRecoveredState {
			eventType = corev1.EventTypeNormal
		}

		// Send to updated target
		h.reconciler.Recorder.Event(objectToNotify, eventType, eventReason, fmt.Sprintf(chaosv1beta1.ALL_DISRUPTION_EVENTS[eventReason].OnTargetTemplateMessage, h.disruption.Name))

		// Send to disruption, broadcast to notifiers
		for _, event := range lastEvents {
			if event.Type == eventReason {
				h.reconciler.Recorder.Event(h.disruption, eventType, eventReason, fmt.Sprintf(chaosv1beta1.ALL_DISRUPTION_EVENTS[eventReason].OnDisruptionTemplateMessage, name))

				return
			}
		}

		h.reconciler.Recorder.Event(h.disruption, eventType, eventReason, chaosv1beta1.ALL_DISRUPTION_EVENTS[eventReason].OnDisruptionTemplateAggMessage)
	}
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

	// Keep events sent during the disruption only, no need to filter events coming from the disruption itself
	if kind != "Disruption" {
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
	eventsToSend, cannotRecoverYet := make(map[string]bool), false

	// https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/events/event.go
	// In case of new events sent from kubelet, we can determine any error event in the pod to propagate it
	for _, event := range disruptionEvents {
		// We stop at the last event sent by us
		if event.Source.Component == chaosv1beta1.SourceDisruptionComponent {
			// if the target has not sent any warnings, we can't recover it as there is nothing to recover
			if event.Type == corev1.EventTypeWarning {
				cannotRecoverYet = false
			}

			break
		}

		if event.Type == corev1.EventTypeWarning {
			h.reconciler.log.Debugw("warning event detected on target pod", "reason", event.Reason, "message", event.Message, "timestamp", event.LastTimestamp.Time.Unix())

			if event.Reason == "Unhealthy" || event.Reason == "ProbeWarning" {
				lowerCasedMessage := strings.ToLower(event.Message)

				switch {
				case strings.Contains(lowerCasedMessage, "liveness probe"):
					eventsToSend[chaosv1beta1.DisLivenessProbeChange] = true
				case strings.Contains(lowerCasedMessage, "liveness probe"):
					eventsToSend[chaosv1beta1.DisReadinessProbeChange] = true
				default:
					eventsToSend[chaosv1beta1.DisPodWarningState] = true
				}
			} else {
				eventsToSend[chaosv1beta1.DisPodWarningState] = true
			}
		} else if event.Reason == "Started" {
			eventsToSend[chaosv1beta1.DisPodRecoveredState] = true
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

				eventsToSend[chaosv1beta1.DisTooManyRestarts] = true
			}

			lastState, _ := getContainerState(oldContainer)
			newState, newReason := getContainerState(container)

			if lastState != newState {
				h.reconciler.log.Debugw("container state change detected on target pod", "pod", fmt.Sprintf("%s/%s", newPod.Namespace, newPod.Name), "container", container.Name, "lastState", lastState, "newState", newState)

				if newState != "Running" && newReason != "ContainerCreating" && newReason != "KillingContainer" {
					eventsToSend[chaosv1beta1.DisContainerWarningState] = true
				} else {
					eventsToSend[chaosv1beta1.DisPodRecoveredState] = true
				}
			}

			break
		}
	}

	if cannotRecoverYet || (eventsToSend[chaosv1beta1.DisPodRecoveredState] && len(eventsToSend) > 1) {
		eventsToSend[chaosv1beta1.DisPodRecoveredState] = false
	}

	return eventsToSend
}

func (h DisruptionTargetWatcherHandler) buildNodeEventsToSend(oldNode corev1.Node, newNode corev1.Node, disruptionEvents []corev1.Event) map[string]bool {
	eventsToSend, cannotRecoverYet := make(map[string]bool), false

	// https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/events/event.go
	// In case of new events sent from kubelet, we can determine any error event in the node to propagate it
	for _, event := range disruptionEvents {
		// We stop at the last event sent by us
		if event.Source.Component == chaosv1beta1.SourceDisruptionComponent {
			// if the target has not sent any warnings, we can't recover it as there is nothing to recover
			if event.Type == corev1.EventTypeWarning {
				cannotRecoverYet = false
			}

			break
		}

		if event.Type == corev1.EventTypeWarning {
			h.reconciler.log.Debugw("warning event detected on target node", "node", fmt.Sprintf("%s/%s", newNode.Namespace, newNode.Name), "reason", event.Reason, "message", event.Message, "timestamp", event.LastTimestamp.Time.Unix())

			eventsToSend[chaosv1beta1.DisNodeWarningState] = true
		} else if event.Reason == "NodeReady" {
			eventsToSend[chaosv1beta1.DisNodeRecoveredState] = true
		}
	}

	// Evaluate the need to send a warning event on node condition changes
	for _, newCondition := range newNode.Status.Conditions {
		for _, oldCondition := range oldNode.Status.Conditions {
			if newCondition.Type != oldCondition.Type || !newCondition.LastTransitionTime.After(oldCondition.LastTransitionTime.Time) {
				continue
			}

			if newCondition.Status != oldCondition.Status {
				h.reconciler.log.Debugw("condition changed on target node",
					"node", fmt.Sprintf("%s/%s", newNode.Namespace, newNode.Name),
					"conditionType", newCondition.Type,
					"newStatus", newCondition.Status,
					"oldStatus", oldCondition.Status,
					"timestamp", newCondition.LastTransitionTime.Unix())
			}

			if newCondition.Status == corev1.ConditionUnknown && oldCondition.Status != corev1.ConditionUnknown {
				eventsToSend[chaosv1beta1.DisNodeWarningState] = true
			}

			switch newCondition.Type {
			case corev1.NodeReady:
				if newCondition.Status == corev1.ConditionFalse && oldCondition.Status == corev1.ConditionTrue {
					eventsToSend[chaosv1beta1.DisNodeWarningState] = true
				} else if newCondition.Status == corev1.ConditionTrue && oldCondition.Status == corev1.ConditionFalse {
					eventsToSend[chaosv1beta1.DisNodeRecoveredState] = true
				}
			case corev1.NodeDiskPressure:
				if newCondition.Status == corev1.ConditionTrue && oldCondition.Status == corev1.ConditionFalse {
					eventsToSend[chaosv1beta1.DisNodeDiskPressureState] = true
				}
			case corev1.NodePIDPressure:
				if newCondition.Status == corev1.ConditionTrue && oldCondition.Status == corev1.ConditionFalse {
					eventsToSend[chaosv1beta1.DisNodeWarningState] = true
				}
			case corev1.NodeMemoryPressure:
				if newCondition.Status == corev1.ConditionTrue && oldCondition.Status == corev1.ConditionFalse {
					eventsToSend[chaosv1beta1.DisNodeMemPressureState] = true
				}
			case corev1.NodeNetworkUnavailable:
				if newCondition.Status == corev1.ConditionTrue && oldCondition.Status == corev1.ConditionFalse {
					eventsToSend[chaosv1beta1.DisNodeUnavailableNetworkState] = true
				}
			}
			break
		}
	}

	switch newNode.Status.Phase {
	case corev1.NodeRunning:
		if oldNode.Status.Phase != corev1.NodeRunning {
			eventsToSend[chaosv1beta1.DisNodeRecoveredState] = true
		}
	case corev1.NodePending, corev1.NodeTerminated:
		if oldNode.Status.Phase == corev1.NodeRunning {
			eventsToSend[chaosv1beta1.DisNodeWarningState] = true
		}
	}

	if newNode.Status.Phase != oldNode.Status.Phase {
		h.reconciler.log.Debugw("condition changed on target node",
			"node", fmt.Sprintf("%s/%s", newNode.Namespace, newNode.Name),
			"newPhase", newNode.Status.Phase,
			"oldPhase", oldNode.Status.Phase,
		)
	}

	if cannotRecoverYet || (eventsToSend[chaosv1beta1.DisNodeRecoveredState] && len(eventsToSend) > 1) {
		eventsToSend[chaosv1beta1.DisNodeRecoveredState] = false
	}

	return eventsToSend
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

// clearExpiredCacheContexts clean up potential expired cache contexts based on context error return
func (r *DisruptionReconciler) clearExpiredCacheContexts() {
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
