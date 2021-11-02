// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package controllers

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"strings"
	"time"

	chaosapi "github.com/DataDog/chaos-controller/api"
	"github.com/DataDog/chaos-controller/metrics"
	"github.com/DataDog/chaos-controller/targetselector"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"github.com/cenkalti/backoff"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/env"
)

const (
	finalizerPrefix     = "finalizer.chaos.datadoghq.com"
	disruptionFinalizer = finalizerPrefix
	chaosPodFinalizer   = finalizerPrefix + "/chaos-pod"
)

// DisruptionReconciler reconciles a Disruption object
type DisruptionReconciler struct {
	client.Client
	BaseLog                               *zap.SugaredLogger
	Scheme                                *runtime.Scheme
	Recorder                              record.EventRecorder
	MetricsSink                           metrics.Sink
	TargetSelector                        targetselector.TargetSelector
	InjectorAnnotations                   map[string]string
	InjectorServiceAccount                string
	InjectorImage                         string
	ImagePullSecrets                      string
	log                                   *zap.SugaredLogger
	InjectorServiceAccountNamespace       string
	InjectorDNSDisruptionDNSServer        string
	InjectorDNSDisruptionKubeDNS          string
	InjectorNetworkDisruptionAllowedHosts []string
	ExpiredDisruptionGCDelay              time.Duration
}

//+kubebuilder:rbac:groups=chaos.datadoghq.com,resources=disruptions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=chaos.datadoghq.com,resources=disruptions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=chaos.datadoghq.com,resources=disruptions/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=nodes,verbs=list;watch
//+kubebuilder:rbac:groups=core,resources=services,verbs=list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *DisruptionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	instance := &chaosv1beta1.Disruption{}
	tsStart := time.Now()

	rand.Seed(time.Now().UnixNano())

	// prepare logger instance context
	// NOTE: it is valid while we don't do concurrent reconciling
	// because the logger instance is pointer, concurrent reconciling would create a race condition
	// where the logger context would change for all ongoing reconcile loops
	// in the case we enable concurrent reconciling, we should create one logger instance per reconciling call
	r.log = r.BaseLog.With("instance", req.Name, "namespace", req.Namespace)

	// reconcile metrics
	r.handleMetricSinkError(r.MetricsSink.MetricReconcile())

	defer func() func() {
		return func() {
			tags := []string{}
			if instance.Name != "" {
				tags = append(tags, "name:"+instance.Name, "namespace:"+instance.Namespace)
			}

			r.handleMetricSinkError(r.MetricsSink.MetricReconcileDuration(time.Since(tsStart), tags))
		}
	}()()

	// fetch the instance
	r.log.Infow("fetching disruption instance")

	if err := r.Get(context.Background(), req.NamespacedName, instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// handle any chaos pods being deleted (either by the disruption deletion or by an external event)
	if err := r.handleChaosPodsTermination(instance); err != nil {
		r.log.Errorw("error handling chaos pods termination", "error", err)

		return ctrl.Result{}, err
	}

	// check whether the object is being deleted or not
	if !instance.ObjectMeta.DeletionTimestamp.IsZero() {
		// the instance is being deleted, clean it if the finalizer is still present
		if controllerutil.ContainsFinalizer(instance, disruptionFinalizer) {
			isCleaned, err := r.cleanDisruption(instance)
			if err != nil {
				return ctrl.Result{}, err
			}

			// if not cleaned yet, requeue and reconcile again in 5s-10s
			// the reason why we don't rely on the exponential backoff here is that it retries too fast at the beginning
			if !isCleaned {
				requeueAfter := time.Duration(rand.Intn(5)+5) * time.Second //nolint:gosec

				r.log.Infow(fmt.Sprintf("disruption has not been fully cleaned yet, re-queuing in %v", requeueAfter))

				return ctrl.Result{
					Requeue:      true,
					RequeueAfter: requeueAfter,
				}, r.Update(context.Background(), instance)
			}

			// we reach this code when all the cleanup pods have succeeded
			// we can remove the finalizer and let the resource being garbage collected
			r.log.Infow("removing finalizer")
			controllerutil.RemoveFinalizer(instance, disruptionFinalizer)

			if err := r.Update(context.Background(), instance); err != nil {
				return ctrl.Result{}, err
			}

			// send reconciling duration metric
			r.handleMetricSinkError(r.MetricsSink.MetricCleanupDuration(time.Since(instance.ObjectMeta.DeletionTimestamp.Time), []string{"name:" + instance.Name, "namespace:" + instance.Namespace}))
			r.handleMetricSinkError(r.MetricsSink.MetricDisruptionCompletedDuration(time.Since(instance.ObjectMeta.CreationTimestamp.Time), []string{"name:" + instance.Name, "namespace:" + instance.Namespace}))
			r.emitKindCountMetrics(instance)

			return ctrl.Result{}, nil
		}
	} else {
		// the injection is being created or modified, apply needed actions
		controllerutil.AddFinalizer(instance, disruptionFinalizer)

		// If the disruption is at least r.ExpiredDisruptionGCDelay older than when its duration ended, then we should delete it.
		// calculateRemainingDurationSeconds returns the seconds until (or since, if negative) the durations deadline. We compare it to negative ExpiredDisruptionGCDelay,
		// and if less than that, it means we have exceeded the deadline by at least ExpiredDisruptionGCDelay, so we can delete
		if calculateRemainingDuration(*instance) <= (-1 * r.ExpiredDisruptionGCDelay) {
			r.log.Infow("disruption has lived for more than its duration, it will now be deleted.", "duration", instance.Spec.Duration)
			r.Recorder.Event(instance, "Normal", "DurationOver", fmt.Sprintf("The disruption has lived %s longer than its specified duration, and will now be deleted.", r.ExpiredDisruptionGCDelay))

			var err error

			if err = r.Client.Delete(context.Background(), instance); err != nil {
				r.log.Errorw("error deleting disruption after its duration expired", "error", err)
			}

			return ctrl.Result{Requeue: true}, err
		}

		// retrieve targets from label selector
		if err := r.selectTargets(instance); err != nil {
			r.log.Errorw("error selecting targets", "error", err)

			return ctrl.Result{}, fmt.Errorf("error selecting targets: %w", err)
		}

		if err := r.validateDisruptionSpec(instance); err != nil {
			return ctrl.Result{Requeue: false}, err
		}

		// start injections
		if err := r.startInjection(instance); err != nil {
			r.log.Errorw("error injecting the disruption", "error", err)

			return ctrl.Result{}, fmt.Errorf("error injecting the disruption: %w", err)
		}

		// send injection duration metric representing the time it took to fully inject the disruption until its creation
		r.handleMetricSinkError(r.MetricsSink.MetricInjectDuration(time.Since(instance.ObjectMeta.CreationTimestamp.Time), []string{"name:" + instance.Name, "namespace:" + instance.Namespace}))

		// update resource status injection
		// requeue the request if the disruption is not fully injected yet
		injected, err := r.updateInjectionStatus(instance)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("error updating disruption injection status: %w", err)
		} else if !injected {
			r.log.Infow("disruption is not fully injected yet, requeuing")

			return ctrl.Result{Requeue: true}, nil
		}
		requeueDelay := time.Duration(math.Max(calculateRemainingDuration(*instance).Seconds(), r.ExpiredDisruptionGCDelay.Seconds())) * time.Second

		r.log.Infow("requeuing disruption", "requeueDelay", requeueDelay.String())

		return ctrl.Result{
				Requeue:      true,
				RequeueAfter: requeueDelay,
			},
			r.Update(context.Background(), instance)
	}

	// stop the reconcile loop, there's nothing else to do
	return ctrl.Result{}, nil
}

