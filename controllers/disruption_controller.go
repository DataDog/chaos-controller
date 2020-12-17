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
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
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
	finalizer = "finalizer.chaos.datadoghq.com"
)

// DisruptionReconciler reconciles a Disruption object
type DisruptionReconciler struct {
	client.Client
	Log             logr.Logger
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
	_ = context.Background()
	_ = r.Log.WithValues("disruption", req.NamespacedName)
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
	r.Log.Info("fetching disruption instance", "instance", req.Name, "namespace", req.Namespace)

	if err := r.Get(context.Background(), req.NamespacedName, instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// check wether the object is being deleted or not
	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		// if not being deleted, add a finalizer if not present yet
		controllerutil.AddFinalizer(instance, finalizer)
	} else {
		// if being deleted, call the finalizer
		if controllerutil.ContainsFinalizer(instance, finalizer) {
			r.Log.Info("creating cleanup pods", "instance", instance.Name, "namespace", instance.Namespace)

			// create cleanup pods
			isCleaned, err := r.cleanDisruptions(instance)
			if err != nil {
				return ctrl.Result{}, err
			}

			// if not cleaned yet, requeue and reconcile again in 5s-10s
			// the reason why we don't rely on the exponential backoff here is that it retries too fast at the beginning
			if !isCleaned {
				requeueAfter := time.Duration(rand.Intn(5)+5) * time.Second

				r.Log.Info(fmt.Sprintf("disruption has not been fully cleaned yet, re-queuing in %v", requeueAfter), "instance", instance.Name, "namespace", instance.Namespace)

				return ctrl.Result{
					Requeue:      true,
					RequeueAfter: requeueAfter,
				}, r.Update(context.Background(), instance)
			}

			// we reach this code when all the cleanup pods have succeeded
			// we can remove the finalizer and let the resource being garbage collected
			r.Log.Info("removing finalizer", "instance", instance.Name, "namespace", instance.Namespace)
			r.handleMetricSinkError(r.MetricsSink.MetricCleanupDuration(time.Since(instance.ObjectMeta.DeletionTimestamp.Time), []string{"name:" + instance.Name, "namespace:" + instance.Namespace}))

			controllerutil.RemoveFinalizer(instance, finalizer)

			return ctrl.Result{}, r.Update(context.Background(), instance)
		}

		// stop the reconcile loop, the finalizing step has finished and the resource should be garbage collected
		return ctrl.Result{}, nil
	}

	// compute spec hash to detect any changes in the spec and warn the user about it
	specBytes, err := json.Marshal(instance.Spec)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error compute instance spec hash: %w", err)
	}

	specHash := fmt.Sprintf("%x", md5.Sum(specBytes))

	if instance.Status.SpecHash != nil {
		if *instance.Status.SpecHash != specHash {
			r.Recorder.Event(instance, "Warning", "Mutated", "A mutation in the disruption spec has been detected. This resource is immutable and changes have no effect. Please delete and re-create the resource for changes to be effective.")

			return ctrl.Result{}, nil
		}
	} else {
		r.Log.Info("computing resource spec hash to detect further changes in spec", "instance", instance.Name, "namespace", instance.Namespace)
		instance.Status.SpecHash = &specHash

		return ctrl.Result{}, r.Update(context.Background(), instance)
	}

	// skip the injection if already done
	if instance.Status.IsInjected {
		return ctrl.Result{}, nil
	}

	// retrieve targets from label selector
	targets, err := r.selectTargets(instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	// select pods eligible for an injection and add them
	// to the instance status for the next loop
	if len(instance.Status.Targets) == 0 {
		// select pods
		r.Log.Info("selecting targets to inject disruption to", "instance", instance.Name, "namespace", instance.Namespace, "selector", instance.Spec.Selector.String())

		// stop here if no pods can be targeted
		if len(targets) == 0 {
			r.Log.Info("the given label selector returned no targets", "instance", instance.Name, "selector", instance.Spec.Selector.String())
			r.Recorder.Event(instance, "Warning", "NoTarget", "The given label selector did not return any targets. Please ensure that both the selector and the count are correct (should be either a percentage or an integer greater than 0).")

			return ctrl.Result{
				Requeue: false,
			}, nil
		}

		// update instance status
		r.Log.Info("updating instance status with targets selected for injection", "instance", instance.Name, "namespace", instance.Namespace)
		instance.Status.Targets = append(instance.Status.Targets, targets...)

		return ctrl.Result{}, r.Update(context.Background(), instance)
	}

	// start injections
	r.Log.Info("starting targets injection", "instance", instance.Name, "namespace", instance.Namespace, "targets", instance.Status.Targets)

	skippedTargets := []string{}

	for _, target := range instance.Status.Targets {
		var targetNodeName, containerID string

		chaosPods := []*corev1.Pod{}

		// retrieve target
		switch instance.Spec.Level {
		case chaostypes.DisruptionLevelUnspecified, chaostypes.DisruptionLevelPod:
			pod := corev1.Pod{}

			if err := r.Get(context.Background(), types.NamespacedName{Namespace: instance.Namespace, Name: target}, &pod); err != nil {
				return ctrl.Result{}, err
			}

			targetNodeName = pod.Spec.NodeName

			// get ID of targeted container or the first container
			containerID, err = getContainerID(&pod, instance.Spec.Container)
			if err != nil {
				return ctrl.Result{}, err
			}
		case chaostypes.DisruptionLevelNode:
			targetNodeName = target
		}

		// generate injection pods specs
		if err := r.generateChaosPods(instance, &chaosPods, target, targetNodeName, chaostypes.PodModeInject, containerID); err != nil {
			r.Log.Error(err, "error generating injection chaos pod for target, skipping it", "instance", instance.Name, "namespace", instance.Namespace, "target", target)
			skippedTargets = append(skippedTargets, target)

			continue
		}

		if len(chaosPods) == 0 {
			r.Recorder.Event(instance, "Warning", "Empty Disruption", fmt.Sprintf("No disruption recognized for \"%s\" therefore no disruption applied.", instance.Name))
		}

		// create injection pods
		for _, chaosPod := range chaosPods {
			// link instance resource and injection pod for garbage collection
			if err := controllerutil.SetControllerReference(instance, chaosPod, r.Scheme); err != nil {
				return ctrl.Result{}, err
			}

			// check if an injection pod already exists for the given (instance, namespace, disruption kind) tuple
			found, err := r.getOwnedPods(instance, chaosPod.Labels)
			if err != nil {
				return ctrl.Result{}, err
			}

			if len(found) == 0 {
				r.Log.Info("creating chaos pod", "instance", instance.Name, "namespace", instance.Namespace, "target", target, "spec", chaosPod)

				if err = r.Create(context.Background(), chaosPod); err != nil {
					r.Recorder.Event(instance, "Warning", "Create failed", fmt.Sprintf("Injection pod for disruption \"%s\" failed to be created", instance.Name))
					r.handleMetricSinkError(r.MetricsSink.MetricPodsCreated(target, instance.Name, instance.Namespace, "inject", false))

					return ctrl.Result{}, err
				}

				// send metrics and events
				r.Recorder.Event(instance, "Normal", "Created", fmt.Sprintf("Created disruption injection pod for \"%s\"", instance.Name))
				r.recordEventOnTarget(instance, target, "Warning", "Disrupted", fmt.Sprintf("Pod %s from disruption %s targeted this resourcer for injection", chaosPod.Name, instance.Name))
				r.handleMetricSinkError(r.MetricsSink.MetricPodsCreated(target, instance.Name, instance.Namespace, "inject", true))
			} else {
				r.Log.Info("an injection pod is already existing for the selected pod", "instance", instance.Name, "namespace", instance.Namespace, "target", target)
			}
		}
	}

	// remove skipped targets from the list so we don't have to clean them up later
	r.removeSkippedTargets(instance, skippedTargets)

	// update resource status injection flag
	// we reach this line only when every injection pods have been created with success
	r.handleMetricSinkError(r.MetricsSink.MetricInjectDuration(time.Since(instance.ObjectMeta.CreationTimestamp.Time), []string{"name:" + instance.Name, "namespace:" + instance.Namespace}))

	r.Log.Info("updating injection status flag", "instance", instance.Name, "namespace", instance.Namespace)
	instance.Status.IsInjected = true

	return ctrl.Result{}, r.Update(context.Background(), instance)
}

