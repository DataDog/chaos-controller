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
	"math/rand"
	"os"
	"reflect"
	"time"

	"github.com/go-logr/logr"
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
	if !instance.ObjectMeta.DeletionTimestamp.IsZero() {
		// if being deleted, call the finalizer
		if controllerutil.ContainsFinalizer(instance, disruptionFinalizer) {
			isCleaned, err := r.cleanDisruption(instance)
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

			controllerutil.RemoveFinalizer(instance, disruptionFinalizer)

			return ctrl.Result{}, r.Update(context.Background(), instance)
		}

		// stop the reconcile loop, the finalizing step has finished and the resource should be garbage collected
		return ctrl.Result{}, nil
	}

	// add the disruption finalizer
	controllerutil.AddFinalizer(instance, disruptionFinalizer)

	// compute spec hash to detect any changes in the spec and warn the user about it
	sameHashes, err := r.computeHash(instance)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error computing instance spec hash: %w", err)
	} else if !sameHashes {
		r.Log.Info("instance spec hash has changed meaning a change to the spec has been made, aborting")

		return ctrl.Result{}, nil
	}

	// retrieve targets from label selector
	if err := r.selectTargets(instance); err != nil {
		r.Log.Error(err, "error selecting targets", "instance", instance.Name, "namespace", instance.Namespace)

		return ctrl.Result{}, fmt.Errorf("error selecting targets: %w", err)
	}

	// start injections
	if err := r.startInjection(instance); err != nil {
		r.Log.Error(err, "error injecting the disruption", "instance", instance.Name, "namespace", instance.Namespace)

		return ctrl.Result{}, fmt.Errorf("error injecting the disruption: %w", err)
	}

	// update resource status injection flag
	// we reach this line only when every injection pods have been created with success
	r.handleMetricSinkError(r.MetricsSink.MetricInjectDuration(time.Since(instance.ObjectMeta.CreationTimestamp.Time), []string{"name:" + instance.Name, "namespace:" + instance.Namespace}))

	r.Log.Info("updating injection status flag", "instance", instance.Name, "namespace", instance.Namespace)
	instance.Status.IsInjected = true

	return ctrl.Result{}, r.Update(context.Background(), instance)
}

// startInjection creates non-existing chaos pod for the given disruption
func (r *DisruptionReconciler) startInjection(instance *chaosv1beta1.Disruption) error {
	var err error

	r.Log.Info("starting targets injection", "instance", instance.Name, "namespace", instance.Namespace, "targets", instance.Status.Targets)

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
			r.Log.Error(err, "error generating injection chaos pod for target, skipping it", "instance", instance.Name, "namespace", instance.Namespace, "target", target)

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
				r.Log.Info("creating chaos pod", "instance", instance.Name, "namespace", instance.Namespace, "target", target, "spec", chaosPod)

				if err = r.Create(context.Background(), chaosPod); err != nil {
					r.Recorder.Event(instance, "Warning", "Create failed", fmt.Sprintf("Injection pod for disruption \"%s\" failed to be created", instance.Name))
					r.handleMetricSinkError(r.MetricsSink.MetricPodsCreated(target, instance.Name, instance.Namespace, "inject", false))

					return fmt.Errorf("error creating chaos pod: %w", err)
				}

				// send metrics and events
				r.Recorder.Event(instance, "Normal", "Created", fmt.Sprintf("Created disruption injection pod for \"%s\"", instance.Name))
				r.recordEventOnTarget(instance, target, "Warning", "Disrupted", fmt.Sprintf("Pod %s from disruption %s targeted this resourcer for injection", chaosPod.Name, instance.Name))
				r.handleMetricSinkError(r.MetricsSink.MetricPodsCreated(target, instance.Name, instance.Namespace, "inject", true))
			} else {
				r.Log.Info("an injection pod is already existing for the selected target", "instance", instance.Name, "namespace", instance.Namespace, "target", target)
			}
		}
	}

	return nil
}