// updateInjectionStatus updates the given instance injection status depending on its chaos pods statuses
// - an instance with all chaos pods "ready" is considered as "injected"
// - an instance with at least one chaos pod as "ready" is considered as "partially injected"
// - an instance with no ready chaos pods is considered as "not injected"
func (r *DisruptionReconciler) updateInjectionStatus(instance *chaosv1beta1.Disruption) (bool, error) {
	r.log.Infow("updating injection status")

	status := chaostypes.DisruptionInjectionStatusNotInjected
	allReady := true

	// get chaos pods
	chaosPods, err := r.getChaosPods(instance, nil)
	if err != nil {
		return false, fmt.Errorf("error getting instance chaos pods: %w", err)
	}

	if calculateRemainingDuration(*instance) < 0 {
		status = chaostypes.DisruptionInjectionStatusPreviouslyInjected
	}

	// consider a disruption not injected if no chaos pods are existing
	if status == chaostypes.DisruptionInjectionStatusNotInjected && len(chaosPods) > 0 {
		// check the chaos pods conditions looking for the ready condition
		for _, chaosPod := range chaosPods {
			podReady := false

			// search for the "Ready" condition in the pod conditions
			// consider the disruption "partially injected" if we found at least one ready pod
			for _, cond := range chaosPod.Status.Conditions {
				if cond.Type == corev1.PodReady {
					if cond.Status == corev1.ConditionTrue {
						podReady = true
						status = chaostypes.DisruptionInjectionStatusPartiallyInjected

						break
					}
				}
			}

			// consider the disruption as not fully injected if at least one not ready pod is found
			if !podReady {
				r.log.Infow("chaos pod is not ready yet", "chaosPod", chaosPod.Name)

				allReady = false
			}
		}

		// consider the disruption as fully injected when all pods are ready
		if allReady {
			status = chaostypes.DisruptionInjectionStatusInjected
		}
	}

	// update instance status
	instance.Status.InjectionStatus = status

	if err := r.Client.Update(context.Background(), instance); err != nil {
		return false, err
	}

	// requeue the request if the disruption is not fully injected so we can
	// eventually catch pods that are not ready yet but will be in the future
	if status != chaostypes.DisruptionInjectionStatusInjected {
		return false, nil
	}

	return status == chaostypes.DisruptionInjectionStatusInjected || status == chaostypes.DisruptionInjectionStatusPreviouslyInjected, nil
}

