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
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	k8scache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
)

// DisruptionSelectorHandler struct used to manage what to do when changes occur on the watched objects in the cache
type DisruptionSelectorHandler struct {
	reconciler *DisruptionReconciler
	disruption *chaosv1beta1.Disruption
}

func (h DisruptionSelectorHandler) OnAdd(obj interface{}) {
	pod, okPod := obj.(*corev1.Pod)
	node, okNode := obj.(*corev1.Node)

	h.OnChangeHandleMetricsSink(pod, node, okPod, okNode)
}

func (h DisruptionSelectorHandler) OnDelete(obj interface{}) {
	pod, okPod := obj.(*corev1.Pod)
	node, okNode := obj.(*corev1.Node)

	h.OnChangeHandleMetricsSink(pod, node, okPod, okNode)
}

func (h DisruptionSelectorHandler) OnUpdate(oldObj, newObj interface{}) {
	oldPod, okOldPod := oldObj.(*corev1.Pod)
	newPod, okNewPod := newObj.(*corev1.Pod)
	oldNode, okOldNode := oldObj.(*corev1.Node)
	newNode, okNewNode := newObj.(*corev1.Node)

	h.OnChangeHandleMetricsSink(newPod, newNode, okNewPod, okNewNode)
	h.OnChangeHandleNotifierSink(oldPod, newPod, oldNode, newNode, okOldPod, okNewPod, okOldNode, okNewNode)
}

// OnChangeHandleMetricsSink Trigger Metric Sink on changes in the targets
func (h DisruptionSelectorHandler) OnChangeHandleMetricsSink(pod *corev1.Pod, node *corev1.Node, okPod, okNode bool) {
	switch {
	case okPod:
		h.reconciler.handleMetricSinkError(h.reconciler.MetricsSink.MetricSelectorCacheTriggered([]string{"name:" + h.disruption.Name, "namespace:" + h.disruption.Namespace, "event:add", "targetKind:pod", "target:" + pod.Name}))
	case okNode:
		h.reconciler.handleMetricSinkError(h.reconciler.MetricsSink.MetricSelectorCacheTriggered([]string{"name:" + h.disruption.Name, "namespace:" + h.disruption.Namespace, "event:add", "targetKind:node", "target:" + node.Name}))
	default:
		h.reconciler.handleMetricSinkError(h.reconciler.MetricsSink.MetricSelectorCacheTriggered([]string{"name:" + h.disruption.Name, "namespace:" + h.disruption.Namespace, "event:add", "targetKind:object"}))
	}
}

func (h DisruptionSelectorHandler) OnChangeHandleNotifierSink(oldPod, newPod *corev1.Pod, oldNode, newNode *corev1.Node, okOldPod, okNewPod, okOldNode, okNewNode bool) {
	switch {
	case okNewPod && okOldPod:
		disruptionEvents, err := h.getPodEventsFromCurrentDisruption(*newPod)
		if err != nil {
			h.reconciler.log.Warnf("couldn't get the list of events from the target. Might not be able to notify on error changes: %s", err.Error())
		}

		if !h.notifyOnPodFailures(*oldPod, *newPod, getNonAnalyzedEvents(disruptionEvents, "Warning")) {
			h.notifyOnPodSuccesses(*oldPod, *newPod, disruptionEvents)
		}
	case okNewNode && okOldNode:
	default:
		h.reconciler.log.Debugw("target observer couldn't detect what type of changes happened on the targets")
	}

}

func getNonAnalyzedEvents(events []corev1.Event, eventType string) []corev1.Event {
	idx := 0
	lastWarningNotificationIsNoticed := false

	for i, event := range events {
		if event.Source.Component == "disruption-controller" && eventType != "" && event.Type == eventType {
			idx = i
			lastWarningNotificationIsNoticed = true
			break
		}
	}

	if !lastWarningNotificationIsNoticed {
		return events
	}

	return events[:idx]
}

func getEventsFromEventType(events []corev1.Event, eventType string) []corev1.Event {
	eventsFromEventType := []corev1.Event{}

	for _, event := range events {
		if event.Reason == eventType {
			eventsFromEventType = append(eventsFromEventType, event)
		}
	}

	return eventsFromEventType
}

