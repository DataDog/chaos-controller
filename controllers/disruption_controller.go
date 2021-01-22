// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"reflect"
	"time"

	"github.com/cenkalti/backoff"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	chaosapi "github.com/DataDog/chaos-controller/api"
	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/metrics"
	chaostypes "github.com/DataDog/chaos-controller/types"
)

const (
	finalizerPrefix = "finalizer.chaos.datadoghq.com"
)

var (
	disruptionFinalizer = finalizerPrefix
	chaosPodFinalizer   = finalizerPrefix + "/chaos-pod"
)

// DisruptionReconciler reconciles a Disruption object
type DisruptionReconciler struct {
	client.Client
	Log             *zap.SugaredLogger
	Scheme          *runtime.Scheme
	Recorder        record.EventRecorder
	MetricsSink     metrics.Sink
	PodTemplateSpec corev1.Pod
	TargetSelector  TargetSelector
}

// +kubebuilder:rbac:groups=chaos.datadoghq.com,resources=disruptions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=chaos.datadoghq.com,resources=disruptions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=list;watch

// Reconcile loop
func (r *DisruptionReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	instance := &chaosv1beta1.Disruption{}
	tsStart := time.Now()

	rand.Seed(time.Now().UnixNano())

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
	r.Log.Infow("fetching disruption instance", "instance", req.Name, "namespace", req.Namespace)

	if err := r.Get(context.Background(), req.NamespacedName, instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
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
				requeueAfter := time.Duration(rand.Intn(5)+5) * time.Second

				r.Log.Infow(fmt.Sprintf("disruption has not been fully cleaned yet, re-queuing in %v", requeueAfter), "instance", instance.Name, "namespace", instance.Namespace)

				return ctrl.Result{
					Requeue:      true,
					RequeueAfter: requeueAfter,
				}, r.Update(context.Background(), instance)
			}

			// we reach this code when all the cleanup pods have succeeded
			// we can remove the finalizer and let the resource being garbage collected
			r.Log.Infow("removing finalizer", "instance", instance.Name, "namespace", instance.Namespace)
			r.handleMetricSinkError(r.MetricsSink.MetricCleanupDuration(time.Since(instance.ObjectMeta.DeletionTimestamp.Time), []string{"name:" + instance.Name, "namespace:" + instance.Namespace}))

			controllerutil.RemoveFinalizer(instance, disruptionFinalizer)

			return ctrl.Result{}, r.Update(context.Background(), instance)
		}
	} else {
		// the injection is being created or modified, apply needed actions
		controllerutil.AddFinalizer(instance, disruptionFinalizer)

		// compute spec hash to detect any changes in the spec and warn the user about it
		sameHashes, err := r.computeHash(instance)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("error computing instance spec hash: %w", err)
		} else if !sameHashes {
			r.Log.Infow("instance spec hash has changed meaning a change to the spec has been made, aborting")

			return ctrl.Result{}, nil
		}

		// retrieve targets from label selector
		if err := r.selectTargets(instance); err != nil {
			r.Log.Errorw("error selecting targets", "error", err, "instance", instance.Name, "namespace", instance.Namespace)

			return ctrl.Result{}, fmt.Errorf("error selecting targets: %w", err)
		}

		err = r.validateDisruptionSpec(instance)
		if err != nil {
			return ctrl.Result{Requeue: false}, err
		}

		// start injections
		if err := r.startInjection(instance); err != nil {
			r.Log.Errorw("error injecting the disruption", "error", err, "instance", instance.Name, "namespace", instance.Namespace)

			return ctrl.Result{}, fmt.Errorf("error injecting the disruption: %w", err)
		}

		// update resource status injection
		// requeue the request if the disruption is not fully injected yet
		injected, err := r.updateInjectionStatus(instance)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("error updating disruption injection status: %w", err)
		} else if !injected {
			r.Log.Infow("disruption is not fully injected yet, requeing", "instance", instance.Name, "namespace", instance.Namespace)

			return ctrl.Result{Requeue: true}, nil
		}

		// send injection duration metric representing the time it took to fully inject the disruption until its creation
		r.handleMetricSinkError(r.MetricsSink.MetricInjectDuration(time.Since(instance.ObjectMeta.CreationTimestamp.Time), []string{"name:" + instance.Name, "namespace:" + instance.Namespace}))

		return ctrl.Result{}, r.Update(context.Background(), instance)
	}

	// stop the reconcile loop, there's nothing else to do
	return ctrl.Result{}, nil
}