// startInjection creates non-existing chaos pod for the given disruption
func (r *DisruptionReconciler) startInjection(instance *chaosv1beta1.Disruption) error {
	var err error

	r.log.Infow("starting targets injection", "targets", instance.Status.Targets)

	for _, target := range instance.Status.Targets {
		targetNodeName := ""
		targetContainerIDs := []string{}
		targetPodIP := ""
		chaosPods := []*corev1.Pod{}

		// retrieve target
		switch instance.Spec.Level {
		case chaostypes.DisruptionLevelUnspecified, chaostypes.DisruptionLevelPod:
			pod := corev1.Pod{}

			if err := r.Get(context.Background(), types.NamespacedName{Namespace: instance.Namespace, Name: target}, &pod); err != nil {
				return fmt.Errorf("error getting target to inject: %w", err)
			}

			targetNodeName = pod.Spec.NodeName

			// get IDs of targeted containers or all containers
			targetContainerIDs, err = getContainerIDs(&pod, instance.Spec.Containers)
			if err != nil {
				return fmt.Errorf("error getting target pod container ID: %w", err)
			}

			// get IP of targeted pod
			targetPodIP = pod.Status.PodIP
		case chaostypes.DisruptionLevelNode:
			targetNodeName = target
		}

		// generate injection pods specs
		r.generateChaosPods(instance, &chaosPods, target, targetNodeName, targetContainerIDs, targetPodIP)

		if len(chaosPods) == 0 {
			r.Recorder.Event(instance, corev1.EventTypeWarning, "EmptyDisruption", fmt.Sprintf("No disruption recognized for \"%s\" therefore no disruption applied.", instance.Name))
			return nil
		}

		// create injection pods
		for _, chaosPod := range chaosPods {
			// check if an injection pod already exists for the given (instance, namespace, disruption kind) tuple
			found, err := r.getChaosPods(instance, chaosPod.Labels)
			if err != nil {
				return fmt.Errorf("error getting existing chaos pods: %w", err)
			}

			// create injection pods if none have been found
			if len(found) == 0 {
				r.log.Infow("creating chaos pod", "target", target)

				// create the pod
				if err = r.Create(context.Background(), chaosPod); err != nil {
					r.Recorder.Event(instance, corev1.EventTypeWarning, "CreateFailed", fmt.Sprintf("Injection pod for disruption \"%s\" failed to be created", instance.Name))
					r.handleMetricSinkError(r.MetricsSink.MetricPodsCreated(target, instance.Name, instance.Namespace, false))

					return fmt.Errorf("error creating chaos pod: %w", err)
				}

				// wait for the pod to be existing
				if err := r.waitForPodCreation(chaosPod); err != nil {
					r.log.Errorw("error waiting for chaos pod to be created", "error", err, "chaosPod", chaosPod.Name, "target", target)

					continue
				}

				// send metrics and events
				r.Recorder.Event(instance, corev1.EventTypeNormal, "Created", fmt.Sprintf("Created disruption injection pod for \"%s\"", instance.Name))
				r.recordEventOnTarget(instance, target, corev1.EventTypeWarning, "Disrupted", fmt.Sprintf("Pod %s from disruption %s targeted this resourcer for injection", chaosPod.Name, instance.Name))
				r.handleMetricSinkError(r.MetricsSink.MetricPodsCreated(target, instance.Name, instance.Namespace, true))
			} else {
				var chaosPodNames []string
				for _, pod := range found {
					chaosPodNames = append(chaosPodNames, pod.Name)
				}
				r.log.Infow("an injection pod is already existing for the selected target", "target", target, "chaosPods", strings.Join(chaosPodNames, ","))
			}
		}
	}

	return nil
}

// waitForPodCreation waits for the given pod to be created
// it tries to get the pod using an exponential backoff with a max retry interval of 1 second and a max duration of 30 seconds
// if an unexpected error occurs (an error other than a "not found" error), the retry loop is stopped
func (r *DisruptionReconciler) waitForPodCreation(pod *corev1.Pod) error {
	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.MaxInterval = time.Second
	expBackoff.MaxElapsedTime = 30 * time.Second

	return backoff.Retry(func() error {
		err := r.Get(context.Background(), types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}, pod)
		if client.IgnoreNotFound(err) != nil {
			return backoff.Permanent(err)
		}

		return err
	}, expBackoff)
}

