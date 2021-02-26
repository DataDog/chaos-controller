// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package controllers

import (
	"context"
	"fmt"
	"math"
	"strings"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"github.com/cenkalti/backoff"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// eventsWatcher represents a set of workers reading events
// incoming into the watcher events channel and processing them
type eventsWatcher struct {
	watcher            watch.Interface
	activeWorkersCount int
}

// createChaosPodsWatcher creates a watcher for the given instance chaos pods
// it'll create a set of workers catching any "modified" event happening on the pod
// the number of workers created depends on the number of targets the disruption has
// the number of workers can only increase until the disruption is removed
func (r *DisruptionReconciler) createChaosPodsWatcher(instance *chaosv1beta1.Disruption) error {
	// check for an already existing watcher
	instanceEventsWatcher, found := watchers[instance.UID]

	if !found {
		r.log.Info("creating a watcher on instance chaos pods")

		// create a lister-watcher for the pod resource scoped to the instance namespace
		listWatch := cache.NewListWatchFromClient(r.RawClient, string(corev1.ResourcePods), instance.Namespace, fields.Everything())

		// create a watcher for the pods owned by the given disruption instance
		set := map[string]string{chaostypes.DisruptionNameLabel: instance.Name}
		ls := labels.SelectorFromSet(set)

		watcher, err := listWatch.Watch(metav1.ListOptions{LabelSelector: ls.String()})
		if err != nil {
			return fmt.Errorf("error watching instance chaos pods: %w", err)
		}

		// add the watcher to the local cache
		instanceEventsWatcher = &eventsWatcher{watcher: watcher}
		watchers[instance.UID] = instanceEventsWatcher
		r.handleMetricSinkError(r.MetricsSink.MetricWatchersCount(float64(len(watchers))))

		// handle existing chaos pods termination before listening to events
		// it is useful if a pod was deleted while we were not listening to events (otherwise they would never be cleaned)
		r.log.Info("reconciling existing chaos pods before starting to watch new events")

		if err := r.handleChaosPodsTermination(instance); err != nil {
			return fmt.Errorf("error handling existing chaos pods termination before starting the watcher: %w", err)
		}
	}

	// compute needed workers count depending on the number of targets
	// workersCount will be at least 1
	// log(0) == -Inf; log(1) == 0; so max(-Inf, 1) or max(0, 1) will return 1
	workersCount := int(math.Max(math.Ceil(math.Log(float64(len(instance.Status.Targets)))), 1))
	neededWorkers := workersCount - instanceEventsWatcher.activeWorkersCount

	// create needed additional watcher workers
	// we never decrease the number of workers
	if neededWorkers > 0 {
		r.log.Infof("creating %d new watcher workers for disruption instance", neededWorkers)

		for i := 0; i < neededWorkers; i++ {
			instanceEventsWatcher.activeWorkersCount++
			go r.watchChaosPodsEvents(instance, instanceEventsWatcher.watcher, i+neededWorkers)
		}
	}

	return nil
}

// watchChaosPodsEvents reads events incoming in the given watcher events channel and processes them
func (r *DisruptionReconciler) watchChaosPodsEvents(instance *chaosv1beta1.Disruption, watcher watch.Interface, workerID int) {
	r.log.Infow("creating a new watcher worker", "instance", instance.Name, "namespace", instance.Namespace, "workerID", workerID)
	defer r.log.Infow("exiting chaos pods watcher function", "instance", instance.Name, "namespace", instance.Namespace, "workerID", workerID)

	// build instance id
	id := types.NamespacedName{
		Namespace: instance.Namespace,
		Name:      instance.Name,
	}

	// read events incoming in the watcher events chan
	for event := range watcher.ResultChan() {
		pod := event.Object.(*corev1.Pod)      // chaos pod
		instance := &chaosv1beta1.Disruption{} // disruption instance

		// we look for modified chaos pods because the deleted event happens only once the finalizer is removed and the resource is garbage collected
		// we will receive multiple modified events on resource deletion
		if event.Type == watch.Modified {
			// handle the pod termination to know if we can remove the finalizer or not
			if err := backoff.Retry(func() error {
				// retrieve instance from given identifier
				if err := r.Get(context.Background(), id, instance); err != nil {
					// the disruption has been deleted in the meantime, so we can ignore the event and safely exit the retry
					if client.IgnoreNotFound(err) == nil {
						return nil
					}

					r.log.Errorw("error getting disruption instance", "instance", id.Name, "namespace", id.Namespace, "error", err, "chaosPod", pod.Name, "workerID", workerID)

					return err
				}

				// retrieve pod from retrieved object
				// we must get it (instead of re-using the event object) because we need the latest
				// version of the resource to be able to update it (and remove its finalizer)
				if err := r.Get(context.Background(), types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}, pod); err != nil {
					// the pod has been deleted in the meantime, so we can ignore the event and safely exit the retry
					if client.IgnoreNotFound(err) == nil {
						return nil
					}

					r.log.Errorw("error getting chaos pod from watcher", "instance", id.Name, "namespace", id.Namespace, "error", err, "chaosPod", pod.Name, "workerID", workerID)

					return err
				}

				// handle event
				if err := r.handleChaosPodTermination(instance, pod); err != nil {
					r.log.Errorw("error handling chaos pod termination from watcher", "instance", id.Name, "namespace", id.Namespace, "error", err, "chaosPod", pod.Name, "workerID", workerID)

					return err
				}

				return nil
			}, backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 5)); err != nil {
				r.log.Errorw("error calling chaos pods termination handling backoff function", "instance", id.Name, "namespace", id.Namespace, "error", err, "workerID", workerID)
			}
		}
	}
}