func (r *DisruptionReconciler) removeSkippedTargets(instance *chaosv1beta1.Disruption, skippedTargets []string) {
	remainingTargets := []string{}

	for _, target := range instance.Status.Targets {
		skipped := false

		for _, skippedTarget := range skippedTargets {
			if target == skippedTarget {
				skipped = true
				break
			}
		}

		if !skipped {
			remainingTargets = append(remainingTargets, target)
		}
	}

	instance.Status.Targets = remainingTargets
}

// cleanDisruptions creates cleanup pods for a given disruption instance
// it returns true once all disruptions are cleaned
func (r *DisruptionReconciler) cleanDisruptions(instance *chaosv1beta1.Disruption) (bool, error) {
	var containerID, targetNodeName string

	chaosPods := []*v1.Pod{}
	isFullyCleaned := true

	// create one cleanup pod for pod to cleanup
	for _, target := range instance.Status.Targets {
		// check target readiness for cleanup
		err := r.TargetSelector.TargetIsHealthy(target, r.Client, instance)

		if err != nil {
			if errors.IsNotFound(err) {
				r.Log.Info("cleanup: target no longer exists (skip)", "instance", instance.Name, "namespace", instance.Namespace, "name", target)
			} else if err.Error() == "Pod is not Running" || err.Error() == "Node is not Ready" {
				r.Log.Info("cleanup: target not healthy (skip)", "instance", instance.Name, "namespace", instance.Namespace, "name", target)
			}
			r.Log.Error(err, err.Error())
			continue
		}

		switch instance.Spec.Level {
		case chaostypes.DisruptionLevelUnspecified, chaostypes.DisruptionLevelPod:
			pod := corev1.Pod{}

			if err := r.Get(context.Background(), types.NamespacedName{Namespace: instance.Namespace, Name: target}, &pod); err != nil {
				return false, err
			}

			targetNodeName = pod.Spec.NodeName

			// get ID of targeted container or the first container
			containerID, err = getContainerID(&pod, instance.Spec.Container)
			if err != nil {
				return false, err
			}
		case chaostypes.DisruptionLevelNode:
			targetNodeName = target
		}

		// generate cleanup pods specs
		if err := r.generateChaosPods(instance, &chaosPods, target, targetNodeName, chaostypes.PodModeClean, containerID); err != nil {
			return false, err
		}

		// create cleanup pods
		for _, chaosPod := range chaosPods {
			isCleaned := false

			r.Log.Info("checking for existing cleanup pods", "instance", instance.Name, "namespace", instance.Namespace)

			// get already existing cleanup pods for the specific disruption and target
			existingCleanupPods, err := r.getOwnedPods(instance, chaosPod.Labels)
			if err != nil {
				return false, err
			}

			// skip if there is at least one pod not erroring (can be running, completed, etc.)
			// consider the disruption cleaned for this target if at least one pod has succeeded
			// limit number of cleanup pods per disruption to 5, after this we expect
			// the users to manually check what's happening
			if len(existingCleanupPods) > 0 {
				skip := false

				// check for a succeeded pod and for any non-erroring pods
				for _, existingChaosPod := range existingCleanupPods {
					if existingChaosPod.Status.Phase == corev1.PodSucceeded {
						isCleaned = true
					}

					if existingChaosPod.Status.Phase != corev1.PodFailed {
						skip = true
					}
				}

				// if no pods have succeeded yet, do not consider the disruptions as cleaned
				// if the disruption could not be cleaned up after 5 pods, skip and wait for
				// a manual verification/clean up
				if !isCleaned {
					isFullyCleaned = false

					if len(existingCleanupPods) >= 5 {
						r.Log.Info("maximum cleanup pods count reached (5), please debug manually", "instance", instance.Name, "namespace", instance.Namespace)
						r.Recorder.Event(instance, "Warning", "Undisruption failed", "Disruption could not being cleaned up, please debug manually")
						r.recordEventOnTarget(instance, target, "Warning", "Undisruption failed", fmt.Sprintf("Disruption %s could not being cleaned up, please debug manually", instance.Name))

						instance.Status.IsStuckOnRemoval = true

						continue
					}
				}

				// do not create a new cleanup pod if at least one pod is not erroring
				if skip {
					continue
				}
			} else {
				isFullyCleaned = false
			}

			// link cleanup pod to instance for garbage collection
			if err := controllerutil.SetControllerReference(instance, chaosPod, r.Scheme); err != nil {
				return false, err
			}

			r.Log.Info("creating chaos cleanup pod", "instance", instance.Name, "namespace", chaosPod.Namespace, "name", chaosPod.Name, "target", target)

			if err := r.Create(context.Background(), chaosPod); err != nil {
				r.Recorder.Event(instance, "Warning", "Create failed", fmt.Sprintf("Cleanup pod for disruption \"%s\" failed to be created", instance.Name))
				r.recordEventOnTarget(instance, target, "Warning", "Undisrupted", fmt.Sprintf("Disruption %s failed to be cleaned up by pod %s", instance.Name, chaosPod.Name))
				r.handleMetricSinkError(r.MetricsSink.MetricPodsCreated(target, instance.Name, instance.Namespace, "cleanup", false))

				return false, err
			}

			r.Recorder.Event(instance, "Normal", "Created", fmt.Sprintf("Created cleanup pod for disruption \"%s\"", instance.Name))
			r.recordEventOnTarget(instance, target, "Normal", "Undisrupted", fmt.Sprintf("Disruption %s is being cleaned up by pod %s", instance.Name, chaosPod.Name))
			r.handleMetricSinkError(r.MetricsSink.MetricPodsCreated(target, instance.Name, instance.Namespace, "cleanup", true))
		}
	}

	return isFullyCleaned, nil
}

