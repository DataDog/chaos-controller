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
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	chaosv1beta1 "github.com/DataDog/chaos-fi-controller/api/v1beta1"
	"github.com/DataDog/chaos-fi-controller/datadog"
	"github.com/DataDog/chaos-fi-controller/helpers"
	chaostypes "github.com/DataDog/chaos-fi-controller/types"
)

const (
	metricPrefixNetworkLatency = "chaos.controller.nfl"
)

// NetworkLatencyInjectionReconciler reconciles a NetworkLatencyInjection object
type NetworkLatencyInjectionReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=chaos.datadoghq.com,resources=networklatencyinjections,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=chaos.datadoghq.com,resources=networklatencyinjections/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch

// Reconcile loop
func (r *NetworkLatencyInjectionReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("networklatencyinjection", req.NamespacedName)

	// your logic here
	instance := &chaosv1beta1.NetworkLatencyInjection{}
	datadog.GetInstance().Incr(metricPrefix+".reconcile", nil, 1)
	tsStart := time.Now()
	defer func() func() {
		return func() {
			tags := []string{}
			if instance.Name != "" {
				tags = append(tags, "name:"+instance.Name, "namespace:"+instance.Namespace)
			}
			datadog.GetInstance().Timing(metricPrefix+".reconcile.duration", time.Since(tsStart), tags, 1)
		}
	}()()

	// Fetch the NetworkLatencyInjection instance
	if err := r.Get(context.Background(), req.NamespacedName, instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// The object is not being deleted, so if it does not have our finalizer,
	// then lets add the finalizer and update the object.
	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		if !containsString(instance.ObjectMeta.Finalizers, cleanupFinalizer) {
			instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, cleanupFinalizer)
			if err := r.Update(context.Background(), instance); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted; call the clean finalizer
		// to remove injected failures
		if containsString(instance.ObjectMeta.Finalizers, cleanupFinalizer) {
			if instance.Status.Finalizing == false {
				if err := r.cleanFailures(instance); err != nil {
					// if fail to clean injected network failures, return with error
					// so that it can be retried
					return ctrl.Result{}, err
				}

				// set the finalizing flag so we can remove the finalizers once the cleanup pod
				// has finisheR its work
				instance.Status.Finalizing = true
				if err := r.Update(context.Background(), instance); err != nil {
					return ctrl.Result{}, err
				}

				return ctrl.Result{}, nil
			}

			cleanupPods, err := r.getCleanupPods(instance)
			if err != nil {
				return ctrl.Result{}, err
			}

			r.Log.Info("Checking status of cleanup pods before deleting nfl", "numcleanuppods", len(cleanupPods), "networklatencyinjection", instance.Name, "namespace", instance.Namespace)

			// Check if cleanup pods have succeeded, requeue until they have
			for _, cleanupPod := range cleanupPods {
				if cleanupPod.Status.Phase != corev1.PodSucceeded {
					r.Log.Info("Cleanup pod has not completed, retrying nfi deletion", "networklatencyinjection", instance.Name, "namespace", instance.Namespace, "cleanuppod", cleanupPod.Name, "phase", cleanupPod.Status.Phase)
					return ctrl.Result{
						Requeue: true,
					}, nil
				}
			}

			// Remove our finalizer from the list and update it.
			instance.ObjectMeta.Finalizers = removeString(instance.ObjectMeta.Finalizers, cleanupFinalizer)
			if err := r.Update(context.Background(), instance); err != nil {
				return ctrl.Result{}, err
			}
			datadog.GetInstance().Timing(metricPrefix+".cleanup.duration", time.Since(tsStart), []string{"name:" + instance.Name, "namespace:" + instance.Namespace}, 1)
		}

		// Our finalizer has finished, so the reconciler can do nothing.
		return ctrl.Result{}, nil
	}

	// Skip the nfi if inject pods were already created for it
	if instance.Status.Injected {
		return ctrl.Result{}, nil
	}

	// Get pods to inject failures into. If Count was not specified, this includes all pods
	// matching the nfi's label selector and namespace.
	// Otherwise, inject failures into Count randomly-selected pods matching the nfi's label selector and namespace.
	pods, err := r.selectPodsForInjection(instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	// For each pod found, start a chaos pod on the same node
	for _, p := range pods.Items {
		// Get ID of first container
		containerID, err := helpers.GetContainerdID(&p)
		if err != nil {
			return ctrl.Result{}, err
		}

		nodeName := p.Spec.NodeName

		// Define the desired pod object
		args := []string{
			"network-latency",
			"inject",
			"--uid",
			string(instance.ObjectMeta.UID),
			"--container-id",
			containerID,
			"--delay",
			strconv.Itoa(int(instance.Spec.Delay)),
			"--hosts",
		}
		args = append(args, strings.Split(strings.Join(instance.Spec.Hosts, " --hosts "), " ")...)
		pod := helpers.GeneratePod(
			instance.Name+"-inject-"+p.Name+"-pod",
			&p,
			args,
			chaostypes.PodModeInject,
		)

		// Link the NFI resource and the chaos pod for garbage collection
		if err := controllerutil.SetControllerReference(instance, pod, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}

		// Create the injection pod if it does not exist
		found := &corev1.Pod{}
		err = r.Get(context.Background(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
		if err != nil && errors.IsNotFound(err) {
			// Create the inject pod
			r.Log.Info("Creating chaos pod", "networklatencyinjection", instance.Name, "namespace", pod.Namespace, "name", pod.Name, "nodename", nodeName)
			if err = r.Create(context.Background(), pod); err != nil {
				r.Recorder.Event(instance, "Warning", "Create failed", fmt.Sprintf("Failure injection pod for networklatencyinjection \"%s\" failed to be created", instance.Name))
				return ctrl.Result{}, err
			}

			// Send metrics
			r.Recorder.Event(instance, "Normal", "Created", fmt.Sprintf("Created failure injection pod for networklatencyinjection \"%s\"", instance.Name))
			datadog.GetInstance().Incr(metricPrefix+".pods.created", []string{"phase:inject", "target_pod:" + p.ObjectMeta.Name, "name:" + instance.Name, "namespace:" + instance.Namespace}, 1)

			continue
		} else if err != nil {
			return ctrl.Result{}, err
		}
		// Do nothing if the inject pod already exists
	}

	instance.Status.Injected = true
	if err = r.Update(context.Background(), instance); err != nil {
		return ctrl.Result{}, err
	}

	// We reach this line only when every injection pods have been created with success
	datadog.GetInstance().Timing(metricPrefix+".inject.duration", time.Since(tsStart), []string{"name:" + instance.Name, "namespace:" + instance.Namespace}, 1)

	return ctrl.Result{}, nil
}

// getPodsToCleanup returns the still-existing pods that were targeted by the nfi, according to the pod names in the Status.Pods
func (r *NetworkLatencyInjectionReconciler) getPodsToCleanup(instance *chaosv1beta1.NetworkLatencyInjection) ([]*corev1.Pod, error) {
	podsToCleanup := make([]*corev1.Pod, 0, len(instance.Status.Pods))

	// Check if each pod still exists; skip if it doesn't
	for _, podName := range instance.Status.Pods {
		// Get the targeted pods' names from the Status.Pods
		podKey := types.NamespacedName{Name: podName, Namespace: instance.Namespace}
		p := &corev1.Pod{}

		// Try to get the pod
		err := r.Get(context.Background(), podKey, p)
		// Skip the pod when it no longer exists
		if errors.IsNotFound(err) {
			r.Log.Info("Cleanup: Pod no longer exists", "networklatencyinjection", instance.Name, "namespace", instance.Namespace, "name", podName, "numpodstotarget", instance.Spec.Count)
			continue
		} else if err != nil {
			return nil, err
		}

		podsToCleanup = append(podsToCleanup, p)
	}

	return podsToCleanup, nil
}

// cleanFailures gets called for NetworkLatencyInjection objects with the cleanupFinalizer finalizer.
// A chaos pod will get created that cleans up injected network failures.
func (r *NetworkLatencyInjectionReconciler) cleanFailures(instance *chaosv1beta1.NetworkLatencyInjection) error {
	pods, err := r.getPodsToCleanup(instance)
	if err != nil {
		return err
	}

	for _, p := range pods {
		// Get ID of first container
		containerID, err := helpers.GetContainerdID(p)
		if err != nil {
			return err
		}

		// Define the cleanup pod object
		args := []string{
			"network-latency",
			"clean",
			"--uid",
			string(instance.ObjectMeta.UID),
			"--container-id",
			containerID,
			"--hosts",
		}
		args = append(args, strings.Split(strings.Join(instance.Spec.Hosts, " --hosts "), " ")...)
		pod := helpers.GeneratePod(
			instance.Name+"-cleanup-"+p.Name+"-pod",
			p,
			args,
			chaostypes.PodModeClean,
		)

		if err := controllerutil.SetControllerReference(instance, pod, r.Scheme); err != nil {
			return err
		}

		// Check if cleanup pod already exists
		found := &corev1.Pod{}
		err = r.Get(context.Background(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
		// Do nothing if cleanup pod already exists, or else
		// retry Reconcile if we get an error other than the cleanup pod does not exist
		if err != nil && reflect.DeepEqual(pod.Spec, found.Spec) {
			continue
		} else if err != nil && !errors.IsNotFound(err) {
			return err
		}

		r.Log.Info("Creating chaos cleanup pod", "networkfailureinjection", instance.Name, "namespace", pod.Namespace, "name", pod.Name, "containerid", containerID)
		err = r.Create(context.Background(), pod)
		if err != nil {
			r.Recorder.Event(instance, "Warning", "Create failed", fmt.Sprintf("Cleanup pod for networkfailureinjection \"%s\" failed to be created", instance.Name))
			return err
		}
		r.Recorder.Event(instance, "Normal", "Created", fmt.Sprintf("Created cleanup pod for networkfailureinjection \"%s\"", instance.Name))
		datadog.GetInstance().Incr(metricPrefix+".pods.created", []string{"phase:cleanup", "target_pod:" + p.ObjectMeta.Name, "name:" + instance.Name, "namespace:" + instance.Namespace}, 1)
	}
	return nil
}

// updateStatusPods will update the nfi instance's status.pods field, if it was not previously set,
// to show the names of pods selected for injection for that instance.
func (r *NetworkLatencyInjectionReconciler) updateStatusPods(instance *chaosv1beta1.NetworkLatencyInjection, pods *corev1.PodList) error {
	// Ignore if we already set the status.pods
	if len(instance.Status.Pods) > 0 {
		return nil
	}

	// Get all the pod names
	for _, pod := range pods.Items {
		instance.Status.Pods = append(instance.Status.Pods, pod.Name)
	}

	// Update the instance's status.pods
	err := r.Update(context.Background(), instance)
	if err != nil {
		r.Log.Error(err, "failed to update nfi's status.pods", "networkfailureinjection", instance.Name, "namespace", instance.Namespace)
		return err
	}

	r.Log.Info("Updated nfi status with pods selected for injection", "networkfailureinjection", instance.Name, "namespace", instance.Namespace, "numpodstotarget", instance.Spec.Count)

	return nil
}

// selectPodsForInjection will select min(Count, all matching pods) random pods from the pods matching the nfi.
// Pods will only be selected once per nfi. The chosen pods' names will be reflected in an nfi's status.pods.
// Subsequent calls to selectPodsForInjection will always return the same pods as the first time selectPodsForInjection
// was successfully called, to ensure that we never select different pods in case we fail to create an inject pod.
func (r *NetworkLatencyInjectionReconciler) selectPodsForInjection(instance *chaosv1beta1.NetworkLatencyInjection) (*corev1.PodList, error) {
	allPods, err := helpers.GetMatchingPods(r.Client, instance.Namespace, instance.Spec.Selector)

	// If Count was not specified, or is greater than the actual number of matching pods,
	// return all pods matching the nfi's label selector and namespace
	if instance.Spec.Count == nil || *instance.Spec.Count >= len(allPods.Items) {
		// Update status.pods with the names of all pods matching the label selector in the same namespace
		err := r.updateStatusPods(instance, allPods)
		if err != nil {
			return nil, err
		}

		return allPods, nil
	}

	// If we had already selected pods for the nfi, only return the already-selected ones to avoid selecting
	// > Count pods if creating an inject pod ever fails
	p := []corev1.Pod{}
	if len(instance.Status.Pods) > 0 {
		for _, pod := range allPods.Items {
			if containsString(instance.Status.Pods, pod.Name) {
				p = append(p, pod)
			}
		}
		allPods.Items = p
		return allPods, nil
	}

	// Otherwise, randomly select Count pods
	rand.Seed(time.Now().UnixNano())
	numPodsToSelect := int(math.Min(float64(*instance.Spec.Count), float64(*instance.Spec.Count-len(instance.Status.Pods))))

	for i := 0; i < numPodsToSelect; i++ {
		// Select and add a random pod
		index := rand.Intn(len(allPods.Items))
		chosenPod := allPods.Items[index]
		p = append(p, chosenPod)

		r.Log.Info("Selected random pod", "name", chosenPod.Name, "namespace", chosenPod.Namespace, "networkfailureinjection", instance.Name, "numpodstotarget", instance.Spec.Count)

		// Remove chosen pod from list of pods from which to select
		allPods.Items[len(allPods.Items)-1], allPods.Items[index] = allPods.Items[index], allPods.Items[len(allPods.Items)-1]
		allPods.Items = allPods.Items[:len(allPods.Items)-1]
	}

	allPods.Items = p

	// Update the status.pods with the names of the randomly chosen pods
	err = r.updateStatusPods(instance, allPods)
	if err != nil {
		return nil, err
	}

	return allPods, nil
}

// getCleanupPods returns all cleanup pods created by the NetworkLatencyInjection instance.
func (r *NetworkLatencyInjectionReconciler) getCleanupPods(instance *chaosv1beta1.NetworkLatencyInjection) ([]corev1.Pod, error) {
	podsInNs := &corev1.PodList{}
	listOptions := &client.ListOptions{
		Namespace: instance.Namespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{
			chaostypes.PodModeLabel: chaostypes.PodModeClean,
		}),
	}
	err := r.Client.List(context.Background(), podsInNs, listOptions)
	if err != nil {
		return nil, err
	}

	// Filter all pods in the same namespace as instance,
	// only returning those owned by instance and which are cleanup pods
	pods := make([]corev1.Pod, 0)
	for _, pod := range podsInNs.Items {
		for _, ownerReference := range pod.OwnerReferences {
			if ownerReference.UID != instance.UID {
				continue
			}
			if len(pod.Spec.Containers) > 0 {
				pods = append(pods, pod)
			}
		}
	}
	return pods, nil
}

// SetupWithManager setups the current reconciler with the given manager
func (r *NetworkLatencyInjectionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chaosv1beta1.NetworkLatencyInjection{}).
		Complete(r)
}
