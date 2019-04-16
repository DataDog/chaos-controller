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
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	chaosv1beta1 "github.com/DataDog/chaos-fi-controller/pkg/apis/chaos/v1beta1"
	"github.com/DataDog/chaos-fi-controller/pkg/datadog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	// ChaosFailureInjectionImageVariableName is the name of the chaos failure injection image variable
	ChaosFailureInjectionImageVariableName = "CHAOS_FI_IMAGE"

	metricPrefix = "chaos.controller.nfi"
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

			log.Info("Checking status of cleanup pods before deleting NFI", "numCleanupPods", len(cleanupPods), "NetworkFailureInjection", instance.Name)

			for _, cleanupPod := range cleanupPods {
				if cleanupPod.Status.Phase != corev1.PodSucceeded {
					log.Info("Cleanup pod has not completed, retrying NFI deletion", "NetworkFailureInjection", instance.Name, "cleanupPod", cleanupPod.Name, "phase", cleanupPod.Status.Phase)
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

	// Get pods matching the NetworkFailureInjection's label selector
	pods, err := r.getMatchingPods(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	// For each pod found, start a chaos pod on the same node
	isPrivileged := true
	hostPathType := corev1.HostPathType("Directory")
	for _, p := range pods.Items {
		// Get ID of first container
		containerID, err := r.getContainerdID(&p)
		if err != nil {
			return reconcile.Result{}, err
		}

		nodeName := p.Spec.NodeName

		// Define the desired pod object
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      instance.Name + "-inject-" + p.Name + "-pod",
				Namespace: instance.Namespace,
			},
			Spec: corev1.PodSpec{
				NodeName:      nodeName,
				RestartPolicy: "Never",
				Containers: []corev1.Container{
					{
						Name:    "chaos-fi-inject",
						Image:   os.Getenv(ChaosFailureInjectionImageVariableName),
						Command: []string{"cmd"},
						Args: []string{
							"inject",
							"network-failure",
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
						VolumeMounts: []corev1.VolumeMount{
							corev1.VolumeMount{
								MountPath: "/run/containerd",
								Name:      "containerd",
							},
							corev1.VolumeMount{
								MountPath: "/mnt/proc",
								Name:      "proc",
							},
						},
						SecurityContext: &corev1.SecurityContext{
							Privileged: &isPrivileged,
						},
					},
				},
				Volumes: []corev1.Volume{
					corev1.Volume{
						Name: "containerd",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/run/containerd",
								Type: &hostPathType,
							},
						},
					},
					corev1.Volume{
						Name: "proc",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/proc",
								Type: &hostPathType,
							},
						},
					},
				},
			},
		}

		// Link the NFI resource and the chaos pod for garbage collection
		if err := controllerutil.SetControllerReference(instance, pod, r.scheme); err != nil {
			return reconcile.Result{}, err
		}

		// Create the injection pod if it does not exist and continue the loop
		found := &corev1.Pod{}
		err = r.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
		if err != nil && errors.IsNotFound(err) {
			log.Info("Creating chaos pod", "namespace", pod.Namespace, "name", pod.Name, "nodename", nodeName)
			err = r.Create(context.TODO(), pod)
			if err != nil {
				r.recorder.Event(instance, "Warning", "Create failed", fmt.Sprintf("Failure injection pod for networkfailureinjection \"%s\" failed to be created", instance.Name))
				return reconcile.Result{}, err
			}
			r.recorder.Event(instance, "Normal", "Created", fmt.Sprintf("Created failure injection pod for networkfailureinjection \"%s\"", instance.Name))
			datadog.GetInstance().Incr(metricPrefix+".pods.created", []string{"phase:inject", "target:" + p.ObjectMeta.Name, "name:" + instance.Name, "namespace:" + instance.Namespace}, 1)
			continue
		} else if err != nil {
			return reconcile.Result{}, err
		}
	}

	// We reach this line only when every injection pods have been created with success
	datadog.GetInstance().Timing(metricPrefix+".inject.duration", time.Since(tsStart), []string{"name:" + instance.Name, "namespace:" + instance.Namespace}, 1)

	return reconcile.Result{}, nil
}