// cleanDisruption triggers the cleanup of the given instance
// for each target and existing chaos pod, it'll take actions depending on the chaos pod status:
//   - a running chaos pod will be deleted (triggering the cleanup phase)
//   - a succeeded chaos pod (which has been deleted and has finished correctly) will see its finalizer removed (and then garbage collected)
//   - a failed chaos pod will trigger the "stuck on removal" status of the disruption instance and will block its deletion
// the function returns true when (and only when) all chaos pods have been successfully removed
// if all pods have completed but are still present (because the finalizer has not been removed yet), it'll still return false
func (r *DisruptionReconciler) cleanDisruption(instance *chaosv1beta1.Disruption) (bool, error) {
	cleaned := true

	// get already existing chaos pods for the given disruption
	chaosPods, err := r.getChaosPods(instance, nil)
	if err != nil {
		return false, err
	}

	// if chaos pods still exist, even if they are completed
	// we consider the disruption as not cleaned
	if len(chaosPods) > 0 {
		cleaned = false
	}

	// terminate running chaos pods to trigger cleanup
	for _, chaosPod := range chaosPods {
		// delete the chaos pod only if it has not been deleted already
		if chaosPod.DeletionTimestamp == nil || chaosPod.DeletionTimestamp.Time.IsZero() {
			r.log.Infow("terminating chaos pod to trigger cleanup", "chaosPod", chaosPod.Name)

			if err := r.Client.Delete(context.Background(), &chaosPod); client.IgnoreNotFound(err) != nil {
				r.log.Errorw("error terminating chaos pod", "error", err, "chaosPod", chaosPod.Name)
			}
		}
	}

	return cleaned, nil
}

// handleChaosPodsTermination looks at the given instance chaos pods status to handle any terminated pods
// such pods will have their finalizer removed so they can be garbage collected by Kubernetes
// the finalizer is removed if:
//   - the pod is pending
//   - the pod is succeeded (exit code == 0)
//   - the pod target is not heatlhy (not existing anymore for instance)
// if a finalizer can't be removed because none of the conditions above are fulfilled, the instance is flagged
// as stuck on removal and the pod finalizer won't be removed unless someone does it manually
// the pod target will be moved to ignored targets so it is not picked up by the next reconcile loop
func (r *DisruptionReconciler) handleChaosPodsTermination(instance *chaosv1beta1.Disruption) error {
	// get already existing chaos pods for the given disruption
	chaosPods, err := r.getChaosPods(instance, nil)
	if err != nil {
		return err
	}

	for _, chaosPod := range chaosPods {
		removeFinalizer := false
		ignoreStatus := false
		target := chaosPod.Labels[chaostypes.TargetLabel]

		// ignore chaos pods not being deleted or not having the finalizer anymore
		if chaosPod.DeletionTimestamp == nil || chaosPod.DeletionTimestamp.IsZero() || !controllerutil.ContainsFinalizer(&chaosPod, chaosPodFinalizer) {
			continue
		}

		// move chaos pods target to ignored targets so it is not reselected after
		if err := r.ignoreTarget(instance, target); err != nil {
			r.log.Errorw("error ignoring chaos pod target", "error", err, "target", target, "chaosPod", chaosPod.Name)

			continue
		}

		// check target readiness for cleanup
		// ignore it if it is not ready anymore
		err := r.TargetSelector.TargetIsHealthy(target, r.Client, instance)
		if err != nil {
			if errors.IsNotFound(err) || strings.ToLower(err.Error()) == "pod is not running" || strings.ToLower(err.Error()) == "node is not ready" {
				// if the target is not in a good shape, we still run the cleanup phase but we don't check for any issues happening during
				// the cleanup to avoid blocking the disruption deletion for nothing
				r.log.Infow("target is not likely to be cleaned (either it does not exist anymore or it is not ready), the injector will TRY to clean it but will not take care about any failures", "target", target)

				// by enabling this, we will remove the target associated chaos pods finalizers and delete them to trigger the cleanup phase
				// but the chaos pods status will not be checked
				ignoreStatus = true
			} else {
				r.log.Error(err.Error())

				continue
			}
		}

		// check the chaos pod status to determine if we can safely delete it or not
		switch chaosPod.Status.Phase {
		case corev1.PodSucceeded, corev1.PodPending:
			// pod has terminated or is pending
			// we can remove the pod and the finalizer and it'll be garbage collected
			removeFinalizer = true
		case corev1.PodFailed:
			// pod has failed
			// we need to determine if we can remove it safely or if we need to block disruption deletion
			// check if a container has been created (if not, the disruption was not injected)
			if len(chaosPod.Status.ContainerStatuses) == 0 {
				removeFinalizer = true
			}

			// if the pod died only because it exceeded its activeDeadlineSeconds, we can remove the finalizer
			if chaosPod.Status.Reason == "DeadlineExceeded" {
				removeFinalizer = true
			}

			// check if the container was able to start or not
			// if not, we can safely delete the pod since the disruption was not injected
			for _, cs := range chaosPod.Status.ContainerStatuses {
				if cs.Name == "injector" {
					if cs.State.Terminated != nil && cs.State.Terminated.Reason == "StartError" {
						removeFinalizer = true
					}

					break
				}
			}
		default:
			if !ignoreStatus {
				// ignoring any pods not being in a "terminated" state
				// if the target is not healthy, we clean up this pod regardless of its state
				continue
			}
		}

		// remove the finalizer if possible or if we can ignore the cleanup status
		if removeFinalizer || ignoreStatus {
			r.log.Infow("chaos pod completed, removing finalizer", "target", target, "chaosPod", chaosPod.Name)

			controllerutil.RemoveFinalizer(&chaosPod, chaosPodFinalizer)

			if err := r.Client.Update(context.Background(), &chaosPod); err != nil {
				r.log.Errorw("error removing chaos pod finalizer", "error", err, "chaosPod", chaosPod.Name)

				continue
			}
		} else {
			// if the chaos pod finalizer must not be removed and the chaos pod must not be deleted
			// and the cleanup status must not be ignored, we are stuck and won't be able to remove the disruption
			r.log.Infow("instance seems stuck on removal for this target, please check manually", "target", target, "chaosPod", chaosPod.Name)
			r.Recorder.Event(instance, corev1.EventTypeWarning, "StuckOnRemoval", "Instance is stuck on removal because of chaos pods not being able to terminate correctly, please check pods logs before manually removing their finalizer")

			instance.Status.IsStuckOnRemoval = true
		}
	}

	return r.Update(context.Background(), instance)
}