func (h DisruptionSelectorHandler) getPodEventsFromCurrentDisruption(pod corev1.Pod) ([]corev1.Event, error) {
	fieldSelector := fields.Set{
		"involvedObject.kind": "Pod",
		"involvedObject.name": pod.Name,
	}

	eventList, err := h.reconciler.DirectClient.CoreV1().Events(pod.Namespace).List(
		context.Background(),
		v1.ListOptions{
			FieldSelector: fieldSelector.AsSelector().String(),
		})
	if err != nil {
		return nil, err
	}

	sort.Slice(eventList.Items, func(i, j int) bool {
		return eventList.Items[i].LastTimestamp.After(eventList.Items[j].LastTimestamp.Time)
	})

	// Get only events sent during the disruption
	idxStartDisruption := 0
	for i, event := range eventList.Items {
		if event.Reason == "Disrupted" && event.Source.Component == "disruption-controller" {
			idxStartDisruption = i
			break
		}
	}

	return eventList.Items[:idxStartDisruption], nil
}

func (h DisruptionSelectorHandler) notifyOnPodFailures(oldPod corev1.Pod, newPod corev1.Pod, disruptionEvents []corev1.Event) bool {
	now := time.Now()
	errorPerDisruptionEvent := make(map[string]bool)

	// https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/events/event.go
	// In case of new events sent from kubelet, we can determine any error event in the pod to propagate it
	for _, event := range disruptionEvents {
		if event.Type == "Normal" {
			continue
		}

		switch event.Reason {
		// Container and Image warning events which might be related to the disruption
		case "Failed", "ErrImageNeverPull", "BackOff", "InspectFailed":
			errorPerDisruptionEvent[chaosv1beta1.DIS_CONTAINER_WARNING_STATE] = true
		// Pod warning events
		case "FailedKillPod", "FailedCreatePodContainer", "NetworkNotReady":
			errorPerDisruptionEvent[chaosv1beta1.DIS_POD_WARNING_STATE] = true
		// Node related events
		case "KubeletSetupFailed", "FailedAttachVolume", "FailedMountVolume", "VolumeResizeFailed", "FileSystemResizeFailed", "FailedMapVolume", "AlreadyMountedVolume", "FailedMountOnFilesystemMismatch":
			errorPerDisruptionEvent[chaosv1beta1.DIS_POD_WARNING_STATE] = true
		// Probe events
		case "Unhealthy", "ProbeWarning":
			lowerCasedMessage := strings.ToLower(event.Message)
			switch {
			case strings.Contains(lowerCasedMessage, "liveness probe"):
				errorPerDisruptionEvent[chaosv1beta1.DIS_LIVENESS_PROBE_CHANGE] = true
			case strings.Contains(lowerCasedMessage, "readiness probe"):
				errorPerDisruptionEvent[chaosv1beta1.DIS_READINESS_PROBE_CHANGE] = true
			default:
				errorPerDisruptionEvent[chaosv1beta1.DIS_PROBE_CHANGE] = true
			}

		// other events
		case "FailedSync", "FailedValidation", "FailedPostStartHook", "FailedPreStopHook":
			errorPerDisruptionEvent[chaosv1beta1.DIS_POD_WARNING_STATE] = true
		}
	}

	// Compare statuses between old and new pod to detect changes
	for _, container := range newPod.Status.ContainerStatuses {
		for _, oldContainer := range oldPod.Status.ContainerStatuses {
			if container.Name != oldContainer.Name { // When restarting, container can change of ID, so we need to verify by name
				continue
			}

			if container.State.Waiting != nil && container.State.Waiting.Reason != "ContainerCreating" {
				if oldContainer.State.Waiting == nil || (oldContainer.State.Waiting != nil && oldContainer.State.Waiting != container.State.Waiting) {
					errorPerDisruptionEvent[chaosv1beta1.DIS_CONTAINER_WARNING_STATE] = true
				}
			}

			if container.State.Terminated != nil && container.State.Terminated.Reason != "Completed" {
				if oldContainer.State.Terminated == nil || (oldContainer.State.Terminated != nil && container.State.Terminated.Reason != oldContainer.State.Terminated.Reason) {
					errorPerDisruptionEvent[chaosv1beta1.DIS_POD_WARNING_STATE] = true
				}
			}

			if container.RestartCount > (oldContainer.RestartCount + 2) {
				errorPerDisruptionEvent[chaosv1beta1.DIS_TOO_MANY_RESTARTS] = true
			}
		}
	}

	// Send event to disruption to propagate with the notifier if the exact same event has been sent more than 5 min ago
	for eventType := range errorPerDisruptionEvent {
		// Send event to target to keep track of sent events
		h.reconciler.Recorder.Event(&newPod, corev1.EventTypeWarning, eventType, fmt.Sprintf(chaosv1beta1.ALL_DISRUPTION_EVENTS[eventType].OnTargetTemplateMessage, h.disruption.Name))

		lastEvents := getEventsFromEventType(disruptionEvents, eventType)
		if len(lastEvents) == 0 {
			h.reconciler.Recorder.Event(h.disruption, corev1.EventTypeWarning, eventType, fmt.Sprintf(chaosv1beta1.ALL_DISRUPTION_EVENTS[eventType].OnDisruptionTemplateMessage, newPod.Name))
		} else if lastEvents[0].LastTimestamp.Time.Before(now.Add(time.Minute * 5)) {
			if len(lastEvents) > 1 {
				h.reconciler.Recorder.Event(h.disruption, corev1.EventTypeWarning, eventType, fmt.Sprintf(chaosv1beta1.ALL_DISRUPTION_EVENTS[eventType].OnDisruptionTemplateAggMessage, newPod.Name, len(lastEvents)))
			} else {
				h.reconciler.Recorder.Event(h.disruption, corev1.EventTypeWarning, eventType, fmt.Sprintf(chaosv1beta1.ALL_DISRUPTION_EVENTS[eventType].OnDisruptionTemplateMessage, newPod.Name))
			}
		}
	}

	return len(errorPerDisruptionEvent) > 0
}

