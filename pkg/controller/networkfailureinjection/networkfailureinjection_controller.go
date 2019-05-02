/*
Copyright 2019 Datadog.

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

package networkfailureinjection

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"time"

	chaosv1beta1 "github.com/DataDog/chaos-fi-controller/pkg/apis/chaos/v1beta1"
	"github.com/DataDog/chaos-fi-controller/pkg/datadog"
	"github.com/DataDog/chaos-fi-controller/pkg/helpers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	cleanupContainerName = "chaos-fi-cleanup"
	cleanupFinalizer     = "clean.nfi.finalizer.datadog.com"
	metricPrefix         = "chaos.controller.nfi"
)

var log = logf.Log.WithName("controller")

// Add creates a new NetworkFailureInjection Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileNetworkFailureInjection{
		Client:   mgr.GetClient(),
		scheme:   mgr.GetScheme(),
		recorder: mgr.GetRecorder("networkfailureinjection-controller"),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("networkfailureinjection-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to NetworkFailureInjection
	err = c.Watch(&source.Kind{Type: &chaosv1beta1.NetworkFailureInjection{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &chaosv1beta1.NetworkFailureInjection{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileNetworkFailureInjection{}

// ReconcileNetworkFailureInjection reconciles a NetworkFailureInjection object
type ReconcileNetworkFailureInjection struct {
	client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

// Reconcile reads that state of the cluster for a NetworkFailureInjection object and makes changes based on the state read
// and what is in the NetworkFailureInjection.Spec
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=,resources=pods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=chaos.datadoghq.com,resources=networkfailureinjections,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=chaos.datadoghq.com,resources=networkfailureinjections/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
func (r *ReconcileNetworkFailureInjection) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	instance := &chaosv1beta1.NetworkFailureInjection{}
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

	// Fetch the NetworkFailureInjection instance
	err := r.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// The object is not being deleted, so if it does not have our finalizer,
	// then lets add the finalizer and update the object.
	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		if !containsString(instance.ObjectMeta.Finalizers, cleanupFinalizer) {
			instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, cleanupFinalizer)
			if err := r.Update(context.Background(), instance); err != nil {
				return reconcile.Result{}, err
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
					return reconcile.Result{}, err
				}

				// set the finalizing flag so we can remove the finalizers once the cleanup pod
				// has finished its work
				instance.Status.Finalizing = true
				if err := r.Update(context.Background(), instance); err != nil {
					return reconcile.Result{}, err
				}

				return reconcile.Result{}, nil
			}

			cleanupPods, err := r.getCleanupPods(instance)
			if err != nil {
				return reconcile.Result{}, err
			}

			log.Info("Checking status of cleanup pods before deleting nfi", "numcleanuppods", len(cleanupPods), "networkfailureinjection", instance.Name, "namespace", instance.Namespace)

			for _, cleanupPod := range cleanupPods {
				if cleanupPod.Status.Phase != corev1.PodSucceeded {
					log.Info("Cleanup pod has not completed, retrying nfi deletion", "networkfailureinjection", instance.Name, "namespace", instance.Namespace, "cleanuppod", cleanupPod.Name, "phase", cleanupPod.Status.Phase)
					return reconcile.Result{}, nil
				}
			}

			// remove our finalizer from the list and update it.
			instance.ObjectMeta.Finalizers = removeString(instance.ObjectMeta.Finalizers, cleanupFinalizer)
			if err := r.Update(context.Background(), instance); err != nil {
				return reconcile.Result{}, err
			}
			datadog.GetInstance().Timing(metricPrefix+".cleanup.duration", time.Since(tsStart), []string{"name:" + instance.Name, "namespace:" + instance.Namespace}, 1)
		}

		// Our finalizer has finished, so the reconciler can do nothing.
		return reconcile.Result{}, nil
	}

	// Skip the nfi if inject pods were already created for it
	if instance.Status.Injected {
		return reconcile.Result{}, nil
	}

	// Get pods to inject failures into. If NumPodsToTarget was not specified, this includes all pods
	// matching the nfi's label selector and namespace.
	// Otherwise, inject failures into NumPodsToTarget randomly-selected pods matching the nfi's label selector and namespace.
	pods, err := r.selectPodsForInjection(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	// For each pod found, start a chaos pod on the same node
	for _, p := range pods.Items {
		// Get ID of first container
		containerID, err := r.getContainerdID(&p)
		if err != nil {
			return reconcile.Result{}, err
		}

		nodeName := p.Spec.NodeName

		// Define the desired pod object
		pod := helpers.GeneratePod(
			instance.Name+"-inject-"+p.Name+"-pod",
			&p,
			[]string{
				"network-failure",
				"inject",
				"--uid",
				string(instance.ObjectMeta.UID),
				"--container-id",
				containerID,
				"--host",
				instance.Spec.Failure.Host,
				"--port",
				strconv.Itoa(instance.Spec.Failure.Port),
				"--protocol",
				instance.Spec.Failure.Protocol,
			},
		)

		// Link the NFI resource and the chaos pod for garbage collection
		if err := controllerutil.SetControllerReference(instance, pod, r.scheme); err != nil {
			return reconcile.Result{}, err
		}

		// Create the injection pod if it does not exist
		found := &corev1.Pod{}
		err = r.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
		if err != nil && errors.IsNotFound(err) {
			// Create the inject pod
			log.Info("Creating chaos pod", "networkfailureinjection", instance.Name, "namespace", pod.Namespace, "name", pod.Name, "nodename", nodeName)
			err = r.Create(context.TODO(), pod)
			if err != nil {
				r.recorder.Event(instance, "Warning", "Create failed", fmt.Sprintf("Failure injection pod for networkfailureinjection \"%s\" failed to be created", instance.Name))
				return reconcile.Result{}, err
			}

			// Send metrics
			r.recorder.Event(instance, "Normal", "Created", fmt.Sprintf("Created failure injection pod for networkfailureinjection \"%s\"", instance.Name))
			datadog.GetInstance().Incr(metricPrefix+".pods.created", []string{"phase:inject", "target_pod:" + p.ObjectMeta.Name, "name:" + instance.Name, "namespace:" + instance.Namespace}, 1)

			continue
		} else if err != nil {
			return reconcile.Result{}, err
		}
		// Do nothing if the inject pod already exists
	}

	instance.Status.Injected = true
	err = r.Update(context.Background(), instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	// We reach this line only when every injection pods have been created with success
	datadog.GetInstance().Timing(metricPrefix+".inject.duration", time.Since(tsStart), []string{"name:" + instance.Name, "namespace:" + instance.Namespace}, 1)

	return reconcile.Result{}, nil
}

// getPodsToCleanup returns the still-existing pods that were targeted by the nfi, according to the pod names in the Status.Pods
func (r *ReconcileNetworkFailureInjection) getPodsToCleanup(instance *chaosv1beta1.NetworkFailureInjection) ([]*corev1.Pod, error) {
	podsToCleanup := make([]*corev1.Pod, 0, len(instance.Status.Pods))

	// Check if each pod still exists; skip if it doesn't
	for _, podName := range instance.Status.Pods {
		// Get the targeted pods' names from the Status.Pods
		podKey := types.NamespacedName{Name: podName, Namespace: instance.Namespace}
		p := &corev1.Pod{}

		// Try to get the pod
		err := r.Get(context.TODO(), podKey, p)
		// Skip the pod when it no longer exists
		if errors.IsNotFound(err) {
			log.Info("Cleanup: Pod no longer exists", "networkfailureinjection", instance.Name, "namespace", instance.Namespace, "name", podName, "numpodstotarget", instance.Spec.NumPodsToTarget)
			continue
		} else if err != nil {
			return nil, err
		}

		podsToCleanup = append(podsToCleanup, p)
	}

	return podsToCleanup, nil
}

// cleanFailures gets called for NetworkFailureInjection objects with the cleanupFinalizer finalizer.
// A chaos pod will get created that cleans up injected network failures.
func (r *ReconcileNetworkFailureInjection) cleanFailures(instance *chaosv1beta1.NetworkFailureInjection) error {
	pods, err := r.getPodsToCleanup(instance)
	if err != nil {
		return err
	}

	for _, p := range pods {
		// Get ID of first container
		containerID, err := r.getContainerdID(p)
		if err != nil {
			return err
		}

		// Define the cleanup pod object
		pod := helpers.GeneratePod(
			instance.Name+"-cleanup-"+p.Name+"-pod",
			p,
			[]string{
				"network-failure",
				"clean",
				"--uid",
				string(instance.ObjectMeta.UID),
				"--container-id",
				containerID,
			},
		)

		if err := controllerutil.SetControllerReference(instance, pod, r.scheme); err != nil {
			return err
		}

		// Check if cleanup pod already exists
		found := &corev1.Pod{}
		err = r.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
		// Do nothing if cleanup pod already exists, or else
		// retry Reconcile if we get an error other than the cleanup pod does not exist
		if err != nil && reflect.DeepEqual(pod.Spec, found.Spec) {
			continue
		} else if err != nil && !errors.IsNotFound(err) {
			return err
		}

		log.Info("Creating chaos cleanup pod", "networkfailureinjection", instance.Name, "namespace", pod.Namespace, "name", pod.Name, "containerid", containerID)
		err = r.Create(context.TODO(), pod)
		if err != nil {
			r.recorder.Event(instance, "Warning", "Create failed", fmt.Sprintf("Cleanup pod for networkfailureinjection \"%s\" failed to be created", instance.Name))
			return err
		}
		r.recorder.Event(instance, "Normal", "Created", fmt.Sprintf("Created cleanup pod for networkfailureinjection \"%s\"", instance.Name))
		datadog.GetInstance().Incr(metricPrefix+".pods.created", []string{"phase:cleanup", "target_pod:" + p.ObjectMeta.Name, "name:" + instance.Name, "namespace:" + instance.Namespace}, 1)
	}
	return nil
}

// updateStatusPods will update the nfi instance's status.pods field, if it was not previously set,
// to show the names of pods selected for injection for that instance.
func (r *ReconcileNetworkFailureInjection) updateStatusPods(instance *chaosv1beta1.NetworkFailureInjection, pods *corev1.PodList) error {
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
		log.Error(err, "failed to update nfi's status.pods", "networkfailureinjection", instance.Name, "namespace", instance.Namespace)
		return err
	}

	log.Info("Updated nfi status with pods selected for injection", "networkfailureinjection", instance.Name, "namespace", instance.Namespace, "numpodstotarget", instance.Spec.NumPodsToTarget)

	return nil
}

// selectPodsForInjection will select min(NumPodsToTarget, all matching pods) random pods from the pods matching the nfi.
// Pods will only be selected once per nfi. The chosen pods' names will be reflected in an nfi's status.pods.
// Subsequent calls to selectPodsForInjection will always return the same pods as the first time selectPodsForInjection
// was successfully called, to ensure that we never select different pods in case we fail to create an inject pod.
func (r *ReconcileNetworkFailureInjection) selectPodsForInjection(instance *chaosv1beta1.NetworkFailureInjection) (*corev1.PodList, error) {
	allPods, err := helpers.GetMatchingPods(r.Client, instance.Namespace, instance.Spec.Selector)

	// If NumPodsToTarget was not specified, or is greater than the actual number of matching pods,
	// return all pods matching the nfi's label selector and namespace
	if instance.Spec.NumPodsToTarget == nil || *instance.Spec.NumPodsToTarget >= len(allPods.Items) {
		// Update status.pods with the names of all pods matching the label selector in the same namespace
		err := r.updateStatusPods(instance, allPods)
		if err != nil {
			return nil, err
		}

		return allPods, nil
	}

	// If we had already selected pods for the nfi, only return the already-selected ones to avoid selecting
	// > NumPodsToTarget pods if creating an inject pod ever fails
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

	// Otherwise, randomly select NumPodsToTarget pods
	rand.Seed(time.Now().UnixNano())
	numPodsToSelect := int(math.Min(float64(*instance.Spec.NumPodsToTarget), float64(*instance.Spec.NumPodsToTarget-len(instance.Status.Pods))))

	for i := 0; i < numPodsToSelect; i++ {
		// Select and add a random pod
		index := rand.Intn(len(allPods.Items))
		chosenPod := allPods.Items[index]
		p = append(p, chosenPod)

		log.Info("Selected random pod", "name", chosenPod.Name, "namespace", chosenPod.Namespace, "networkfailureinjection", instance.Name, "numpodstotarget", instance.Spec.NumPodsToTarget)

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

// getCleanupPods returns all cleanup pods created by the NetworkFailureInjection instance.
func (r *ReconcileNetworkFailureInjection) getCleanupPods(instance *chaosv1beta1.NetworkFailureInjection) ([]corev1.Pod, error) {
	podsInNs := &corev1.PodList{}
	listOptions := &client.ListOptions{}
	listOptions = listOptions.InNamespace(instance.Namespace)
	err := r.Client.List(context.TODO(), listOptions, podsInNs)
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
			if len(pod.Spec.Containers) > 0 && pod.Spec.Containers[0].Name == cleanupContainerName {
				pods = append(pods, pod)
			}
		}
	}
	return pods, nil
}

// getContainerdID gets the ID of the first container ID found in a Pod.
// It expects container ids to follow the format "containerd://<ID>".
func (r *ReconcileNetworkFailureInjection) getContainerdID(pod *corev1.Pod) (string, error) {
	if len(pod.Status.ContainerStatuses) < 1 {
		return "", fmt.Errorf("Missing container ids for pod '%s'", pod.Name)
	}

	containerID := strings.Split(pod.Status.ContainerStatuses[0].ContainerID, "containerd://")
	if len(containerID) != 2 {
		return "", fmt.Errorf("Unrecognized container ID format '%s', expecting 'containerd://<ID>'", pod.Status.ContainerStatuses[0].ContainerID)
	}

	return containerID[1], nil
}

//
// Helper functions to check and remove string from a slice of strings.
//
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}