// ignoreTarget moves the given target from the list of targets to the list of ignored targets
// so it is not picked up during targets selection
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

// selectTargets will select min(count, all matching targets) random targets (pods or nodes depending on the disruption level)
// from the targets matching the instance label selector
// targets will only be selected once per instance
// the chosen targets names will be reflected in the intance status
// subsequent calls to this function will always return the same targets as the first call
func (r *DisruptionReconciler) selectTargets(instance *chaosv1beta1.Disruption) error {
	matchingTargets := []string{}

	// exit early if we already have targets selected for the given instance
	if len(instance.Status.Targets) > 0 {
		return nil
	}

	r.log.Infow("selecting targets to inject disruption to", "selector", instance.Spec.Selector.String())

	// validate the given label selector to avoid any formating issues due to special chars
	if instance.Spec.Selector != nil {
		if err := validateLabelSelector(instance.Spec.Selector.AsSelector()); err != nil {
			r.Recorder.Event(instance, corev1.EventTypeWarning, "InvalidLabelSelector", fmt.Sprintf("%s. No targets will be selected.", err.Error()))

			return err
		}
	}

	// select either pods or nodes depending on the disruption level
	switch instance.Spec.Level {
	case chaostypes.DisruptionLevelUnspecified, chaostypes.DisruptionLevelPod:
		pods, err := r.TargetSelector.GetMatchingPods(r.Client, instance)
		if err != nil {
			return fmt.Errorf("can't get pods matching the given label selector: %w", err)
		}

		for _, pod := range pods.Items {
			matchingTargets = append(matchingTargets, pod.Name)
		}
	case chaostypes.DisruptionLevelNode:
		nodes, err := r.TargetSelector.GetMatchingNodes(r.Client, instance)
		if err != nil {
			return fmt.Errorf("can't get pods matching the given label selector: %w", err)
		}

		for _, node := range nodes.Items {
			matchingTargets = append(matchingTargets, node.Name)
		}
	}

	// return an error if the selector returned no targets
	if len(matchingTargets) == 0 {
		r.log.Info("the given label selector did not return any targets, skipping")
		r.Recorder.Event(instance, corev1.EventTypeWarning, "NoTarget", "The given label selector did not return any targets. Please ensure that both the selector and the count are correct (should be either a percentage or an integer greater than 0).")

		return nil
	}

	// instance.Spec.Count is a string that either represents a percentage or a value, we do the translation here
	targetsCount, err := getScaledValueFromIntOrPercent(instance.Spec.Count, len(matchingTargets), true)
	if err != nil {
		targetsCount = instance.Spec.Count.IntValue()
	}

	// subtract already ignored targets from the targets count to avoid going through all the
	// eligible targets with a disruption having a chaos pod failing everytime
	// so a disruption having a count of 1 with an already ignored target (because the chaos pod has been removed)
	// won't pick up another one
	targetsCount -= len(instance.Status.IgnoredTargets)

	// filter matching targets to only get eligible ones
	eligibleTargets, err := r.getEligibleTargets(instance, matchingTargets)
	if err != nil {
		r.log.Errorw("error getting eligible targets", "error", err)

		return fmt.Errorf("error getting eligible targets: %w", err)
	}

	// if the asked targets count is greater than the amount of found targets, we take all of them
	targetsCount = int(math.Min(float64(targetsCount), float64(len(eligibleTargets))))
	if targetsCount < 1 {
		r.log.Info("ignored targets has reached target count, skipping")
		r.Recorder.Event(instance, corev1.EventTypeWarning, "NoTarget", "No more targets found for injection for this disruption (either ignored or already targeted by another disruption)")

		return nil
	}

	// randomly pick up targets from the found ones
	for i := 0; i < targetsCount; i++ {
		index := rand.Intn(len(eligibleTargets)) //nolint:gosec
		selectedTarget := eligibleTargets[index]
		instance.Status.Targets = append(instance.Status.Targets, selectedTarget)
		eligibleTargets[len(eligibleTargets)-1], eligibleTargets[index] = eligibleTargets[index], eligibleTargets[len(eligibleTargets)-1]
		eligibleTargets = eligibleTargets[:len(eligibleTargets)-1]
	}

	r.log.Infow("updating instance status with targets selected for injection")

	return r.Update(context.Background(), instance)
}