// selectTargets will select min(count, all matching targets) random targets (pods or nodes depending on the disruption level)
// from the targets matching the instance label selector
// targets will only be selected once per instance
// the chosen targets names will be reflected in the intance status
// subsequent calls to this function will always return the same targets as the first call
func (r *DisruptionReconciler) selectTargets(instance *chaosv1beta1.Disruption) ([]string, error) {
	var (
		err                         error
		allTargets, selectedTargets []string
	)

	// select either pods or nodes depending on the disruption level
	switch instance.Spec.Level {
	case chaostypes.DisruptionLevelUnspecified, chaostypes.DisruptionLevelPod:
		pods, err := r.TargetSelector.GetMatchingPods(r.Client, instance)
		if err != nil {
			return nil, fmt.Errorf("can't get pods matching the given label selector: %w", err)
		}

		for _, pod := range pods.Items {
			allTargets = append(allTargets, pod.Name)
		}
	case chaostypes.DisruptionLevelNode:
		nodes, err := r.TargetSelector.GetMatchingNodes(r.Client, instance)
		if err != nil {
			return nil, fmt.Errorf("can't get pods matching the given label selector: %w", err)
		}

		for _, node := range nodes.Items {
			allTargets = append(allTargets, node.Name)
		}
	}

	// instance.Spec.Count is a string that either represents a percentage or a value, we do the translation here
	targetsCount, err := getScaledValueFromIntOrPercent(instance.Spec.Count, len(allTargets), true)
	if err != nil {
		targetsCount = instance.Spec.Count.IntValue()
	}

	// computed count should not be 0 unless the given count was not expected
	if targetsCount == 0 {
		return nil, fmt.Errorf("parsing error, either incorrectly formatted percentage or incorrectly formatted integer: %s\n%w", instance.Spec.Count.String(), err)
	}

	// if count is greater than the actual number of matching targets, return all of them
	if targetsCount >= len(allTargets) {
		return allTargets, nil
	}

	// if we had already selected pods for the instance, only return the already-selected ones
	if len(instance.Status.Targets) > 0 {
		for _, target := range allTargets {
			if containsString(instance.Status.Targets, target) {
				selectedTargets = append(selectedTargets, target)
			}
		}

		return selectedTargets, nil
	}

	// otherwise, randomly select targets within the list of targets
	// and take care to remove it from the list once done
	for i := 0; i < targetsCount; i++ {
		index := rand.Intn(len(allTargets))
		selectedTarget := allTargets[index]
		selectedTargets = append(selectedTargets, selectedTarget)
		allTargets[len(allTargets)-1], allTargets[index] = allTargets[index], allTargets[len(allTargets)-1]
		allTargets = allTargets[:len(allTargets)-1]
	}

	return selectedTargets, nil
}

