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
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
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
	finalizer = "finalizer.chaos.datadoghq.com"
)

// DisruptionReconciler reconciles a Disruption object
type DisruptionReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=chaos.datadoghq.com,resources=disruptions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=chaos.datadoghq.com,resources=disruptions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch

// Reconcile loop
func (r *DisruptionReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("networklatencyinjection", req.NamespacedName)

	// reconcile metrics
	instance := &chaosv1beta1.Disruption{}
	datadog.GetInstance().Incr("chaos.controller.reconcile", nil, 1)
	tsStart := time.Now()
	defer func() func() {
		return func() {
			tags := []string{}
			if instance.Name != "" {
				tags = append(tags, "name:"+instance.Name, "namespace:"+instance.Namespace)
			}
			datadog.GetInstance().Timing("chaos.controller.reconcile.duration", time.Since(tsStart), tags, 1)
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
			if err := r.Update(context.Background(), instance); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// if being deleted, call the finalizer
		if helpers.ContainsString(instance.ObjectMeta.Finalizers, finalizer) {
			// if the finalizing stage hasn't been triggered yet, start the cleaning
			if instance.Status.IsFinalizing == false {
				if err := r.cleanFailures(instance); err != nil {
					return ctrl.Result{}, err
				}

				// set the finalizing flag
				instance.Status.IsFinalizing = true
				r.Log.Info("updating finalizing flag", "instance", instance.Name, "namespace", instance.Namespace)
				if err := r.Update(context.Background(), instance); err != nil {
					return ctrl.Result{}, err
				}

				// stop the reconcile loop, the next one will take care of checking for finalizer status
				return ctrl.Result{}, nil
			}

			// retrieve cleanup pods to check their states
			cleanupPods, err := r.getChaosPods(instance, chaostypes.PodModeClean)
			if err != nil {
				return ctrl.Result{}, err
			}

			r.Log.Info("checking status of cleanup pods before deleting nfl", "numcleanuppods", len(cleanupPods), "networklatencyinjection", instance.Name, "namespace", instance.Namespace)

			// check if cleanup pods have succeeded, requeue until they have
			for _, cleanupPod := range cleanupPods {
				if cleanupPod.Status.Phase != corev1.PodSucceeded {
					r.Log.Info("cleanup pod has not completed, retrying nfi deletion", "networklatencyinjection", instance.Name, "namespace", instance.Namespace, "cleanuppod", cleanupPod.Name, "phase", cleanupPod.Status.Phase)
					return ctrl.Result{
						Requeue: true,
					}, nil
				}
			}

			// we reach this code when all the cleanup pods have succeeded
			// we can remove the finalizer and let the resource being garbage collected
			r.Log.Info("removing finalizer", "instance", instance.Name, "namespace", instance.Namespace)
			instance.ObjectMeta.Finalizers = helpers.RemoveString(instance.ObjectMeta.Finalizers, finalizer)
			if err := r.Update(context.Background(), instance); err != nil {
				return ctrl.Result{}, err
			}
			datadog.GetInstance().Timing("chaos.controller.cleanup.duration", time.Since(instance.ObjectMeta.DeletionTimestamp.Time), []string{"name:" + instance.Name, "namespace:" + instance.Namespace}, 1)
		}

		// stop the reconcile loop, the finalizing step has finished and the resource should be garbage collected
		return ctrl.Result{}, nil
	}

	// skip the injection if already done
	if instance.Status.IsInjected {
		return ctrl.Result{}, nil
	}

	// select pods eligible for an injection
	pods, err := r.selectPodsForInjection(instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	// start injections
	for _, p := range pods.Items {
		chaosPods := []*corev1.Pod{}

		// get ID of first container
		containerID, err := helpers.GetContainerdID(&p)
		if err != nil {
			return ctrl.Result{}, err
		}

		// generate injection pods specs
		if instance.Spec.NetworkFailure != nil {
			args := instance.Spec.NetworkFailure.GenerateArgs(chaostypes.PodModeInject, instance.UID, containerID)
			chaosPods = append(chaosPods, helpers.GeneratePod(instance.Name, &p, args, chaostypes.PodModeInject, chaostypes.DisruptionKindNetworkFailure))
		}
		if instance.Spec.NetworkLatency != nil {
			args := instance.Spec.NetworkLatency.GenerateArgs(chaostypes.PodModeInject, instance.UID, containerID)
			chaosPods = append(chaosPods, helpers.GeneratePod(instance.Name, &p, args, chaostypes.PodModeInject, chaostypes.DisruptionKindNetworkLatency))
		}
		if instance.Spec.NodeFailure != nil {
			args := instance.Spec.NodeFailure.GenerateArgs(chaostypes.PodModeInject, instance.UID, containerID)
			chaosPods = append(chaosPods, helpers.GeneratePod(instance.Name, &p, args, chaostypes.PodModeInject, chaostypes.DisruptionKindNodeFailure))
		}

		// create injection pods
		for _, chaosPod := range chaosPods {
			// link instance resource and injection pod for garbage collection
			if err := controllerutil.SetControllerReference(instance, chaosPod, r.Scheme); err != nil {
				return ctrl.Result{}, err
			}

			// check if an injection pod already exists for the given (instance, namespace, disruption kind) tuple
			found, err := helpers.GetOwnedPods(r.Client, instance, chaosPod.Labels)
			if err != nil {
				return ctrl.Result{}, err
			}
			if len(found.Items) == 0 {
				r.Log.Info("creating chaos pod", "instance", instance.Name, "namespace", instance.Namespace)
				if err = r.Create(context.Background(), chaosPod); err != nil {
					r.Recorder.Event(instance, "Warning", "Create failed", fmt.Sprintf("Injection pod for disruption \"%s\" failed to be created", instance.Name))
					return ctrl.Result{}, err
				}

				// send metrics and events
				r.Recorder.Event(instance, "Normal", "Created", fmt.Sprintf("Created disruption injection pod for \"%s\"", instance.Name))
				datadog.GetInstance().Incr("chaos.controller.pods.created", []string{"phase:inject", "target_pod:" + p.ObjectMeta.Name, "name:" + instance.Name, "namespace:" + instance.Namespace}, 1)
			}
		}
	}

	// update resource status injection flag
	r.Log.Info("updating injection status flag", "instance", instance.Name, "namespace", instance.Namespace)
	instance.Status.IsInjected = true
	if err = r.Update(context.Background(), instance); err != nil {
		return ctrl.Result{}, err
	}

	// we reach this line only when every injection pods have been created with success
	datadog.GetInstance().Timing("chaos.controller.inject.duration", time.Since(instance.ObjectMeta.CreationTimestamp.Time), []string{"name:" + instance.Name, "namespace:" + instance.Namespace}, 1)

	return ctrl.Result{}, nil
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
		containerID, err := helpers.GetContainerdID(p)
		if err != nil {
			return err
		}

		// generate cleanup pods specs
		if instance.Spec.NetworkFailure != nil {
			args := instance.Spec.NetworkFailure.GenerateArgs(chaostypes.PodModeClean, instance.UID, containerID)
			chaosPods = append(chaosPods, helpers.GeneratePod(instance.Name, p, args, chaostypes.PodModeClean, chaostypes.DisruptionKindNetworkFailure))
		}
		if instance.Spec.NetworkLatency != nil {
			args := instance.Spec.NetworkLatency.GenerateArgs(chaostypes.PodModeClean, instance.UID, containerID)
			chaosPods = append(chaosPods, helpers.GeneratePod(instance.Name, p, args, chaostypes.PodModeClean, chaostypes.DisruptionKindNetworkLatency))
		}

		// create cleanup pods
		for _, chaosPod := range chaosPods {
			// link cleanup pod to instance for garbage collection
			if err := controllerutil.SetControllerReference(instance, chaosPod, r.Scheme); err != nil {
				return err
			}

			// do nothing if cleanup pod already exists
			found := &corev1.Pod{}
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
				return err
			}
			r.Recorder.Event(instance, "Normal", "Created", fmt.Sprintf("Created cleanup pod for disruption \"%s\"", instance.Name))
			datadog.GetInstance().Incr("chaos.controller.pods.created", []string{"phase:cleanup", "target_pod:" + p.ObjectMeta.Name, "name:" + instance.Name, "namespace:" + instance.Namespace}, 1)
		}
	}
	return nil
}