// updateInjectionStatus updates the given instance injection status depending on its chaos pods statuses
// - an instance with all chaos pods "ready" is considered as "injected"
// - an instance with at least one chaos pod as "ready" is considered as "partially injected"
// - an instance with no ready chaos pods is considered as "not injected"
func (r *DisruptionReconciler) updateInjectionStatus(instance *chaosv1beta1.Disruption) (bool, error) {
	r.Log.Infow("updating injection status", "instance", instance.Name, "namespace", instance.Namespace)

	status := chaostypes.DisruptionInjectionStatusNotInjected
	allReady := true

	// get chaos pods
	chaosPods, err := r.getChaosPods(instance, nil)
	if err != nil {
		return false, fmt.Errorf("error getting instance chaos pods: %w", err)
	}

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
			r.Log.Infow("chaos pod is not ready yet", "instance", instance.Name, "namespace", instance.Namespace, "chaosPod", chaosPod.Name)

			allReady = false
		}
	}

	// consider the disruption as fully injected when all pods are ready
	if allReady {
		status = chaostypes.DisruptionInjectionStatusInjected
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

	return true, nil
}

// startInjection creates non-existing chaos pod for the given disruption
func (r *DisruptionReconciler) startInjection(instance *chaosv1beta1.Disruption) error {
	var err error

	r.Log.Infow("starting targets injection", "instance", instance.Name, "namespace", instance.Namespace, "targets", instance.Status.Targets)

	for _, target := range instance.Status.Targets {
		var targetNodeName, containerID string

		chaosPods := []*corev1.Pod{}

		// retrieve target
		switch instance.Spec.Level {
		case chaostypes.DisruptionLevelUnspecified, chaostypes.DisruptionLevelPod:
			pod := corev1.Pod{}

			if err := r.Get(context.Background(), types.NamespacedName{Namespace: instance.Namespace, Name: target}, &pod); err != nil {
				return fmt.Errorf("error getting target to inject: %w", err)
			}

			targetNodeName = pod.Spec.NodeName

			// get ID of targeted container or the first container
			containerID, err = getContainerID(&pod, instance.Spec.Container)
			if err != nil {
				return fmt.Errorf("error getting target pod container ID: %w", err)
			}
		case chaostypes.DisruptionLevelNode:
			targetNodeName = target
		}

		// generate injection pods specs
		if err := r.generateChaosPods(instance, &chaosPods, target, targetNodeName, containerID); err != nil {
			r.Log.Errorw("error generating injection chaos pod for target, skipping it", "error", err, "instance", instance.Name, "namespace", instance.Namespace, "target", target)

			continue
		}

		if len(chaosPods) == 0 {
			r.Recorder.Event(instance, "Warning", "Empty Disruption", fmt.Sprintf("No disruption recognized for \"%s\" therefore no disruption applied.", instance.Name))

			return nil
		}

		// create injection pods
		for _, chaosPod := range chaosPods {
			// link instance resource and injection pod for garbage collection
			if err := controllerutil.SetControllerReference(instance, chaosPod, r.Scheme); err != nil {
				return fmt.Errorf("error setting chaos pod owner reference: %w", err)
			}

			// check if an injection pod already exists for the given (instance, namespace, disruption kind) tuple
			found, err := r.getChaosPods(instance, chaosPod.Labels)
			if err != nil {
				return fmt.Errorf("error getting existing chaos pods: %w", err)
			}

			// create injection pods if none have been found
			if len(found) == 0 {
				r.Log.Infow("creating chaos pod", "instance", instance.Name, "namespace", instance.Namespace, "target", target)

				// create the pod
				if err = r.Create(context.Background(), chaosPod); err != nil {
					r.Recorder.Event(instance, "Warning", "Create failed", fmt.Sprintf("Injection pod for disruption \"%s\" failed to be created", instance.Name))
					r.handleMetricSinkError(r.MetricsSink.MetricPodsCreated(target, instance.Name, instance.Namespace, "inject", false))

					return fmt.Errorf("error creating chaos pod: %w", err)
				}

				// wait for the pod to be existing
				if err := r.waitForPodCreation(chaosPod); err != nil {
					r.Log.Errorw("error waiting for chaos pod to be created", "error", err, "instance", instance.Name, "namespace", instance.Namespace, "chaosPod", chaosPod.Name, "target", target)

					continue
				}

				// send metrics and events
				r.Recorder.Event(instance, "Normal", "Created", fmt.Sprintf("Created disruption injection pod for \"%s\"", instance.Name))
				r.recordEventOnTarget(instance, target, "Warning", "Disrupted", fmt.Sprintf("Pod %s from disruption %s targeted this resourcer for injection", chaosPod.Name, instance.Name))
				r.handleMetricSinkError(r.MetricsSink.MetricPodsCreated(target, instance.Name, instance.Namespace, "inject", true))
			} else {
				r.Log.Infow("an injection pod is already existing for the selected target", "instance", instance.Name, "namespace", instance.Namespace, "target", target, "chaosPod", chaosPod.Name)
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

// computeHash computes the given instance spec hash and returns true if both hashes (the computed one and the stored one) are the same
// if it is not present in the instance status yet, the hash is stored and the function returns true
// if it is present, both hash are compared: if they are different, a modification has been made to the disruption and the function returns false
func (r *DisruptionReconciler) computeHash(instance *chaosv1beta1.Disruption) (bool, error) {
	// serialize instance spec to JSON and compute bytes hash
	specBytes, err := json.Marshal(instance.Spec)
	if err != nil {
		return false, fmt.Errorf("error serializing instance spec: %w", err)
	}

	specHash := fmt.Sprintf("%x", md5.Sum(specBytes))

	// compare computed and stored hashes if present
	// register an event in the instance if hash are different
	if instance.Status.SpecHash != nil {
		if *instance.Status.SpecHash != specHash {
			r.Recorder.Event(instance, "Warning", "Mutated", "A mutation in the disruption spec has been detected. This resource is immutable and changes have no effect. Please delete and re-create the resource for changes to be effective.")

			return false, nil
		}
	} else {
		// store the computed hash
		r.Log.Infow("storing resource spec hash to detect further changes in spec", "instance", instance.Name, "namespace", instance.Namespace)
		instance.Status.SpecHash = &specHash

		return true, r.Update(context.Background(), instance)
	}

	return true, nil
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

	for _, target := range instance.Status.Targets {
		ignoreCleanupStatus := false

		// check target readiness for cleanup
		// ignore it if it is not ready anymore
		err := r.TargetSelector.TargetIsHealthy(target, r.Client, instance)
		if err != nil {
			if errors.IsNotFound(err) || err.Error() == "Pod is not Running" || err.Error() == "Node is not Ready" {
				// if the target is not in a good shape, we still run the cleanup phase but we don't check for any issues happening during
				// the cleanup to avoid blocking the disruption deletion for nothing
				r.Log.Infow("target is not likely to be cleaned (either it does not exist anymore or it is not ready), the injector will TRY to clean it but will not take care about any failures", "instance", instance.Name, "namespace", instance.Namespace, "target", target)

				// by enabling this, we will remove the target associated chaos pods finalizers and delete them to trigger the cleanup phase
				// but the chaos pods status will not be checked
				ignoreCleanupStatus = true
			} else {
				r.Log.Error(err.Error())

				continue
			}
		}

		// get already existing cleanup pods for the specific disruption and target
		chaosPods, err := r.getChaosPods(instance, map[string]string{chaostypes.TargetLabel: target})
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
			// if the chaos pod has succeeded, remove the finalizer so it can be garbage collected
			if chaosPod.Status.Phase == corev1.PodSucceeded || chaosPod.Status.Phase == corev1.PodPending || ignoreCleanupStatus {
				r.Log.Infow("chaos pod completed, removing finalizer", "instance", instance.Name, "target", target, "chaosPod", chaosPod.Name)

				controllerutil.RemoveFinalizer(&chaosPod, chaosPodFinalizer)

				if err := r.Client.Update(context.Background(), &chaosPod); err != nil {
					r.Log.Errorw("error removing chaos pod finalizer", "error", err, "instance", instance.Name, "target", target, "chaosPod", chaosPod.Name)
				}
			}

			// if the chaos pod is running or pending, delete it to trigger the termination and the cleanup
			if chaosPod.Status.Phase == corev1.PodRunning || chaosPod.Status.Phase == corev1.PodPending {
				r.Log.Infow("terminating chaos pod to trigger cleanup", "instance", instance.Name, "target", target, "chaosPod", chaosPod.Name)

				if err := r.Client.Delete(context.Background(), &chaosPod); client.IgnoreNotFound(err) != nil {
					r.Log.Errorw("error terminating chaos pod", "error", err, "instance", instance.Name, "target", target, "chaosPod", chaosPod.Name)
				}
			}

			// if the chaos pod has failed, declare the disruption as stuck on removal so it can be investigated
			if chaosPod.Status.Phase == corev1.PodFailed && !ignoreCleanupStatus {
				r.Log.Infow("instance seems stuck on removal for this target, please check manually", "instance", instance.Name, "target", target, "chaosPod", chaosPod.Name)
				r.Recorder.Event(instance, "Warning", "StuckOnRemoval", "Instance is stuck on removal because of chaos pods not being able to terminate correctly, please check pods logs before manually removing their finalizer")

				instance.Status.IsStuckOnRemoval = true
			}
		}
	}

	return cleaned, nil
}

// selectTargets will select min(count, all matching targets) random targets (pods or nodes depending on the disruption level)
// from the targets matching the instance label selector
// targets will only be selected once per instance
// the chosen targets names will be reflected in the intance status
// subsequent calls to this function will always return the same targets as the first call
func (r *DisruptionReconciler) selectTargets(instance *chaosv1beta1.Disruption) error {
	allTargets := []string{}

	// exit early if we already have targets selected for the given instance
	if len(instance.Status.Targets) > 0 {
		return nil
	}

	r.Log.Infow("selecting targets to inject disruption to", "instance", instance.Name, "namespace", instance.Namespace, "selector", instance.Spec.Selector.String())

	// validate the given label selector to avoid any formating issues due to special chars
	if err := validateLabelSelector(instance.Spec.Selector.AsSelector()); err != nil {
		r.Recorder.Event(instance, "Warning", "InvalidLabelSelector", fmt.Sprintf("%s. No targets will be selected.", err.Error()))

		return err
	}

	// select either pods or nodes depending on the disruption level
	switch instance.Spec.Level {
	case chaostypes.DisruptionLevelUnspecified, chaostypes.DisruptionLevelPod:
		pods, err := r.TargetSelector.GetMatchingPods(r.Client, instance)
		if err != nil {
			return fmt.Errorf("can't get pods matching the given label selector: %w", err)
		}

		for _, pod := range pods.Items {
			allTargets = append(allTargets, pod.Name)
		}
	case chaostypes.DisruptionLevelNode:
		nodes, err := r.TargetSelector.GetMatchingNodes(r.Client, instance)
		if err != nil {
			return fmt.Errorf("can't get pods matching the given label selector: %w", err)
		}

		for _, node := range nodes.Items {
			allTargets = append(allTargets, node.Name)
		}
	}

	// return an error if the selector returned no targets
	if len(allTargets) == 0 {
		r.Recorder.Event(instance, "Warning", "NoTarget", "The given label selector did not return any targets. Please ensure that both the selector and the count are correct (should be either a percentage or an integer greater than 0).")

		return fmt.Errorf("the label selector returned no targets")
	}

	// instance.Spec.Count is a string that either represents a percentage or a value, we do the translation here
	targetsCount, err := getScaledValueFromIntOrPercent(instance.Spec.Count, len(allTargets), true)
	if err != nil {
		targetsCount = instance.Spec.Count.IntValue()
	}

	// computed count should not be 0 unless the given count was not expected
	if targetsCount == 0 {
		return fmt.Errorf("parsing error, either incorrectly formatted percentage or incorrectly formatted integer: %s\n%w", instance.Spec.Count.String(), err)
	}

	// if the asked targets count is greater than the amount of found targets, we take all of them
	targetsCount = int(math.Min(float64(targetsCount), float64(len(allTargets))))

	// randomly pick up targets from the found ones
	for i := 0; i < targetsCount; i++ {
		index := rand.Intn(len(allTargets))
		selectedTarget := allTargets[index]
		instance.Status.Targets = append(instance.Status.Targets, selectedTarget)
		allTargets[len(allTargets)-1], allTargets[index] = allTargets[index], allTargets[len(allTargets)-1]
		allTargets = allTargets[:len(allTargets)-1]
	}

	r.Log.Infow("updating instance status with targets selected for injection", "instance", instance.Name, "namespace", instance.Namespace)

	return r.Update(context.Background(), instance)
}

// getChaosPods returns chaos pods owned by the given instance and having the given labels
func (r *DisruptionReconciler) getChaosPods(instance *chaosv1beta1.Disruption, ls labels.Set) ([]corev1.Pod, error) {
	ownedPods := make([]corev1.Pod, 0)
	pods := &corev1.PodList{}

	// list pods in the instance namespace and for the given target
	listOptions := &client.ListOptions{
		Namespace:     instance.Namespace,
		LabelSelector: labels.SelectorFromSet(ls),
	}

	err := r.Client.List(context.Background(), pods, listOptions)
	if err != nil {
		return nil, fmt.Errorf("error listing owned pods: %w", err)
	}

	// filter all pods in the same namespace as instance,
	// only returning those owned by the given instance
	for _, pod := range pods.Items {
		if metav1.IsControlledBy(&pod, instance) {
			ownedPods = append(ownedPods, pod)
		}
	}

	return ownedPods, nil
}

// generatePod generates a pod from a generic pod template in the same namespace
// and on the same node as the given pod
func (r *DisruptionReconciler) generatePod(instance *chaosv1beta1.Disruption, targetName string, targetNodeName string, args []string, kind chaostypes.DisruptionKind) (*corev1.Pod, error) {
	pod := corev1.Pod{}

	image, ok := os.LookupEnv("CHAOS_INJECTOR_IMAGE")
	if !ok {
		image = "chaos-injector"
	}

	// copy pod template
	data, err := json.Marshal(r.PodTemplateSpec)
	if err != nil {
		return nil, fmt.Errorf("error marshaling chaos pod template: %w", err)
	}

	err = json.Unmarshal(data, &pod)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling chaos pod template: %w", err)
	}

	// customize pod template
	if pod.ObjectMeta.Labels == nil {
		pod.ObjectMeta.Labels = map[string]string{}
	}

	pod.ObjectMeta.GenerateName = fmt.Sprintf("chaos-%s-", instance.Name)
	pod.ObjectMeta.Namespace = instance.Namespace
	pod.ObjectMeta.Labels[chaostypes.TargetLabel] = targetName
	pod.ObjectMeta.Labels[chaostypes.DisruptionKindLabel] = string(kind)
	pod.ObjectMeta.Labels[chaostypes.DisruptionNameLabel] = instance.Name
	pod.Spec.NodeName = targetNodeName
	pod.Spec.Containers[0].Image = image
	pod.Spec.Containers[0].Args = args
	pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, corev1.EnvVar{
		Name: chaostypes.TargetPodHostIPEnv,
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "status.hostIP",
			},
		},
	})

	// add finalizer to the pod so it is not deleted before we can control its exit status
	controllerutil.AddFinalizer(&pod, chaosPodFinalizer)

	return &pod, nil
}