func (r *DisruptionReconciler) getOwnedPods(instance *chaosv1beta1.Disruption, selector labels.Set) ([]corev1.Pod, error) {
	ownedPods := make([]corev1.Pod, 0)
	pods := &corev1.PodList{}

	listOptions := &client.ListOptions{
		Namespace:     instance.Namespace,
		LabelSelector: labels.SelectorFromSet(selector),
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

// getContainerID gets the ID of the targeted container or of the first container ID found in a Pod
func getContainerID(pod *corev1.Pod, target string) (string, error) {
	if len(pod.Status.ContainerStatuses) < 1 {
		return "", fmt.Errorf("missing container ids for pod '%s'", pod.Name)
	}

	for _, container := range pod.Status.ContainerStatuses {
		if container.Name == target {
			return container.ContainerID, nil
		}
	}

	return pod.Status.ContainerStatuses[0].ContainerID, nil
}

// generatePod generates a pod from a generic pod template in the same namespace
// and on the same node as the given pod
func (r *DisruptionReconciler) generatePod(instance *chaosv1beta1.Disruption, targetName string, targetNodeName string, args []string, mode chaostypes.PodMode, kind chaostypes.DisruptionKind) (*corev1.Pod, error) {
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

	pod.ObjectMeta.GenerateName = fmt.Sprintf("chaos-%s-%s-", instance.Name, mode)
	pod.ObjectMeta.Namespace = instance.Namespace
	pod.ObjectMeta.Labels[chaostypes.PodModeLabel] = string(mode)
	pod.ObjectMeta.Labels[chaostypes.TargetPodLabel] = targetName
	pod.ObjectMeta.Labels[chaostypes.DisruptionKindLabel] = string(kind)
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

	return &pod, nil
}

// handleMetricSinkError logs the given metric sink error if it is not nil
func (r *DisruptionReconciler) handleMetricSinkError(err error) {
	if err != nil {
		r.Log.Error(err, "error sending a metric")
	}
}

// generateChaosPods generates a chaos pod for the given instance and disruption kind if set
func (r *DisruptionReconciler) generateChaosPods(instance *chaosv1beta1.Disruption, pods *[]*corev1.Pod, targetName string, targetNodeName string, mode chaostypes.PodMode, containerID string) error {
	var kinds []chaostypes.DisruptionKind

	// choose chaos pods to create depending on the mode (injection or cleanup)
	// since some disruptions don't need any cleanup pods to be created
	if mode == chaostypes.PodModeInject {
		kinds = chaostypes.DisruptionKindsInject
	} else {
		kinds = chaostypes.DisruptionKindsClean
	}

	// generate chaos pods for each found kind depending on the mode
	// and on the disruption spec
	for _, kind := range kinds {
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
		args := generator.GenerateArgs(mode, level, containerID, r.MetricsSink.GetSinkName())

		// generate pod
		pod, err := r.generatePod(instance, targetName, targetNodeName, args, mode, kind)
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
	r.Log.Info("registering an event on a target", "target", target, "instance", instance.Name, "namespace", instance.Namespace, "level", instance.Spec.Level, "eventtype", eventtype, "reason", reason, "message", message)

	var o runtime.Object

	switch instance.Spec.Level {
	case chaostypes.DisruptionLevelUnspecified, chaostypes.DisruptionLevelPod:
		p := &corev1.Pod{}

		if err := r.Get(context.Background(), types.NamespacedName{Namespace: instance.Namespace, Name: target}, p); err != nil {
			r.Log.Error(err, "event failed to be registered on target", "target", target)
		}

		o = p
	case chaostypes.DisruptionLevelNode:
		n := &corev1.Node{}

		if err := r.Get(context.Background(), types.NamespacedName{Name: target}, n); err != nil {
			r.Log.Error(err, "event failed to be registered on target", "target", target)
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
			r.Log.Error(err, "error listing disruptions")
			continue
		}

		// check for stuck ones
		for _, d := range l.Items {
			if d.Status.IsStuckOnRemoval {
				count++

				if err := r.MetricsSink.MetricStuckOnRemoval([]string{"name:" + d.Name, "namespace:" + d.Namespace}); err != nil {
					r.Log.Error(err, "error sending stuck_on_removal metric")
				}
			}
		}

		// send count metric
		if err := r.MetricsSink.MetricStuckOnRemovalCount(float64(count)); err != nil {
			r.Log.Error(err, "error sending stuck_on_removal_count metric")
		}
	}
}

// This method returns a scaled value from an IntOrString type. If the IntOrString
// is a percentage string value it's treated as a percentage and scaled appropriately
// in accordance to the total, if it's an int value it's treated as a a simple value and
// if it is a string value which is either non-numeric or numeric but lacking a trailing '%' it returns an error.
func getScaledValueFromIntOrPercent(intOrPercent *intstr.IntOrString, total int, roundUp bool) (int, error) {
	if intOrPercent == nil {
		return 0, errors.NewBadRequest("nil value for IntOrString")
	}

	value, isPercent, err := getIntOrPercentValueSafely(intOrPercent)

	if err != nil {
		return 0, fmt.Errorf("invalid value for IntOrString: %v", err)
	}

	if isPercent {
		if roundUp {
			value = int(math.Ceil(float64(value) * (float64(total)) / 100))
		} else {
			value = int(math.Floor(float64(value) * (float64(total)) / 100))
		}
	}

	return value, nil
}

func getIntOrPercentValueSafely(intOrStr *intstr.IntOrString) (int, bool, error) {
	switch intOrStr.Type {
	case intstr.Int:
		return intOrStr.IntValue(), false, nil
	case intstr.String:
		isPercent := false
		s := intOrStr.StrVal

		if strings.HasSuffix(s, "%") {
			isPercent = true
			s = strings.TrimSuffix(intOrStr.StrVal, "%")
		} else {
			return 0, false, fmt.Errorf("invalid type: string is not a percentage")
		}

		v, err := strconv.Atoi(s)

		if err != nil {
			return 0, false, fmt.Errorf("invalid value %q: %v", intOrStr.StrVal, err)
		}

		return v, isPercent, nil
	}

	return 0, false, fmt.Errorf("invalid type: neither int nor percentage")
}

// containsString returns true if the given slice contains the given string,
// or returns false otherwise
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}

	return false
}