// getEligibleTargets returns targets which can be targeted by the given instance from the given targets pool
// it skips ignored targets and targets being already targeted by another disruption
func (r *DisruptionReconciler) getEligibleTargets(instance *chaosv1beta1.Disruption, targets []string) ([]string, error) {
	r.log.Info("getting eligible targets for disruption injection")

	eligibleTargets := []string{}

	for _, target := range targets {
		// skip ignored targets
		if contains(instance.Status.IgnoredTargets, target) {
			continue
		}

		// skip targets already targeted by a chaos pod from another disruption
		chaosPods, err := r.getChaosPods(nil, map[string]string{
			chaostypes.TargetLabel:              target,             // filter with target name
			chaostypes.DisruptionNamespaceLabel: instance.Namespace, // filter with current instance namespace (to avoid getting targets having the same name but living in different namespaces)
		})
		if err != nil {
			return nil, fmt.Errorf("error getting chaos pods targeting the given target (%s): %w", target, err)
		}

		if len(chaosPods) > 0 {
			r.log.Infow("target is already affected by another disruption, skipping", "target", target)

			continue
		}

		// add target if eligible
		eligibleTargets = append(eligibleTargets, target)
	}

	return eligibleTargets, nil
}

// getChaosPods returns chaos pods owned by the given instance and having the given labels
// both instance and label set are optional but at least one must be provided
func (r *DisruptionReconciler) getChaosPods(instance *chaosv1beta1.Disruption, ls labels.Set) ([]corev1.Pod, error) {
	pods := &corev1.PodList{}

	// ensure we always have at least a disruption instance or a label set to filter on
	if instance == nil && ls == nil {
		return nil, fmt.Errorf("you must specify at least a disruption instance or a label set to get chaos pods")
	}

	if ls == nil {
		ls = make(map[string]string)
	}

	// add instance specific labels if provided
	if instance != nil {
		ls[chaostypes.DisruptionNameLabel] = instance.Name
		ls[chaostypes.DisruptionNamespaceLabel] = instance.Namespace
	}

	r.log.Infow("searching for chaos pods with label selector...", "labels", ls.String())

	// list pods in the defined namespace and for the given target
	listOptions := &client.ListOptions{
		Namespace:     r.InjectorServiceAccountNamespace,
		LabelSelector: labels.SelectorFromValidatedSet(ls),
	}

	err := r.Client.List(context.Background(), pods, listOptions)
	if err != nil {
		return nil, fmt.Errorf("error listing owned pods: %w", err)
	}

	return pods.Items, nil
}