// updateStatusPods will update the instance's status.pods field, if it was not previously set,
// to show the names of pods selected for injection for that instance.
func (r *DisruptionReconciler) updateStatusPods(instance *chaosv1beta1.Disruption, pods *corev1.PodList) error {
	// ignore if we already set the status.pods
	if len(instance.Status.TargetPods) > 0 {
		return nil
	}

	// get all the pod names
	for _, pod := range pods.Items {
		instance.Status.TargetPods = append(instance.Status.TargetPods, pod.Name)
	}

	// update the instance's status
	r.Log.Info("updating instance status with pods selected for injection", "instance", instance.Name, "namespace", instance.Namespace)
	if err := r.Update(context.Background(), instance); err != nil {
		r.Log.Error(err, "failed to update instance status", "disruption", instance.Name, "namespace", instance.Namespace)
		return err
	}

	return nil
}

// selectPodsForInjection will select min(count, all matching pods) random pods from the pods matching the instance label selector
// target pods will only be selected once per instance
// the chosen pods names will be reflected in the intance status
// subsequent calls to this function will always return the same pods as the first call
func (r *DisruptionReconciler) selectPodsForInjection(instance *chaosv1beta1.Disruption) (*corev1.PodList, error) {
	// get pods matching the instance label selector
	allPods, err := helpers.GetMatchingPods(r.Client, instance.Namespace, instance.Spec.Selector)

	// if count has not been specified or is greater than the actual number of matching pods,
	// return all pods matching the label selector
	if instance.Spec.Count == nil || *instance.Spec.Count >= len(allPods.Items) {
		err := r.updateStatusPods(instance, allPods)
		if err != nil {
			return nil, err
		}

		return allPods, nil
	}

	// if we had already selected pods for the instance, only return the already-selected ones
	p := []corev1.Pod{}
	if len(instance.Status.TargetPods) > 0 {
		for _, pod := range allPods.Items {
			if helpers.ContainsString(instance.Status.TargetPods, pod.Name) {
				p = append(p, pod)
			}
		}
		allPods.Items = p
		return allPods, nil
	}

	// otherwise, randomly select pods
	rand.Seed(time.Now().UnixNano())
	numPodsToSelect := int(math.Min(float64(*instance.Spec.Count), float64(*instance.Spec.Count-len(instance.Status.TargetPods))))
	for i := 0; i < numPodsToSelect; i++ {
		// select and add a random pod
		index := rand.Intn(len(allPods.Items))
		chosenPod := allPods.Items[index]
		p = append(p, chosenPod)

		r.Log.Info("Selected random pod", "name", chosenPod.Name, "namespace", chosenPod.Namespace, "disruption", instance.Name, "count", instance.Spec.Count)

		// remove the chosen pod from list of pods from which to select
		allPods.Items[len(allPods.Items)-1], allPods.Items[index] = allPods.Items[index], allPods.Items[len(allPods.Items)-1]
		allPods.Items = allPods.Items[:len(allPods.Items)-1]
	}

	allPods.Items = p

	// update the instance status with the names of the randomly chosen pods
	err = r.updateStatusPods(instance, allPods)
	if err != nil {
		return nil, err
	}

	return allPods, nil
}

// getChaosPods returns all cleanup pods created by the Disruption instance.
func (r *DisruptionReconciler) getChaosPods(instance *chaosv1beta1.Disruption, mode chaostypes.PodMode) ([]corev1.Pod, error) {
	podsInNs := &corev1.PodList{}
	listOptions := &client.ListOptions{
		Namespace: instance.Namespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{
			chaostypes.PodModeLabel: string(mode),
		}),
	}
	err := r.Client.List(context.Background(), podsInNs, listOptions)
	if err != nil {
		return nil, err
	}

	// filter all pods in the same namespace as instance,
	// only returning those owned by the given instance
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
func (r *DisruptionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chaosv1beta1.Disruption{}).
		Complete(r)
}
