package controllers

import (
	"context"
	"fmt"
	"log"
	"reflect"

	"github.com/DataDog/chaos-controller/targetselector"
	chaostypes "github.com/DataDog/chaos-controller/types"
	corev1 "k8s.io/api/core/v1"
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

type DisruptionEvent struct {
	EventType string
	Reason    string
	Status    string
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
		h.notifyOnTargetPodError(*oldPod, *newPod)
	case okNewNode && okOldNode:
	default:
		h.reconciler.log.Debugw("target observer couldn't detect what type of changes happened on the targets")
	}

}

func (h DisruptionSelectorHandler) notifyOnTargetPodResolve(oldPod corev1.Pod, newPod corev1.Pod) {
}

func (h DisruptionSelectorHandler) getPodContainerFailure(oldPod corev1.Pod, newPod corev1.Pod) map[string]DisruptionEvent {
	failuresPerContainers := map[string]DisruptionEvent{}

	for _, container := range newPod.Status.ContainerStatuses {
		for _, oldContainer := range oldPod.Status.ContainerStatuses {
			if container.ContainerID != oldContainer.ContainerID {
				continue
			}

			if reflect.DeepEqual(container.State, oldContainer.State) ||
				container.State.Running != nil ||
				(container.State.Terminated != nil && ContainerValidReason(container.State.Terminated.Reason) == CONTAINER_COMPLETED) ||
				(container.State.Waiting != nil && ContainerValidReason(container.State.Waiting.Reason) == CONTAINER_CREATING) {
				continue
			}

			if container.State.Waiting != nil {
				if !((container.State.Waiting.Reason == "ErrImagePull" && oldContainer.State.Waiting.Reason == "ImagePullBackOff") ||
					(container.State.Waiting.Reason == "ImagePullBackOff" && oldContainer.State.Waiting.Reason == "ErrImagePull")) {
					failuresPerContainers[container.Name] = DisruptionEvent{
						EventType: v1beta1.DIS_CONTAINER_STATE_WAIT_CHANGE,
						Reason:    container.State.Waiting.Reason,
					}
				}
			}

			if container.State.Terminated != nil {
				failuresPerContainers[container.Name] = DisruptionEvent{
					EventType: v1beta1.DIS_CONTAINER_STATE_TERMINATE_CHANGE,
					Reason:    container.State.Terminated.Reason,
				}
			}

			if container.RestartCount > oldContainer.RestartCount && container.RestartCount > 5 {
				failuresPerContainers[container.Name] = DisruptionEvent{
					EventType: v1beta1.DIS_TOO_MANY_RESTARTS,
					Reason:    "Too many restart on target",
				}
			}
		}
	}

	return failuresPerContainers
}

func (h DisruptionSelectorHandler) notifyOnTargetPodError(oldPod corev1.Pod, newPod corev1.Pod) {
	//now := time.Now()

	containerFailures := h.getPodContainerFailure(oldPod, newPod)
	reason := "Pod failing mostly due to the disruption on containers: %s"
	containers := ""
	idx := 0

	for containerName, failure := range containerFailures {
		containers += fmt.Sprintf("%s with %s", containerName, failure.Reason)
		if idx < len(containerFailures) {
			containers += ", "
		}
		idx++
	}

	if len(containerFailures) > 0 {
		h.PostDisruptionStateEvent(&newPod, v1beta1.DIS_CONTAINER_STATE_WAIT_CHANGE, fmt.Sprintf(reason, containers))
	}
}

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
