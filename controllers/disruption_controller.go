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
	"github.com/DataDog/chaos-controller/helpers"
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
}

// +kubebuilder:rbac:groups=chaos.datadoghq.com,resources=disruptions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=chaos.datadoghq.com,resources=disruptions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch

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

	// select pods eligible for an injection and add them
	// to the instance status for the next loop
	if len(instance.Status.TargetPods) == 0 {
		// select pods
		r.Log.Info("selecting pods to inject disruption to", "instance", instance.Name, "namespace", instance.Namespace)

		pods, err := r.selectPodsForInjection(instance)
		if err != nil {
			return ctrl.Result{}, err
		}

		// stop here if no pods can be targeted
		if len(pods.Items) == 0 {
			r.Log.Info("the given label selector returned no pods", "instance", instance.Name, "selector", instance.Spec.Selector.String())
			r.Recorder.Event(instance, "Warning", "NoTarget", "The given label selector did not target any pods")

			return ctrl.Result{
				Requeue: false,
			}, nil
		}

		// update instance status
		r.Log.Info("updating instance status with pods selected for injection", "instance", instance.Name, "namespace", instance.Namespace)

		for _, pod := range pods.Items {
			instance.Status.TargetPods = append(instance.Status.TargetPods, pod.Name)
		}

		return ctrl.Result{}, r.Update(context.Background(), instance)
	}

	// start injections
	r.Log.Info("starting pods injection", "instance", instance.Name, "namespace", instance.Namespace, "targetPods", instance.Status.TargetPods)

	for _, targetPodName := range instance.Status.TargetPods {
		chaosPods := []*corev1.Pod{}
		targetPod := corev1.Pod{}

		// retrieve target pod resource
		if err := r.Get(context.Background(), types.NamespacedName{Namespace: instance.Namespace, Name: targetPodName}, &targetPod); err != nil {
			return ctrl.Result{}, err
		}

		// get ID of first container
		containerID, err := getContainerID(&targetPod)
		if err != nil {
			return ctrl.Result{}, err
		}

		// generate injection pods specs
		if err := r.generateChaosPod(instance, &chaosPods, targetPod, chaostypes.PodModeInject, containerID, chaostypes.DisruptionKindNetworkDisruption); err != nil {
			return ctrl.Result{}, err
		}

		if err := r.generateChaosPod(instance, &chaosPods, targetPod, chaostypes.PodModeInject, containerID, chaostypes.DisruptionKindNodeFailure); err != nil {
			return ctrl.Result{}, err
		}

		if err := r.generateChaosPod(instance, &chaosPods, targetPod, chaostypes.PodModeInject, containerID, chaostypes.DisruptionKindCPUPressure); err != nil {
			return ctrl.Result{}, err
		}

		if err := r.generateChaosPod(instance, &chaosPods, targetPod, chaostypes.PodModeInject, containerID, chaostypes.DisruptionKindDiskPressure); err != nil {
			return ctrl.Result{}, err
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
				r.Log.Info("creating chaos pod", "instance", instance.Name, "namespace", instance.Namespace, "targetPod", targetPod.Name, "spec", chaosPod)

				if err = r.Create(context.Background(), chaosPod); err != nil {
					r.Recorder.Event(instance, "Warning", "Create failed", fmt.Sprintf("Injection pod for disruption \"%s\" failed to be created", instance.Name))
					r.handleMetricSinkError(r.MetricsSink.MetricPodsCreated(targetPod.Name, instance.Name, instance.Namespace, "inject", false))

					return ctrl.Result{}, err
				}

				// send metrics and events
				r.Recorder.Event(instance, "Normal", "Created", fmt.Sprintf("Created disruption injection pod for \"%s\"", instance.Name))
				r.Recorder.Event(&targetPod, "Warning", "Disrupted", fmt.Sprintf("Pod %s from disruption %s targeted this pod for injection", chaosPod.Name, instance.Name))
				r.handleMetricSinkError(r.MetricsSink.MetricPodsCreated(targetPod.Name, instance.Name, instance.Namespace, "inject", true))
			} else {
				r.Log.Info("an injection pod is already existing for the selected pod", "instance", instance.Name, "namespace", instance.Namespace, "targetPod", targetPod.Name)
			}
		}
	}

	// update resource status injection flag
	// we reach this line only when every injection pods have been created with success
	r.handleMetricSinkError(r.MetricsSink.MetricInjectDuration(time.Since(instance.ObjectMeta.CreationTimestamp.Time), []string{"name:" + instance.Name, "namespace:" + instance.Namespace}))

	r.Log.Info("updating injection status flag", "instance", instance.Name, "namespace", instance.Namespace)
	instance.Status.IsInjected = true

	return ctrl.Result{}, r.Update(context.Background(), instance)
}