// generatePod generates a pod from a generic pod template in the same namespace
// and on the same node as the given pod
func (r *DisruptionReconciler) generatePod(instance *chaosv1beta1.Disruption, targetName string, targetNodeName string, args []string, kind chaostypes.DisruptionKindName) *corev1.Pod {
	// volume host path type definitions
	hostPathDirectory := corev1.HostPathDirectory
	hostPathFile := corev1.HostPathFile

	// The default TerminationGracePeriodSeconds is 30s. This can be too low for a chaos pod to finish cleaning. After TGPS passes,
	// the signal sent to a pod becomes SIGKILL, which will interrupt any in-progress cleaning. By double this to 1 minute in the pod spec itself,
	// ensures that whether a chaos pod is deleted directly or by deleting a disruption, it will have time to finish cleaning up after itself.
	terminationGracePeriod := int64(60)
	r.log.Infow("GOTTA SOLVE THE BUGS", "duration", instance.Spec.Duration, "creation", instance.ObjectMeta.CreationTimestamp.Time)
	activeDeadlineSeconds := int64(calculateRemainingDuration(*instance).Seconds())

	podSpec := corev1.PodSpec{
		HostPID:                       true,                      // enable host pid
		RestartPolicy:                 corev1.RestartPolicyNever, // do not restart the pod on fail or completion
		NodeName:                      targetNodeName,            // specify node name to schedule the pod
		ServiceAccountName:            r.InjectorServiceAccount,  // service account to use
		TerminationGracePeriodSeconds: &terminationGracePeriod,
		ActiveDeadlineSeconds:         &activeDeadlineSeconds,
		Containers: []corev1.Container{
			{
				Name:            "injector",              // container name
				Image:           r.InjectorImage,         // container image gathered from controller flags
				ImagePullPolicy: corev1.PullIfNotPresent, // pull the image only when it is not present
				Args:            args,                    // pass disruption arguments
				SecurityContext: &corev1.SecurityContext{
					Privileged: func() *bool { b := true; return &b }(), // enable privileged mode
				},
				ReadinessProbe: &corev1.Probe{ // define readiness probe (file created by the injector when the injection is successful)
					PeriodSeconds:    1,
					FailureThreshold: 5,
					Handler: corev1.Handler{
						Exec: &corev1.ExecAction{
							Command: []string{"cat", "/tmp/readiness_probe"},
						},
					},
				},
				Resources: corev1.ResourceRequirements{ // set resources requests and limits to zero
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    *resource.NewQuantity(0, resource.DecimalSI),
						corev1.ResourceMemory: *resource.NewQuantity(0, resource.DecimalSI),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    *resource.NewQuantity(0, resource.DecimalSI),
						corev1.ResourceMemory: *resource.NewQuantity(0, resource.DecimalSI),
					},
				},
				Env: []corev1.EnvVar{ // define environment variables
					{
						Name: env.InjectorTargetPodHostIP,
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: "status.hostIP",
							},
						},
					},
					{
						Name: env.InjectorChaosPodIP,
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: "status.podIP",
							},
						},
					},
					{
						Name:  env.InjectorMountHost,
						Value: "/mnt/host/",
					},
					{
						Name:  env.InjectorMountProc,
						Value: "/mnt/host/proc/",
					},
					{
						Name:  env.InjectorMountSysrq,
						Value: "/mnt/sysrq",
					},
					{
						Name:  env.InjectorMountSysrqTrigger,
						Value: "/mnt/sysrq-trigger",
					},
					{
						Name:  env.InjectorMountCgroup,
						Value: "/mnt/cgroup/",
					},
				},
				VolumeMounts: []corev1.VolumeMount{ // define volume mounts required for disruptions to work
					{
						Name:      "run",
						MountPath: "/run",
					},
					{
						Name:      "sysrq",
						MountPath: "/mnt/sysrq",
					},
					{
						Name:      "sysrq-trigger",
						MountPath: "/mnt/sysrq-trigger",
					},
					{
						Name:      "cgroup",
						MountPath: "/mnt/cgroup",
					},
					{
						Name:      "host",
						MountPath: "/mnt/host",
						ReadOnly:  true,
					},
				},
			},
		},
		Volumes: []corev1.Volume{ // declare volumes required for disruptions to work
			{
				Name: "run",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/run",
						Type: &hostPathDirectory,
					},
				},
			},
			{
				Name: "proc",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/proc",
						Type: &hostPathDirectory,
					},
				},
			},
			{
				Name: "sysrq",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/proc/sys/kernel/sysrq",
						Type: &hostPathFile,
					},
				},
			},
			{
				Name: "sysrq-trigger",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/proc/sysrq-trigger",
						Type: &hostPathFile,
					},
				},
			},
			{
				Name: "cgroup",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/sys/fs/cgroup",
						Type: &hostPathDirectory,
					},
				},
			},
			{
				Name: "host",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/",
						Type: &hostPathDirectory,
					},
				},
			},
		},
	}

	if r.ImagePullSecrets != "" {
		podSpec.ImagePullSecrets = []corev1.LocalObjectReference{
			{
				Name: r.ImagePullSecrets,
			},
		}
	}

	// define injector pod
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("chaos-%s-", instance.Name), // generate the pod name automatically with a prefix
			Namespace:    r.InjectorServiceAccountNamespace,       // chaos pods need to be in the same namespace as their service account to run
			Annotations:  r.InjectorAnnotations,                   // add extra annotations passed to the controller
			Labels: map[string]string{
				chaostypes.TargetLabel:              targetName,         // target name label
				chaostypes.DisruptionKindLabel:      string(kind),       // disruption kind label
				chaostypes.DisruptionNameLabel:      instance.Name,      // disruption name label, used to determine ownership
				chaostypes.DisruptionNamespaceLabel: instance.Namespace, // disruption namespace label, used to determine ownership
			},
		},
		Spec: podSpec,
	}

	// add finalizer to the pod so it is not deleted before we can control its exit status
	controllerutil.AddFinalizer(&pod, chaosPodFinalizer)

	return &pod
}

// handleMetricSinkError logs the given metric sink error if it is not nil
func (r *DisruptionReconciler) handleMetricSinkError(err error) {
	if err != nil {
		r.log.Errorw("error sending a metric", "error", err)
	}
}

func (r *DisruptionReconciler) emitKindCountMetrics(instance *chaosv1beta1.Disruption) {
	for _, kind := range instance.Spec.GetKindNames() {
		r.handleMetricSinkError((r.MetricsSink.MetricDisruptionsCount(kind, []string{"name:" + instance.Name, "namespace:" + instance.Namespace})))
	}
}

