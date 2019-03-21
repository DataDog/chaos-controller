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

package dependencyfailureinjection

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	chaosv1beta1 "github.com/DataDog/chaos-fi-controller/pkg/apis/chaos/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const cleanupFinalizer = "clean.dfi.finalizer.datadog.com"

var log = logf.Log.WithName("controller")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new DependencyFailureInjection Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileDependencyFailureInjection{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("dependencyfailureinjection-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to DependencyFailureInjection
	err = c.Watch(&source.Kind{Type: &chaosv1beta1.DependencyFailureInjection{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &chaosv1beta1.DependencyFailureInjection{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileDependencyFailureInjection{}

// ReconcileDependencyFailureInjection reconciles a DependencyFailureInjection object
type ReconcileDependencyFailureInjection struct {
	client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a DependencyFailureInjection object and makes changes based on the state read
// and what is in the DependencyFailureInjection.Spec
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=,resources=pods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=chaos.datadoghq.com,resources=dependencyfailureinjections,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=chaos.datadoghq.com,resources=dependencyfailureinjections/status,verbs=get;update;patch
func (r *ReconcileDependencyFailureInjection) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the DependencyFailureInjection instance
	instance := &chaosv1beta1.DependencyFailureInjection{}
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
			if err := r.cleanFailures(instance); err != nil {
				// if fail to clean injected dependency failures, return with error
				// so that it can be retried
				return reconcile.Result{}, err
			}

			// remove our finalizer from the list and update it.
			instance.ObjectMeta.Finalizers = removeString(instance.ObjectMeta.Finalizers, cleanupFinalizer)
			if err := r.Update(context.Background(), instance); err != nil {
				return reconcile.Result{}, err
			}
		}

		// Our finalizer has finished, so the reconciler can do nothing.
		return reconcile.Result{}, nil
	}

	// Get pods matching the DependencyFailureInjection's label selector
	pods, err := r.getMatchingPods(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	// For each pod found, start a chaos pod on the same node
	isPrivileged := true
	hostPathType := corev1.HostPathType("Directory")
	for _, p := range pods.Items {
		// Get ID of first container
		containerId, err := r.getContainerdId(&p)
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
						Name:            "chaos-fi-inject",
						Image:           "eu.gcr.io/datadog-staging/chaos-fi:0.0.1",
						ImagePullPolicy: "Always",
						Command:         []string{"cmd"},
						Args: []string{
							"inject",
							"network-failure",
							"--container-id",
							containerId,
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

		if err := controllerutil.SetControllerReference(instance, pod, r.scheme); err != nil {
			return reconcile.Result{}, err
		}

		// Check if the pod already exists
		found := &corev1.Pod{}
		err = r.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
		if err != nil && errors.IsNotFound(err) {
			log.Info("Creating chaos pod", "namespace", pod.Namespace, "name", pod.Name, "nodename", nodeName)
			err = r.Create(context.TODO(), pod)
			return reconcile.Result{}, err
		} else if err != nil {
			return reconcile.Result{}, err
		}

		// Update the found object and write the result back if there are any changes
		if !reflect.DeepEqual(pod.Spec, found.Spec) {
			found.Spec = pod.Spec
			log.Info("Updating chaos pod", "namespace", pod.Namespace, "name", pod.Name, "nodename", nodeName)
			err = r.Update(context.TODO(), found)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
	}
	return reconcile.Result{}, nil
}

// cleanFailures gets called for DependencyFailureInjection objects with the cleanupFinalizer finalizer.
// A chaos pod will get created that cleans up injected dependency failures.
func (r *ReconcileDependencyFailureInjection) cleanFailures(instance *chaosv1beta1.DependencyFailureInjection) error {
	isPrivileged := true
	hostPathType := corev1.HostPathType("Directory")

	pods, err := r.getMatchingPods(instance)
	if err != nil {
		return err
	}

	for _, p := range pods.Items {
		// Get ID of first container
		containerId, err := r.getContainerdId(&p)
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
						Name:            "chaos-fi-cleanup",
						Image:           "eu.gcr.io/datadog-staging/chaos-fi:0.0.1",
						ImagePullPolicy: "Always",
						Command:         []string{"cmd"},
						Args:            []string{"clean", "--container-id", containerId},
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

		found := &corev1.Pod{}
		err = r.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
		if err != nil && errors.IsAlreadyExists(err) {
			continue
		} else if err != nil && !errors.IsNotFound(err) {
			return err
		}

		log.Info("Creating chaos cleanup pod", "namespace", pod.Namespace, "name", pod.Name, "containerid", containerId)
		err = r.Create(context.TODO(), pod)
		if err != nil {
			return err
		}
	}
	return nil
}

// getMatchingPods returns a PodList containing all pods matching the DependencyFailureInjection's label selector.
func (r *ReconcileDependencyFailureInjection) getMatchingPods(instance *chaosv1beta1.DependencyFailureInjection) (*corev1.PodList, error) {
	// Fetch pods from label selector
	labelSelector := instance.Spec.LabelSelector
	listOptions := &client.ListOptions{}
	err := listOptions.SetLabelSelector(labelSelector)
	if err != nil {
		return nil, err
	}

	pods := &corev1.PodList{}
	err = r.Client.List(context.TODO(), listOptions, pods)
	if err != nil {
		return nil, err
	}

	return pods, nil
}

// getContainerdId gets the ID of the first container ID found in a Pod.
// It expects container ids to follow the format "containerd://<ID>".
func (r *ReconcileDependencyFailureInjection) getContainerdId(pod *corev1.Pod) (string, error) {
	if len(pod.Status.ContainerStatuses) < 1 {
		return "", fmt.Errorf("Missing container ids for pod '%s'", pod.Name)
	}

	containerId := strings.Split(pod.Status.ContainerStatuses[0].ContainerID, "containerd://")
	if len(containerId) != 2 {
		return "", fmt.Errorf("Unrecognized container ID format '%s', expecting 'containerd://<ID>'", pod.Status.ContainerStatuses[0].ContainerID)
	}

	return containerId[1], nil
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