// computeHash computes the given instance spec hash and returns true if both hashes (the computed one and the stored one) are the same
// if it is not present in the instance status yet, the hash is stored and the function returns true
// if it is present, both hash are compared: if they are different, a modification has been made to the disruption and the function returns false
func (r *DisruptionReconciler) computeHash(instance *chaosv1beta1.Disruption) (bool, error) {
	// serialize instance spec to JSON and compute bytes hash
	specBytes, err := json.Marshal(instance.Spec)
	if err != nil {
		return false, fmt.Errorf("error computing instance spec hash: %w", err)
	}

	specHash := fmt.Sprintf("%x", md5.Sum(specBytes))

	// compare computed and stored hashes if present
	// register an event in the instance if hash are different
	if instance.Status.SpecHash != nil {
		if *instance.Status.SpecHash != specHash {
			r.Recorder.Event(instance, "Warning", "Mutated", "A mutation in the disruption spec has been detected. This resource is immutable and changes have no effect. Please delete and re-create the resource for changes to be effective.")

			return false, nil
		}
	}

	// store the computed hash
	r.Log.Info("storing resource spec hash to detect further changes in spec", "instance", instance.Name, "namespace", instance.Namespace)
	instance.Status.SpecHash = &specHash

	return true, r.Update(context.Background(), instance)
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
				r.Log.Info("target is not likely to be cleaned (either it does not exist anymore or it is not ready), the injector will TRY to clean it but will not take care about any failures", "instance", instance.Name, "namespace", instance.Namespace, "target", target)

				// by enabling this, we will remove the target associated chaos pods finalizers and delete them to trigger the cleanup phase
				// but the chaos pods status will not be checked
				ignoreCleanupStatus = true
			} else {
				r.Log.Error(err, err.Error())

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
				r.Log.Info("chaos pod completed, removing finalizer", "instance", instance.Name, "target", target, "chaosPod", chaosPod.Name)

				controllerutil.RemoveFinalizer(&chaosPod, chaosPodFinalizer)

				if err := r.Client.Update(context.Background(), &chaosPod); err != nil {
					r.Log.Error(err, "error removing chaos pod finalizer", "instance", instance.Name, "target", target, "chaosPod", chaosPod.Name)
				}
			}

			// if the chaos pod is running or pending, delete it to trigger the termination and the cleanup
			if chaosPod.Status.Phase == corev1.PodRunning || chaosPod.Status.Phase == corev1.PodPending {
				r.Log.Info("terminating chaos pod to trigger cleanup", "instance", instance.Name, "target", target, "chaosPod", chaosPod.Name)

				if err := r.Client.Delete(context.Background(), &chaosPod); err != nil {
					r.Log.Error(err, "error terminating chaos pod", "instance", instance.Name, "target", target, "chaosPod", chaosPod.Name)
				}
			}

			// if the chaos pod has failed, declare the disruption as stuck on removal so it can be investigated
			if chaosPod.Status.Phase == corev1.PodFailed && !ignoreCleanupStatus {
				r.Log.Info("instance seems stuck on removal for this target, please check manually", "instance", instance.Name, "target", target, "chaosPod", chaosPod.Name)
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

	r.Log.Info("selecting targets to inject disruption to", "instance", instance.Name, "namespace", instance.Namespace, "selector", instance.Spec.Selector.String())

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

	// if count is greater than the actual number of matching targets, cap it to avoid any issues
	if targetsCount > len(allTargets) {
		targetsCount = len(allTargets)
	}

	// randomly pick up targets from the found ones
	for i := 0; i < targetsCount; i++ {
		index := rand.Intn(len(allTargets))
		selectedTarget := allTargets[index]
		instance.Status.Targets = append(instance.Status.Targets, selectedTarget)
		allTargets[len(allTargets)-1], allTargets[index] = allTargets[index], allTargets[len(allTargets)-1]
		allTargets = allTargets[:len(allTargets)-1]
	}

	r.Log.Info("updating instance status with targets selected for injection", "instance", instance.Name, "namespace", instance.Namespace)

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
		r.Log.Error(err, "error sending a metric")
	}
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
		args := generator.GenerateArgs(level, containerID, r.MetricsSink.GetSinkName())

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