func (h DisruptionSelectorHandler) notifyOnPodSuccesses(oldPod corev1.Pod, newPod corev1.Pod, disruptionEvents []corev1.Event) {
	recover := false
	canRecover := false

	for i, event := range disruptionEvents {
		if v1beta1.IsDisruptionEvent(event, "Warning") {
			canRecover = true
			disruptionEvents = disruptionEvents[:i]

			break
		}

		// If there is no warning event sent before the last recovered event
		if event.Type == chaosv1beta1.DIS_RECOVERED_STATE {
			return
		}
	}

	// If there is no warning event
	if !canRecover {
		return
	}

	for _, event := range disruptionEvents {
		if event.Type != "Normal" {
			return
		}

		if event.Type == "Started" {
			recover = true
		}
	}

	for _, container := range newPod.Status.ContainerStatuses {
		for _, oldContainer := range oldPod.Status.ContainerStatuses {
			if container.Name != oldContainer.Name { // When restarting, container can change of ID, so we need to verify by name
				continue
			}

			if container.State.Running != nil && oldContainer.State.Waiting != nil && oldContainer.State.Waiting.Reason != "ContainerCreating" {
				recover = true
			}
		}
	}

	if recover {
		eventReason := chaosv1beta1.DIS_RECOVERED_STATE
		h.reconciler.Recorder.Event(h.disruption, corev1.EventTypeNormal, eventReason, fmt.Sprintf(chaosv1beta1.ALL_DISRUPTION_EVENTS[eventReason].OnDisruptionTemplateMessage, newPod.Name))
		h.reconciler.Recorder.Event(h.disruption, "Recovering", eventReason, fmt.Sprintf(chaosv1beta1.ALL_DISRUPTION_EVENTS[eventReason].OnDisruptionTemplateMessage, newPod.Name))
	}
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

		info.AddEventHandler(DisruptionSelectorHandler{disruption: instance, reconciler: r})

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
					log.Printf("\n\nRECONCILE TRIGGERED\n\n")
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
