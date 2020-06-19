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
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

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
		if !helpers.ContainsString(instance.ObjectMeta.Finalizers, finalizer) {
			r.Log.Info("adding finalizer", "instance", instance.Name, "namespace", instance.Namespace)
			instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, finalizer)

			return ctrl.Result{}, r.Update(context.Background(), instance)
		}
	} else {
		// if being deleted, call the finalizer
		if helpers.ContainsString(instance.ObjectMeta.Finalizers, finalizer) {
			// if the finalizing stage hasn't been triggered yet, start the cleaning
			if !instance.Status.IsFinalizing {
				if err := r.cleanFailures(instance); err != nil {
					return ctrl.Result{}, err
				}

				// set the finalizing flag
				instance.Status.IsFinalizing = true
				r.Log.Info("updating finalizing flag", "instance", instance.Name, "namespace", instance.Namespace)
				return ctrl.Result{}, r.Update(context.Background(), instance)
			}

			// retrieve cleanup pods to check their states
			cleanupPods, err := r.getChaosPods(instance, chaostypes.PodModeClean)
			if err != nil {
				return ctrl.Result{}, err
			}

			r.Log.Info("checking status of cleanup pods before deleting failure", "numcleanuppods", len(cleanupPods), "instance", instance.Name, "namespace", instance.Namespace)

			// check if cleanup pods have succeeded, requeue until they have
			for _, cleanupPod := range cleanupPods {
				if cleanupPod.Status.Phase != corev1.PodSucceeded {
					r.Log.Info("cleanup pod has not completed, retrying failure deletion", "instance", instance.Name, "namespace", instance.Namespace, "cleanuppod", cleanupPod.Name, "phase", cleanupPod.Status.Phase)
					return ctrl.Result{
						Requeue: true,
					}, nil
				}
			}

			// we reach this code when all the cleanup pods have succeeded
			// we can remove the finalizer and let the resource being garbage collected
			r.Log.Info("removing finalizer", "instance", instance.Name, "namespace", instance.Namespace)
			r.handleMetricSinkError(r.MetricsSink.MetricCleanupDuration(time.Since(instance.ObjectMeta.DeletionTimestamp.Time), []string{"name:" + instance.Name, "namespace:" + instance.Namespace}))
			instance.ObjectMeta.Finalizers = helpers.RemoveString(instance.ObjectMeta.Finalizers, finalizer)
			return ctrl.Result{}, r.Update(context.Background(), instance)
		}

		// stop the reconcile loop, the finalizing step has finished and the resource should be garbage collected
		return ctrl.Result{}, nil
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

		// retrieve pod resource
		if err := r.Get(context.Background(), types.NamespacedName{Namespace: instance.Namespace, Name: targetPodName}, &targetPod); err != nil {
			return ctrl.Result{}, err
		}

		// get ID of first container
		containerID, err := getContainerID(&targetPod)
		if err != nil {
			return ctrl.Result{}, err
		}

		// generate injection pods specs
		if instance.Spec.NetworkFailure != nil {
			args := instance.Spec.NetworkFailure.GenerateArgs(chaostypes.PodModeInject, instance.UID, containerID, r.MetricsSink.GetSinkName())

			chaosPod, err := r.generatePod(instance, &targetPod, args, chaostypes.PodModeInject, chaostypes.DisruptionKindNetworkFailure)
			if err != nil {
				return ctrl.Result{}, err
			}

			chaosPods = append(chaosPods, chaosPod)
		}

		if instance.Spec.NetworkLatency != nil {
			args := instance.Spec.NetworkLatency.GenerateArgs(chaostypes.PodModeInject, instance.UID, containerID, r.MetricsSink.GetSinkName())

			chaosPod, err := r.generatePod(instance, &targetPod, args, chaostypes.PodModeInject, chaostypes.DisruptionKindNetworkLatency)
			if err != nil {
				return ctrl.Result{}, err
			}

			chaosPods = append(chaosPods, chaosPod)
		}

		if instance.Spec.NodeFailure != nil {
			args := instance.Spec.NodeFailure.GenerateArgs(chaostypes.PodModeInject, instance.UID, containerID, r.MetricsSink.GetSinkName())

			chaosPod, err := r.generatePod(instance, &targetPod, args, chaostypes.PodModeInject, chaostypes.DisruptionKindNodeFailure)
			if err != nil {
				return ctrl.Result{}, err
			}

			chaosPods = append(chaosPods, chaosPod)
		}

		if instance.Spec.CPUPressure != nil {
			args := instance.Spec.CPUPressure.GenerateArgs(chaostypes.PodModeInject, instance.UID, containerID, r.MetricsSink.GetSinkName())

			chaosPod, err := r.generatePod(instance, &targetPod, args, chaostypes.PodModeInject, chaostypes.DisruptionKindCPUPressure)
			if err != nil {
				return ctrl.Result{}, err
			}

			chaosPods = append(chaosPods, chaosPod)
		}

		if instance.Spec.NetworkLimitation != nil {
			args := instance.Spec.NetworkLimitation.GenerateArgs(chaostypes.PodModeInject, instance.UID, containerID, r.MetricsSink.GetSinkName())

			chaosPod, err := r.generatePod(instance, &targetPod, args, chaostypes.PodModeInject, chaostypes.DisruptionKindNetworkLimitation)
			if err != nil {
				return ctrl.Result{}, err
			}

			chaosPods = append(chaosPods, chaosPod)
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

// cleanFailures creates cleanup pods for a given disruption instance
func (r *DisruptionReconciler) cleanFailures(instance *chaosv1beta1.Disruption) error {
	// retrieve pods to cleanup
	pods, err := r.getPodsToCleanup(instance)
	if err != nil {
		return err
	}

	// create one cleanup pod for pod to cleanup
	for _, p := range pods {
		chaosPods := []*v1.Pod{}

		// get ID of first container
		containerID, err := getContainerID(p)
		if err != nil {
			return err
		}

		// generate cleanup pods specs
		if instance.Spec.NetworkFailure != nil {
			args := instance.Spec.NetworkFailure.GenerateArgs(chaostypes.PodModeClean, instance.UID, containerID, r.MetricsSink.GetSinkName())

			chaosPod, err := r.generatePod(instance, p, args, chaostypes.PodModeClean, chaostypes.DisruptionKindNetworkFailure)
			if err != nil {
				return err
			}

			chaosPods = append(chaosPods, chaosPod)
		}

		if instance.Spec.NetworkLatency != nil {
			args := instance.Spec.NetworkLatency.GenerateArgs(chaostypes.PodModeClean, instance.UID, containerID, r.MetricsSink.GetSinkName())

			chaosPod, err := r.generatePod(instance, p, args, chaostypes.PodModeClean, chaostypes.DisruptionKindNetworkLatency)
			if err != nil {
				return err
			}

			chaosPods = append(chaosPods, chaosPod)
		}

		if instance.Spec.NetworkLimitation != nil {
			args := instance.Spec.NetworkLimitation.GenerateArgs(chaostypes.PodModeClean, instance.UID, containerID, r.MetricsSink.GetSinkName())

			chaosPod, err := r.generatePod(instance, p, args, chaostypes.PodModeClean, chaostypes.DisruptionKindNetworkLimitation)
			if err != nil {
				return err
			}

			chaosPods = append(chaosPods, chaosPod)
		}

		// create cleanup pods
		for _, chaosPod := range chaosPods {
			found := &corev1.Pod{}

			// link cleanup pod to instance for garbage collection
			if err := controllerutil.SetControllerReference(instance, chaosPod, r.Scheme); err != nil {
				return err
			}

			// do nothing if cleanup pod already exists
			err = r.Get(context.Background(), types.NamespacedName{Name: chaosPod.Name, Namespace: chaosPod.Namespace}, found)
			if err != nil && reflect.DeepEqual(chaosPod.Spec, found.Spec) {
				continue
			} else if err != nil && !errors.IsNotFound(err) {
				return err
			}

			r.Log.Info("creating chaos cleanup chaosPod", "instance", instance.Name, "namespace", chaosPod.Namespace, "name", chaosPod.Name, "containerid", containerID)

			err = r.Create(context.Background(), chaosPod)
			if err != nil {
				r.Recorder.Event(instance, "Warning", "Create failed", fmt.Sprintf("Cleanup pod for disruption \"%s\" failed to be created", instance.Name))
				r.handleMetricSinkError(r.MetricsSink.MetricPodsCreated(p.ObjectMeta.Name, instance.Name, instance.Namespace, "cleanup", false))

				return err
			}

			r.Recorder.Event(instance, "Normal", "Created", fmt.Sprintf("Created cleanup pod for disruption \"%s\"", instance.Name))
			r.handleMetricSinkError(r.MetricsSink.MetricPodsCreated(p.ObjectMeta.Name, instance.Name, instance.Namespace, "cleanup", true))
		}
	}

	return nil
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

	// if count has not been specified or is greater than the actual number of matching pods,
	// return all pods matching the label selector
	if instance.Spec.Count == -1 || instance.Spec.Count >= len(allPods.Items) {
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
	numPodsToSelect := int(math.Min(float64(instance.Spec.Count), float64(instance.Spec.Count-len(instance.Status.TargetPods))))
	for i := 0; i < numPodsToSelect; i++ {
		// select and add a random pod
		index := rand.Intn(len(allPods.Items))
		chosenPod := allPods.Items[index]
		selectedPods = append(selectedPods, chosenPod)

		r.Log.Info("Selected random pod", "name", chosenPod.Name, "namespace", chosenPod.Namespace, "disruption", instance.Name, "count", instance.Spec.Count)

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

// getChaosPods returns all pods created by the given Disruption instance and being in the given mode (injection or cleanup)
func (r *DisruptionReconciler) getChaosPods(instance *chaosv1beta1.Disruption, mode chaostypes.PodMode) ([]corev1.Pod, error) {
	selector := map[string]string{
		chaostypes.PodModeLabel: string(mode),
	}

	return r.getOwnedPods(instance, selector)
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
func (r *DisruptionReconciler) generatePod(instance *chaosv1beta1.Disruption, target *corev1.Pod, args []string, mode chaostypes.PodMode, kind chaostypes.DisruptionKind) (*corev1.Pod, error) {
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

	return &pod, nil
}

// handleMetricSinkError logs the given metric sink error if it is not nil
func (r *DisruptionReconciler) handleMetricSinkError(err error) {
	if err != nil {
		r.Log.Error(err, "error sending a metric")
	}
}

// SetupWithManager setups the current reconciler with the given manager
func (r *DisruptionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chaosv1beta1.Disruption{}).
		Complete(r)
}
