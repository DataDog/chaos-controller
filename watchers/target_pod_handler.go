// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package watchers

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DisruptionTargetHandler struct used to manage what to do when changes occur on the watched objects in the cache
type DisruptionTargetHandler struct {
	recorder       record.EventRecorder
	reader         client.Reader
	enableObserver bool
	disruption     *v1beta1.Disruption
	log            *zap.SugaredLogger
	metricsAdapter WatcherMetricsAdapter
}

const DisruptionTargetHandlerName = "DisruptionTargetHandler"

// OnAdd new target
func (d DisruptionTargetHandler) OnAdd(obj interface{}) {
	pod, okPod := obj.(*corev1.Pod)
	node, okNode := obj.(*corev1.Node)

	targetName, targetKind := getTargetNameAndKind(obj)

	d.log.Debugw("DisruptionTargetHandler ADD",
		"disruptionName", d.disruption.Name,
		"disruptionNamespace", d.disruption.Namespace,
		"targetName", targetName,
		"targetKind", targetKind,
	)

	d.OnChangeHandleMetricsSink(pod, node, okPod, okNode, WatcherAddEvent)
}

// OnDelete target
func (d DisruptionTargetHandler) OnDelete(obj interface{}) {
	pod, okPod := obj.(*corev1.Pod)
	node, okNode := obj.(*corev1.Node)

	targetName, targetKind := getTargetNameAndKind(obj)

	d.log.Debugw("DisruptionTargetHandler DELETE",
		"disruptionName", d.disruption.Name,
		"disruptionNamespace", d.disruption.Namespace,
		"targetName", targetName,
		"targetKind", targetKind,
	)

	d.OnChangeHandleMetricsSink(pod, node, okPod, okNode, WatcherDeleteEvent)
}

// OnUpdate target
func (d DisruptionTargetHandler) OnUpdate(oldObj, newObj interface{}) {
	oldPod, okOldPod := oldObj.(*corev1.Pod)
	newPod, okNewPod := newObj.(*corev1.Pod)
	oldNode, okOldNode := oldObj.(*corev1.Node)
	newNode, okNewNode := newObj.(*corev1.Node)

	oldTargetName, oldTargetKind := getTargetNameAndKind(oldObj)
	newTargetName, newTargetKind := getTargetNameAndKind(newObj)

	d.log.Debugw("DisruptionTargetHandler UPDATE",
		"disruptionName", d.disruption.Name,
		"disruptionNamespace", d.disruption.Namespace,
		"oldTargetName", oldTargetName,
		"oldTargetKind", oldTargetKind,
		"newTargetName", newTargetName,
		"newTargetKind", newTargetKind,
	)

	d.OnChangeHandleMetricsSink(newPod, newNode, okNewPod, okNewNode, WatcherUpdateEvent)

	if d.enableObserver {
		d.OnChangeHandleNotifierSink(oldPod, newPod, oldNode, newNode, okOldPod, okNewPod, okOldNode, okNewNode)
	}
}

// OnChangeHandleMetricsSink Trigger Metric Sink on changes in the targets
func (d DisruptionTargetHandler) OnChangeHandleMetricsSink(pod *corev1.Pod, node *corev1.Node, okPod, okNode bool, event WatcherEventType) {
	d.metricsAdapter.OnChange(d.disruption, DisruptionTargetHandlerName, pod, node, okPod, okNode, event)
}