// unwatchChaosPods stops any watcher present on the disruption instance
// and removes it from the local cache
func (r *DisruptionReconciler) unwatchChaosPods(instance *chaosv1beta1.Disruption) {
	if eventsWatcher, found := watchers[instance.UID]; found {
		r.log.Info("removing chaos pods watcher")

		eventsWatcher.watcher.Stop()
		delete(watchers, instance.UID)
		r.handleMetricSinkError(r.MetricsSink.MetricWatchersCount(float64(len(watchers))))
	}
}

// handleChaosPodsTermination runs the pod termination handling for all chaos pods of the given instance
// it returns true if all pods are cleaned successfully, false if at least one of them is not
func (r *DisruptionReconciler) handleChaosPodsTermination(instance *chaosv1beta1.Disruption) error {
	// get chaos pods
	chaosPods, err := r.getChaosPods(instance, nil)
	if err != nil {
		return fmt.Errorf("error getting disruption chaos pods: %w", err)
	}

	// handle each pod termination status
	for _, chaosPod := range chaosPods {
		err := r.handleChaosPodTermination(instance, &chaosPod)
		if err != nil {
			return fmt.Errorf("error handling chaos pod termination: %w", err)
		}
	}

	return nil
}

// handleChaosPodTermination determines if the finalizer of the chaos pod can be removed and removes it when it can
// it returns true when the finalizer has been removed and false otherwise
// when this function returns false with no error, it means that the pod finalizer can't be removed at all
func (r *DisruptionReconciler) handleChaosPodTermination(instance *chaosv1beta1.Disruption, pod *corev1.Pod) error {
	removeFinalizer := false
	ignoreCleanupStatus := false

	// exit early for pods not being deleted so we don't remove the finalizer by mistake
	// and for pods not having a finalizer anymore
	if pod.DeletionTimestamp == nil || (pod.DeletionTimestamp != nil && pod.DeletionTimestamp.IsZero()) || !controllerutil.ContainsFinalizer(pod, chaosPodFinalizer) {
		return nil
	}

	// determine chaos pod target health to know if we can ignore the cleanup status or not
	target := pod.Labels[chaostypes.TargetLabel]

	if err := r.TargetSelector.TargetIsHealthy(target, r.Client, instance); err != nil {
		if errors.IsNotFound(err) || strings.ToLower(err.Error()) == "pod is not running" || strings.ToLower(err.Error()) == "node is not ready" {
			// if the target is not in a good shape, we still run the cleanup phase but we don't check for any issues happening during
			// the cleanup to avoid blocking the disruption deletion for nothing
			r.log.Infow("target is not likely to be cleaned (either it does not exist anymore or it is not ready), the injector will TRY to clean it but will not take care about any failures", "target", target)

			// by enabling this, we will remove the target associated chaos pods finalizers and delete them to trigger the cleanup phase
			// but the chaos pods status will not be checked
			ignoreCleanupStatus = true
		} else {
			return fmt.Errorf("error checking chaos pod target health: %w", err)
		}
	}

	// determine if the finalizer can be removed or not
	switch pod.Status.Phase {
	case corev1.PodSucceeded, corev1.PodPending:
		// pod has terminated successfully or is pending
		// we can remove the pod and the finalizer and it'll be garbage collected
		removeFinalizer = true
	case corev1.PodFailed:
		// pod has failed
		// we need to determine if we can remove it safely or if we need to block disruption deletion
		// check if a container has been created (if not, the disruption was not injected)
		if len(pod.Status.ContainerStatuses) == 0 {
			removeFinalizer = true
		}

		// check if the container was able to start or not
		// if not, we can safely delete the pod since the disruption was not injected
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.Name == "injector" {
				if cs.State.Terminated != nil && cs.State.Terminated.Reason == "StartError" {
					removeFinalizer = true
				}

				break
			}
		}
	default:
		// pod is not in a termination state, we ignore it
		return nil
	}

	// remove the finalizer once possible
	if removeFinalizer || ignoreCleanupStatus {
		r.log.Infow("chaos pod has terminated, removing finalizer", "target", target, "chaosPod", pod.Name)

		// move the target to ignored targets so it is not selected again later
		if err := r.ignoreTarget(instance, target); err != nil {
			return fmt.Errorf("error moving target to ignored targets: %w", err)
		}

		// remove the finalizer from the chaos pod
		controllerutil.RemoveFinalizer(pod, chaosPodFinalizer)

		if err := r.Client.Update(context.Background(), pod); err != nil {
			return fmt.Errorf("error removing chaos pod finalizer: %w", err)
		}
	} else {
		// instance is stuck on removal because the pod finalizer can't be removed safely
		r.log.Infow("instance is or will be stuck on removal, its status will be updated", "chaosPod", pod.Name)

		r.Recorder.Event(instance, "Warning", "StuckOnRemoval", fmt.Sprintf("Instance is or will be stuck on removal because of a chaos pod (%s) not being able to terminate correctly, please check pods logs before manually removing their finalizer", pod.Name))

		instance.Status.IsStuckOnRemoval = true

		if err := r.Update(context.Background(), instance); err != nil {
			return fmt.Errorf("error updating instance status: %w", err)
		}
	}

	return nil
}

func (r *DisruptionReconciler) ignoreTarget(instance *chaosv1beta1.Disruption, target string) error {
	r.log.Infow("adding the target to ignored targets", "target", target)

	// remove target from targets list
	for i, t := range instance.Status.Targets {
		if t == target {
			instance.Status.Targets[len(instance.Status.Targets)-1], instance.Status.Targets[i] = instance.Status.Targets[i], instance.Status.Targets[len(instance.Status.Targets)-1]
			instance.Status.Targets = instance.Status.Targets[:len(instance.Status.Targets)-1]

			break
		}
	}

	// add target to ignored targets list
	if !contains(instance.Status.IgnoredTargets, target) {
		instance.Status.IgnoredTargets = append(instance.Status.IgnoredTargets, target)
	}

	return r.Update(context.Background(), instance)
}