// getPodsToCleanup returns the still-existing pods that were targeted by the disruption, according to the pod names in the instance status
func (r *DisruptionReconciler) getPodsToCleanup(instance *chaosv1beta1.Disruption) ([]*corev1.Pod, error) {
	podsToCleanup := make([]*corev1.Pod, 0, len(instance.Status.TargetPods))

	// check if each pod still exists; skip if it doesn't
	for _, podName := range instance.Status.TargetPods {
		// get the targeted pods names from the status
		podKey := types.NamespacedName{Name: podName, Namespace: instance.Namespace}
		p := &corev1.Pod{}
		err := r.Get(context.Background(), podKey, p)

		// skip if the pod doesn't exist anymore
		if errors.IsNotFound(err) {
			r.Log.Info("cleanup: pod no longer exists", "instance", instance.Name, "namespace", instance.Namespace, "name", podName)
			continue
		} else if err != nil {
			return nil, err
		}

		podsToCleanup = append(podsToCleanup, p)
	}

	return podsToCleanup, nil
}

// cleanDisruptions creates cleanup pods for a given disruption instance
// it returns true once all disruptions are cleaned
func (r *DisruptionReconciler) cleanDisruptions(instance *chaosv1beta1.Disruption) (bool, error) {
	isFullyCleaned := true

	// retrieve pods to cleanup
	podsToCleanup, err := r.getPodsToCleanup(instance)
	if err != nil {
		return false, err
	}

	// create one cleanup pod for pod to cleanup
	for _, p := range podsToCleanup {
		chaosPods := []*v1.Pod{}

		// get ID of first container
		containerID, err := getContainerID(p)
		if err != nil {
			return false, err
		}

		// generate cleanup pods specs
		if err := r.generateChaosPod(instance, &chaosPods, *p, chaostypes.PodModeClean, containerID, chaostypes.DisruptionKindNetworkDisruption); err != nil {
			return false, err
		}

		if err := r.generateChaosPod(instance, &chaosPods, *p, chaostypes.PodModeClean, containerID, chaostypes.DisruptionKindDiskPressure); err != nil {
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
						r.Recorder.Event(p, "Warning", "Undisruption failed", fmt.Sprintf("Disruption %s could not being cleaned up, please debug manually", instance.Name))

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

			r.Log.Info("creating chaos cleanup chaosPod", "instance", instance.Name, "namespace", chaosPod.Namespace, "name", chaosPod.Name, "containerid", containerID)

			if err := r.Create(context.Background(), chaosPod); err != nil {
				r.Recorder.Event(instance, "Warning", "Create failed", fmt.Sprintf("Cleanup pod for disruption \"%s\" failed to be created", instance.Name))
				r.Recorder.Event(p, "Warning", "Undisrupted", fmt.Sprintf("Disruption %s failed to be cleaned up by pod %s", instance.Name, chaosPod.Name))
				r.handleMetricSinkError(r.MetricsSink.MetricPodsCreated(p.ObjectMeta.Name, instance.Name, instance.Namespace, "cleanup", false))

				return false, err
			}

			r.Recorder.Event(instance, "Normal", "Created", fmt.Sprintf("Created cleanup pod for disruption \"%s\"", instance.Name))
			r.Recorder.Event(p, "Normal", "Undisrupted", fmt.Sprintf("Disruption %s is being cleaned up by pod %s", instance.Name, chaosPod.Name))
			r.handleMetricSinkError(r.MetricsSink.MetricPodsCreated(p.ObjectMeta.Name, instance.Name, instance.Namespace, "cleanup", true))
		}
	}

	return isFullyCleaned, nil
}