// OnChangeHandleNotifierSink Trigger Notifier Sink on changes in the targets
func (d DisruptionTargetHandler) OnChangeHandleNotifierSink(oldPod, newPod *corev1.Pod, oldNode, newNode *corev1.Node, okOldPod, okNewPod, okOldNode, okNewNode bool) {
	var objectToNotify runtime.Object

	eventsToSend := make(map[v1beta1.DisruptionEventReason]bool)
	name := ""

	switch {
	case okNewPod && okOldPod:
		objectToNotify, name = newPod, newPod.Name

		disruptionEvents, err := d.getEventsFromCurrentDisruption("Pod", newPod.ObjectMeta, d.disruption.CreationTimestamp.Time)
		if err != nil {
			d.log.Warnf("couldn't get the list of events from the target. Might not be able to notify on error changes: %s", err.Error())
		}

		// we detect and compute the error / warning events, status changes, conditions of the updated pod
		eventsToSend = d.buildPodEventsToSend(*oldPod, *newPod, disruptionEvents)
	case okNewNode && okOldNode:
		objectToNotify, name = newNode, newNode.Name

		disruptionEvents, err := d.getEventsFromCurrentDisruption("Node", newNode.ObjectMeta, d.disruption.CreationTimestamp.Time)
		if err != nil {
			d.log.Warnf("couldn't get the list of events from the target. Might not be able to notify on error changes: %s", err.Error())
		}

		// we detect and compute the error / warning events, status changes, conditions of the updated node
		eventsToSend = d.buildNodeEventsToSend(*oldNode, *newNode, disruptionEvents)
	default:
		d.log.Warnw("target observer couldn't detect what type of changes happened on the targets")
	}

	// Send events to notifier / to disruption
	lastEvents, _ := d.getEventsFromCurrentDisruption("disruption", d.disruption.ObjectMeta, d.disruption.CreationTimestamp.Time)

	for eventReason, toSend := range eventsToSend {
		if !toSend {
			continue
		}

		eventType := v1beta1.Events[eventReason].Type

		// Send to updated target
		d.recorder.Event(objectToNotify, eventType, string(eventReason), fmt.Sprintf(v1beta1.Events[eventReason].OnTargetTemplateMessage, d.disruption.Name))

		// Send to disruption, broadcast to notifiers
		for _, event := range lastEvents {
			if event.Type == string(eventReason) {
				d.recorder.Event(d.disruption, eventType, string(eventReason), fmt.Sprintf(v1beta1.Events[eventReason].OnDisruptionTemplateMessage, name))

				return
			}
		}

		d.recorder.Event(d.disruption, eventType, string(eventReason), v1beta1.Events[eventReason].OnDisruptionTemplateAggMessage)
	}
}