// handleMetricSinkError logs the given metric sink error if it is not nil
func (r *DisruptionReconciler) handleMetricSinkError(err error) {
	if err != nil {
		r.Log.Errorw("error sending a metric", "error", err)
	}
}

func (r *DisruptionReconciler) validateDisruptionSpec(instance *chaosv1beta1.Disruption) error {
	for _, kind := range chaostypes.DisruptionKinds {
		var validator chaosapi.DisruptionValidator

		// check for disruption kind
		switch kind {
		case chaostypes.DisruptionKindNodeFailure:
			validator = instance.Spec.NodeFailure
		case chaostypes.DisruptionKindNetworkDisruption:
			validator = instance.Spec.Network
		case chaostypes.DisruptionKindCPUPressure:
			validator = instance.Spec.CPUPressure
		case chaostypes.DisruptionKindDiskPressure:
			validator = instance.Spec.DiskPressure
		}

		// ensure that the underlying disruption spec is not nil
		if reflect.ValueOf(validator).IsNil() {
			continue
		}

		err := validator.Validate()
		if err != nil {
			r.Recorder.Event(instance, "Warning", "InvalidSpec", err.Error())
			return err
		}
	}

	return nil
}

// generateChaosPods generates a chaos pod for the given instance and disruption kind if set
func (r *DisruptionReconciler) generateChaosPods(instance *chaosv1beta1.Disruption, pods *[]*corev1.Pod, targetName string, targetNodeName string, containerID string) error {
	// generate chaos pods for each possible disruptions
	for _, kind := range chaostypes.DisruptionKinds {
		var generator chaosapi.DisruptionArgsGenerator

		// check for disruption kind
		switch kind {
		case chaostypes.DisruptionKindNodeFailure:
			generator = instance.Spec.NodeFailure
		case chaostypes.DisruptionKindNetworkDisruption:
			generator = instance.Spec.Network
		case chaostypes.DisruptionKindCPUPressure:
			generator = instance.Spec.CPUPressure
		case chaostypes.DisruptionKindDiskPressure:
			generator = instance.Spec.DiskPressure
		}

		// ensure that the underlying disruption spec is not nil
		if reflect.ValueOf(generator).IsNil() {
			continue
		}

		// default level to pod if not specified
		level := instance.Spec.Level
		if level == chaostypes.DisruptionLevelUnspecified {
			level = chaostypes.DisruptionLevelPod
		}

		// generate args for pod
		args := generator.GenerateArgs(level, containerID, r.MetricsSink.GetSinkName(), instance.Spec.DryRun)

		// generate pod
		pod, err := r.generatePod(instance, targetName, targetNodeName, args, kind)
		if err != nil {
			return err
		}

		// append pod to chaos pods
		*pods = append(*pods, pod)
	}

	return nil
}