// selectPodsForInjection will select min(count, all matching pods) random pods from the pods matching the instance label selector
// target pods will only be selected once per instance
// the chosen pods names will be reflected in the intance status
// subsequent calls to this function will always return the same pods as the first call
func (r *DisruptionReconciler) selectPodsForInjection(instance *chaosv1beta1.Disruption) (*corev1.PodList, error) {
	selectedPods := []corev1.Pod{}

	rand.Seed(time.Now().UnixNano())

	// get pods matching the instance label selector
	allPods, err := helpers.GetMatchingPods(r.Client, instance.Namespace, instance.Spec.Selector)
	if err != nil {
		return nil, fmt.Errorf("can't get pods matching the given label selector: %w", err)
	}

	// instance.Spec.Count is a string that either represents a percentage or a value, we do the translation here
	count := 0

	count, err = getScaledValueFromIntOrPercent(instance.Spec.Count, len(allPods.Items), true)
	if err != nil {
		count = instance.Spec.Count.IntValue()
	}

	if count == 0 {
		return nil, fmt.Errorf("parsing error, either incorrectly formatted percentage or incorrectly formatted integer: %s\n%w", instance.Spec.Count.String(), err)
	}
	// if count has not been specified or is greater than the actual number of matching pods,
	// return all pods matching the label selector
	if count >= len(allPods.Items) {
		return allPods, nil
	}

	// if we had already selected pods for the instance, only return the already-selected ones
	if len(instance.Status.TargetPods) > 0 {
		for _, pod := range allPods.Items {
			if helpers.ContainsString(instance.Status.TargetPods, pod.Name) {
				selectedPods = append(selectedPods, pod)
			}
		}

		return &corev1.PodList{Items: selectedPods}, nil
	}

	// otherwise, randomly select pods
	for i := 0; i < count; i++ {
		// select and add a random pod
		index := rand.Intn(len(allPods.Items))
		chosenPod := allPods.Items[index]
		selectedPods = append(selectedPods, chosenPod)

		r.Log.Info("Selected random pod", "name", chosenPod.Name, "namespace", chosenPod.Namespace, "disruption", instance.Name, "count", count)

		// remove the chosen pod from list of pods from which to select
		allPods.Items[len(allPods.Items)-1], allPods.Items[index] = allPods.Items[index], allPods.Items[len(allPods.Items)-1]
		allPods.Items = allPods.Items[:len(allPods.Items)-1]
	}

	return &corev1.PodList{Items: selectedPods}, nil
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

// getContainerID gets the ID of the first container ID found in a Pod
func getContainerID(pod *corev1.Pod) (string, error) {
	if len(pod.Status.ContainerStatuses) < 1 {
		return "", fmt.Errorf("missing container ids for pod '%s'", pod.Name)
	}

	return pod.Status.ContainerStatuses[0].ContainerID, nil
}

// generatePod generates a pod from a generic pod template in the same namespace
// and on the same node as the given pod
func (r *DisruptionReconciler) generatePod(instance *chaosv1beta1.Disruption, target corev1.Pod, args []string, mode chaostypes.PodMode, kind chaostypes.DisruptionKind) (*corev1.Pod, error) {
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
	pod.ObjectMeta.Namespace = target.Namespace
	pod.ObjectMeta.Labels[chaostypes.PodModeLabel] = string(mode)
	pod.ObjectMeta.Labels[chaostypes.TargetPodLabel] = target.Name
	pod.ObjectMeta.Labels[chaostypes.DisruptionKindLabel] = string(kind)
	pod.Spec.NodeName = target.Spec.NodeName
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

// generateChaosPod generates a chaos pod for the given instance and disruption kind if set
func (r *DisruptionReconciler) generateChaosPod(instance *chaosv1beta1.Disruption, pods *[]*corev1.Pod, target corev1.Pod, mode chaostypes.PodMode, containerID string, kind chaostypes.DisruptionKind) error {
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
		return nil
	}

	// default level to pod if not specified
	level := instance.Spec.Level
	if level == chaostypes.DisruptionLevelUnspecified {
		level = chaostypes.DisruptionLevelPod
	}

	// generate args for pod
	args := generator.GenerateArgs(mode, instance.UID, level, containerID, r.MetricsSink.GetSinkName())

	// generate pod
	pod, err := r.generatePod(instance, target, args, mode, kind)
	if err != nil {
		return err
	}

	// append pod to chaos pods
	*pods = append(*pods, pod)

	return nil
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