func (d DisruptionTargetHandler) getEventsFromCurrentDisruption(kind string, objectMeta metav1.ObjectMeta, disruptionStartTime time.Time) ([]corev1.Event, error) {
	eventList := &corev1.EventList{}
	fieldSelector := fields.Set{
		"involvedObject.kind": kind,
		"involvedObject.name": objectMeta.Name,
	}

	err := d.reader.List(context.Background(), eventList, &client.ListOptions{
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
			if event.Type == corev1.EventTypeWarning && event.Reason == string(v1beta1.EventDisrupted) || event.LastTimestamp.Time.Before(disruptionStartTime) {
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
func (d DisruptionTargetHandler) findNotifiableEvents(eventsToSend map[v1beta1.DisruptionEventReason]bool, eventsFromTarget []corev1.Event, recoverTimestamp *time.Time, targetName string) (map[v1beta1.DisruptionEventReason]bool, bool) {
	cannotRecoverYet := true

	for _, event := range eventsFromTarget {
		if event.Source.Component == v1beta1.SourceDisruptionComponent {
			// if the target has not sent any warnings, we can't recover it as there is nothing to recover
			if v1beta1.IsTargetEvent(event) && event.Type == corev1.EventTypeWarning {
				cannotRecoverYet = false
			}

			break
		}
	}

	// ordered by last timestamp
	for _, event := range eventsFromTarget {
		// We stop at the last event sent by us
		if event.Source.Component == v1beta1.SourceDisruptionComponent || (recoverTimestamp != nil && recoverTimestamp.After(event.LastTimestamp.Time)) {
			break
		}

		// if warning event has been sent after target recovering
		switch event.InvolvedObject.Kind {
		case "Pod":
			switch {
			case event.Type == corev1.EventTypeWarning:
				if event.Reason == "Unhealthy" || event.Reason == "ProbeWarning" {
					lowerCasedMessage := strings.ToLower(event.Message)

					switch {
					case strings.Contains(lowerCasedMessage, "liveness probe"):
						eventsToSend[v1beta1.EventTargetLivenessProbeChange] = true
					case strings.Contains(lowerCasedMessage, "readiness probe"):
						// If the object of the disruption is in the list of targets, it means it has been injected.
						// The readiness probe is failing during the injection
						if d.disruption.Status.HasTarget(event.InvolvedObject.Name) {
							eventsToSend[v1beta1.EventTargetReadinessProbeChangeDuringDisruption] = true
						} else {
							eventsToSend[v1beta1.EventTargetReadinessProbeChangeBeforeDisruption] = true
						}
					default:
						eventsToSend[v1beta1.EventTargetPodWarningState] = true
					}
				} else {
					eventsToSend[v1beta1.EventTargetPodWarningState] = true
				}

				d.log.Debugw("warning event detected on target",
					"target", targetName,
					"reason", event.Reason,
					"message", event.Message,
					"timestamp", event.LastTimestamp.Time.Unix(),
				)
			case event.Reason == "Started":
				if recoverTimestamp == nil {
					recoverTimestamp = &event.LastTimestamp.Time
				}

				eventsToSend[v1beta1.EventTargetPodRecoveredState] = true

				d.log.Debugw("recovering event detected on target",
					"target", targetName,
					"reason", event.Reason,
					"message", event.Message,
					"timestamp", event.LastTimestamp.Time.Unix(),
				)
			case event.Reason == "Killing" && strings.Contains(event.Message, "Stopping container") && eventsToSend[v1beta1.EventTargetContainerWarningState]:
				// this event indicates a safe killing of a container (can occur when we rollout or manually delete a pod for example)
				// we remove the warning state event if it has been created when we compared the state of the containers
				delete(eventsToSend, v1beta1.EventTargetContainerWarningState)
			}
		case "Node":
			if event.Type == corev1.EventTypeWarning {
				eventsToSend[v1beta1.EventTargetNodeWarningState] = true

				d.log.Debugw("warning event detected on target",
					"target", targetName,
					"reason", event.Reason,
					"message", event.Message,
					"timestamp", event.LastTimestamp.Time.Unix(),
				)
			} else if event.Reason == "NodeReady" {
				if recoverTimestamp == nil {
					recoverTimestamp = &event.LastTimestamp.Time
				}

				eventsToSend[v1beta1.EventTargetNodeRecoveredState] = true

				d.log.Debugw("recovering event detected on target",
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

func (d DisruptionTargetHandler) buildPodEventsToSend(oldPod corev1.Pod, newPod corev1.Pod, disruptionEvents []corev1.Event) map[v1beta1.DisruptionEventReason]bool {
	var recoverTimestamp *time.Time // keep track of the timestamp of a recovering event / state

	eventsToSend := make(map[v1beta1.DisruptionEventReason]bool)
	runningState := "Running"

	// compare statuses between old and new pod to detect changes
	for _, container := range newPod.Status.ContainerStatuses {
		for _, oldContainer := range oldPod.Status.ContainerStatuses {
			if container.Name != oldContainer.Name { // When restarting, container can change of ID, so we need to verify by name
				continue
			}

			// Warning events
			if container.RestartCount > (oldContainer.RestartCount + 2) {
				d.log.Infow("container restart detected on target",
					"target", fmt.Sprintf("%s/%s", newPod.Namespace, newPod.Name),
					"container", container.Name,
					"restarts", container.RestartCount,
				)

				eventsToSend[v1beta1.EventTargetTooManyRestarts] = true
			}

			lastState, lastReason := getContainerState(oldContainer)
			newState, newReason := getContainerState(container)

			if lastState != newState {
				d.log.Infow("container state change detected on target",
					"target", fmt.Sprintf("%s/%s", newPod.Namespace, newPod.Name),
					"container", container.Name,
					"lastState", lastState,
					"newState", newState,
				)

				switch {
				case newReason == "Completed": // if pod is terminated in a normal way
					continue
				case newState != runningState && newReason != "ContainerCreating": // if pod is in Waiting or Terminated state with warning reasons
					eventsToSend[v1beta1.EventTargetContainerWarningState] = true
				case lastReason != "ContainerCreating" && newState == runningState: // if pod is spawned normally, it was not in a warning state before
					if recoverTimestamp == nil {
						recoverTimestamp = &container.State.Running.StartedAt.Time
					}

					eventsToSend[v1beta1.EventTargetPodRecoveredState] = true
				}
			}

			break
		}
	}

	// remove recovering event if pod has another warning container state
	if eventsToSend[v1beta1.EventTargetPodRecoveredState] && len(eventsToSend) > 1 {
		eventsToSend[v1beta1.EventTargetPodRecoveredState] = false
		recoverTimestamp = nil
	}

	eventsToSend, cannotRecoverYet := d.findNotifiableEvents(eventsToSend, disruptionEvents, recoverTimestamp, fmt.Sprintf("%s/%s", newPod.Namespace, newPod.Name))

	// if other warning events have been detected, the target hasn't recovered
	if cannotRecoverYet || (eventsToSend[v1beta1.EventTargetPodRecoveredState] && len(eventsToSend) > 1) {
		eventsToSend[v1beta1.EventTargetPodRecoveredState] = false
	}

	return eventsToSend
}

func (d DisruptionTargetHandler) buildNodeEventsToSend(oldNode corev1.Node, newNode corev1.Node, targetEvents []corev1.Event) map[v1beta1.DisruptionEventReason]bool {
	var recoverTimestamp *time.Time // keep track of the timestamp of a recovering event / condition / phase

	eventsToSend := make(map[v1beta1.DisruptionEventReason]bool)

	// Evaluate the need to send a warning event on node condition changes
	for _, newCondition := range newNode.Status.Conditions {
		for _, oldCondition := range oldNode.Status.Conditions {
			if newCondition.Type != oldCondition.Type || !newCondition.LastTransitionTime.After(oldCondition.LastTransitionTime.Time) {
				continue
			}

			if newCondition.Status != oldCondition.Status {
				d.log.Debugw("condition changed on target node",
					"target", newNode.Name,
					"conditionType", newCondition.Type,
					"newStatus", newCondition.Status,
					"oldStatus", oldCondition.Status,
					"timestamp", newCondition.LastTransitionTime.Unix())
			}

			if newCondition.Status == corev1.ConditionUnknown && oldCondition.Status != corev1.ConditionUnknown {
				eventsToSend[v1beta1.EventTargetNodeWarningState] = true
			}

			switch newCondition.Type {
			case corev1.NodeReady:
				if newCondition.Status == corev1.ConditionFalse && oldCondition.Status == corev1.ConditionTrue {
					eventsToSend[v1beta1.EventTargetNodeWarningState] = true
				} else if newCondition.Status == corev1.ConditionTrue && oldCondition.Status == corev1.ConditionFalse {
					if recoverTimestamp == nil {
						recoverTimestamp = &newCondition.LastTransitionTime.Time
					}

					eventsToSend[v1beta1.EventTargetNodeRecoveredState] = true
				}
			case corev1.NodeDiskPressure:
				if newCondition.Status == corev1.ConditionTrue && oldCondition.Status == corev1.ConditionFalse {
					eventsToSend[v1beta1.EventTargetNodeDiskPressureState] = true
				}
			case corev1.NodePIDPressure:
				if newCondition.Status == corev1.ConditionTrue && oldCondition.Status == corev1.ConditionFalse {
					eventsToSend[v1beta1.EventTargetNodeWarningState] = true
				}
			case corev1.NodeMemoryPressure:
				if newCondition.Status == corev1.ConditionTrue && oldCondition.Status == corev1.ConditionFalse {
					eventsToSend[v1beta1.EventTargetNodeMemPressureState] = true
				}
			case corev1.NodeNetworkUnavailable:
				if newCondition.Status == corev1.ConditionTrue && oldCondition.Status == corev1.ConditionFalse {
					eventsToSend[v1beta1.EventTargetNodeUnavailableNetworkState] = true
				}
			}

			break
		}
	}

	if newNode.Status.Phase != oldNode.Status.Phase {
		d.log.Debugw("condition changed on target node",
			"target", newNode.Name,
			"newPhase", newNode.Status.Phase,
			"oldPhase", oldNode.Status.Phase,
		)

		switch newNode.Status.Phase {
		case corev1.NodeRunning:
			eventsToSend[v1beta1.EventTargetNodeRecoveredState] = true
		case corev1.NodePending, corev1.NodeTerminated:
			if oldNode.Status.Phase == corev1.NodeRunning {
				eventsToSend[v1beta1.EventTargetNodeWarningState] = true
			}
		}
	}

	// remove recovering event if node has another warning condition / phase
	if eventsToSend[v1beta1.EventTargetNodeRecoveredState] && len(eventsToSend) > 1 {
		eventsToSend[v1beta1.EventTargetNodeRecoveredState] = false
		recoverTimestamp = nil
	}

	eventsToSend, cannotRecoverYet := d.findNotifiableEvents(eventsToSend, targetEvents, recoverTimestamp, newNode.Name)

	// if other warning events have been detected, the target hasn't recovered
	if cannotRecoverYet || (eventsToSend[v1beta1.EventTargetNodeRecoveredState] && len(eventsToSend) > 1) {
		eventsToSend[v1beta1.EventTargetNodeRecoveredState] = false
	}

	return eventsToSend
}

// getTargetNameAnd kind return the name and the kind of object
func getTargetNameAndKind(obj interface{}) (string, string) {
	pod, okPod := obj.(*corev1.Pod)
	node, okNode := obj.(*corev1.Node)

	targetName := "unknown"
	targetKind := "object"

	if okPod {
		targetName = pod.Name
		targetKind = "pod"
	} else if okNode {
		targetName = node.Name
		targetKind = "node"
	}

	return targetName, targetKind
}