// recordEventOnTarget records an event on the given target which can be either a pod or a node depending on the given disruption level
func (r *DisruptionReconciler) recordEventOnTarget(instance *chaosv1beta1.Disruption, target string, eventtype, reason, message string) {
	r.Log.Infow("registering an event on a target", "target", target, "instance", instance.Name, "namespace", instance.Namespace, "level", instance.Spec.Level, "eventtype", eventtype, "reason", reason, "message", message)

	var o runtime.Object

	switch instance.Spec.Level {
	case chaostypes.DisruptionLevelUnspecified, chaostypes.DisruptionLevelPod:
		p := &corev1.Pod{}

		if err := r.Get(context.Background(), types.NamespacedName{Namespace: instance.Namespace, Name: target}, p); err != nil {
			r.Log.Errorw("event failed to be registered on target", "error", err, "target", target)
		}

		o = p
	case chaostypes.DisruptionLevelNode:
		n := &corev1.Node{}

		if err := r.Get(context.Background(), types.NamespacedName{Name: target}, n); err != nil {
			r.Log.Errorw("event failed to be registered on target", "error", err, "target", target)
		}

		o = n
	}

	r.Recorder.Event(o, eventtype, reason, message)
}

// SetupWithManager setups the current reconciler with the given manager
func (r *DisruptionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chaosv1beta1.Disruption{}).
		Complete(r)
}

// WatchStuckOnRemoval lists disruptions every minutes and increment the "stuck on removal" metric
// for every disruptions stuck on removal
func (r *DisruptionReconciler) WatchStuckOnRemoval() {
	for {
		// wait for a minute
		<-time.After(time.Minute)

		l := chaosv1beta1.DisruptionList{}
		count := 0

		// list disruptions
		if err := r.Client.List(context.Background(), &l); err != nil {
			r.Log.Errorw("error listing disruptions", "error", err)
			continue
		}

		// check for stuck ones
		for _, d := range l.Items {
			if d.Status.IsStuckOnRemoval {
				count++

				if err := r.MetricsSink.MetricStuckOnRemoval([]string{"name:" + d.Name, "namespace:" + d.Namespace}); err != nil {
					r.Log.Errorw("error sending stuck_on_removal metric", "error", err)
				}
			}
		}

		// send count metric
		if err := r.MetricsSink.MetricStuckOnRemovalCount(float64(count)); err != nil {
			r.Log.Errorw("error sending stuck_on_removal_count metric", "error", err)
		}
	}
}