func (r *DisruptionReconciler) validateDisruptionSpec(instance *chaosv1beta1.Disruption) error {
	err := instance.Spec.Validate()
	if err != nil {
		r.Recorder.Event(instance, corev1.EventTypeWarning, "InvalidSpec", err.Error())
		return err
	}

	return nil
}

// generateChaosPods generates a chaos pod for the given instance and disruption kind if set
func (r *DisruptionReconciler) generateChaosPods(instance *chaosv1beta1.Disruption, pods *[]*corev1.Pod, targetName string, targetNodeName string, targetContainerIDs []string, targetPodIP string) {
	// generate chaos pods for each possible disruptions
	for _, kind := range chaostypes.DisruptionKindNames {
		subspec := instance.Spec.DisruptionKindPicker(kind)
		if reflect.ValueOf(subspec).IsNil() {
			continue
		}

		// default level to pod if not specified
		level := instance.Spec.Level
		if level == chaostypes.DisruptionLevelUnspecified {
			level = chaostypes.DisruptionLevelPod
		}

		// generate args for pod
		args := chaosapi.AppendArgs(subspec.GenerateArgs(),
			level, kind, targetContainerIDs, targetPodIP, r.MetricsSink.GetSinkName(), instance.Spec.DryRun,
			instance.Name, instance.Namespace, targetName, instance.Spec.OnInit, r.InjectorNetworkDisruptionAllowedHosts, r.InjectorDNSDisruptionDNSServer, r.InjectorDNSDisruptionKubeDNS)

		// append pod to chaos pods
		*pods = append(*pods, r.generatePod(instance, targetName, targetNodeName, args, kind))
	}
}

// recordEventOnTarget records an event on the given target which can be either a pod or a node depending on the given disruption level
func (r *DisruptionReconciler) recordEventOnTarget(instance *chaosv1beta1.Disruption, target string, eventtype, reason, message string) {
	r.log.Infow("registering an event on a target", "target", target, "eventtype", eventtype, "reason", reason, "message", message)

	var o runtime.Object

	switch instance.Spec.Level {
	case chaostypes.DisruptionLevelUnspecified, chaostypes.DisruptionLevelPod:
		p := &corev1.Pod{}

		if err := r.Get(context.Background(), types.NamespacedName{Namespace: instance.Namespace, Name: target}, p); err != nil {
			r.log.Errorw("event failed to be registered on target", "error", err, "target", target)
		}

		o = p
	case chaostypes.DisruptionLevelNode:
		n := &corev1.Node{}

		if err := r.Get(context.Background(), types.NamespacedName{Name: target}, n); err != nil {
			r.log.Errorw("event failed to be registered on target", "error", err, "target", target)
		}

		o = n
	}

	r.Recorder.Event(o, eventtype, reason, message)
}

// SetupWithManager setups the current reconciler with the given manager
func (r *DisruptionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chaosv1beta1.Disruption{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}

// ReportMetrics reports some controller metrics every minute:
// - stuck on removal disruptions count
// - ongoing disruptions count
func (r *DisruptionReconciler) ReportMetrics() {
	for {
		// wait for a minute
		<-time.After(time.Minute)

		// declare counters
		stuckOnRemoval := 0
		chaosPodsCount := 0

		l := chaosv1beta1.DisruptionList{}

		// list disruptions
		if err := r.Client.List(context.Background(), &l); err != nil {
			r.log.Errorw("error listing disruptions", "error", err)
			continue
		}

		// check for stuck durations, count chaos pods, and track ongoing disruption duration
		for _, d := range l.Items {
			if d.Status.IsStuckOnRemoval {
				stuckOnRemoval++

				if err := r.MetricsSink.MetricStuckOnRemoval([]string{"name:" + d.Name, "namespace:" + d.Namespace}); err != nil {
					r.log.Errorw("error sending stuck_on_removal metric", "error", err)
				}
			}

			chaosPods, err := r.getChaosPods(&d, nil)
			if err != nil {
				r.log.Errorw("error listing chaos pods to send pods.gauge metric", "error", err)
			}

			chaosPodsCount += len(chaosPods)

			r.handleMetricSinkError(r.MetricsSink.MetricDisruptionOngoingDuration(time.Since(d.ObjectMeta.CreationTimestamp.Time), []string{"name:" + d.Name, "namespace:" + d.Namespace}))
		}

		// send metrics
		if err := r.MetricsSink.MetricStuckOnRemovalGauge(float64(stuckOnRemoval)); err != nil {
			r.log.Errorw("error sending stuck_on_removal_total metric", "error", err)
		}

		if err := r.MetricsSink.MetricDisruptionsGauge(float64(len(l.Items))); err != nil {
			r.log.Errorw("error sending disruptions.gauge metric", "error", err)
		}

		if err := r.MetricsSink.MetricPodsGauge(float64(chaosPodsCount)); err != nil {
			r.log.Errorw("error sending pods.gauge metric", "error", err)
		}
	}
}