// cleanFailures gets called for NetworkFailureInjection objects with the cleanupFinalizer finalizer.
// A chaos pod will get created that cleans up injected network failures.
func (r *ReconcileNetworkFailureInjection) cleanFailures(instance *chaosv1beta1.NetworkFailureInjection) error {
	isPrivileged := true
	hostPathType := corev1.HostPathType("Directory")

	pods, err := r.getMatchingPods(instance)
	if err != nil {
		return err
	}

	for _, p := range pods.Items {
		// Get ID of first container
		containerID, err := r.getContainerdID(&p)
		if err != nil {
			return err
		}

		// Define the cleanup pod object
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      instance.Name + "-cleanup-" + p.Name + "-pod",
				Namespace: instance.Namespace,
			},
			Spec: corev1.PodSpec{
				NodeName:      p.Spec.NodeName,
				RestartPolicy: "Never",
				Containers: []corev1.Container{
					{
						Name:    cleanupContainerName,
						Image:   os.Getenv(ChaosFailureInjectionImageVariableName),
						Command: []string{"cmd"},
						Args:    []string{"clean", "--uid", string(instance.ObjectMeta.UID), "--container-id", containerID},
						VolumeMounts: []corev1.VolumeMount{
							corev1.VolumeMount{
								MountPath: "/run/containerd",
								Name:      "containerd",
							},
							corev1.VolumeMount{
								MountPath: "/mnt/proc",
								Name:      "proc",
							},
						},
						SecurityContext: &corev1.SecurityContext{
							Privileged: &isPrivileged,
						},
					},
				},
				Volumes: []corev1.Volume{
					corev1.Volume{
						Name: "containerd",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/run/containerd",
								Type: &hostPathType,
							},
						},
					},
					corev1.Volume{
						Name: "proc",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/proc",
								Type: &hostPathType,
							},
						},
					},
				},
			},
		}

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

		// Create the cleanup pod
		log.Info("Creating chaos cleanup pod", "namespace", pod.Namespace, "name", pod.Name, "containerid", containerID)
		err = r.Create(context.TODO(), pod)
		if err != nil {
			r.recorder.Event(instance, "Warning", "Create failed", fmt.Sprintf("Cleanup pod for networkfailureinjection \"%s\" failed to be created", instance.Name))
			return err
		}
		r.recorder.Event(instance, "Normal", "Created", fmt.Sprintf("Created cleanup pod for networkfailureinjection \"%s\"", instance.Name))
		datadog.GetInstance().Incr(metricPrefix+".pods.created", []string{"phase:cleanup", "target:" + p.ObjectMeta.Name, "name:" + instance.Name, "namespace:" + instance.Namespace}, 1)
	}
	return nil
}

// getMatchingPods returns a PodList containing all pods matching the NetworkFailureInjection's label selector and namespace.
func (r *ReconcileNetworkFailureInjection) getMatchingPods(instance *chaosv1beta1.NetworkFailureInjection) (*corev1.PodList, error) {
	// We want to ensure we never run into the possibility of using an empty label selector
	labelSelector := instance.Spec.Selector
	if len(labelSelector) < 1 || labelSelector == nil {
		err := fmt.Errorf("nfi '%s' in namespace '%s' is missing a label selector", instance.Name, instance.Namespace)
		log.Error(err, "missing label selector", "namespace", instance.Namespace, "name", instance.Name)
		return nil, err
	}

	// Filter pods based on the nfi's label selector, and only consider those within the same namespace as the nfi
	listOptions := &client.ListOptions{
		LabelSelector: labelSelector.AsSelector(),
		Namespace:     instance.Namespace,
	}

	// Fetch pods from label selector
	pods := &corev1.PodList{}
	err := r.Client.List(context.TODO(), listOptions, pods)
	if err != nil {
		return nil, err
	}

	return pods, nil
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
