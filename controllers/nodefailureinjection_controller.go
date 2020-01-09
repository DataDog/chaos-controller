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
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
	metricPrefix = "chaos.controller.nofi"
)

// NodeFailureInjectionReconciler reconciles a NodeFailureInjection object
type NodeFailureInjectionReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=chaos.datadoghq.com,resources=nodefailureinjections,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=chaos.datadoghq.com,resources=nodefailureinjections/status,verbs=get;update;patch

// Reconcile loop
func (r *NodeFailureInjectionReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("nodefailureinjection", req.NamespacedName)

	instance := &chaosv1beta1.NodeFailureInjection{}
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

	// Fetch the NodeFailureInjection instance
	if err := r.Get(context.Background(), req.NamespacedName, instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Define the quantity
	quantity := 1
	if instance.Spec.Quantity != nil {
		quantity = *instance.Spec.Quantity
	}

	//Initialize nodeNames
	if instance.Status.Injected == nil {
		instance.Status.Injected = []chaosv1beta1.NodeFailureInjectionStatusInjectedEntry{}
	}

	// Create as many pods as needed
	if quantity > len(instance.Status.Injected) {
		// Get matching pods and generate associated nodes
		pods, err := helpers.GetMatchingPods(r.Client, instance.Namespace, instance.Spec.Selector)
		if err != nil {
			r.Log.Error(err, "Failed to get pods matching the resource label selector", "instance", instance.Name)
			return ctrl.Result{}, err
		}
		for _, p := range helpers.PickRandomPods(uint(quantity), pods.Items) {
			args := []string{
				"node-failure",
				"inject",
				"--uid",
				string(instance.ObjectMeta.UID),
			}
			if instance.Spec.Shutdown {
				args = append(args, "--shutdown")
			}
			pod := helpers.GeneratePod(instance.Name+"-"+p.Name, &p, args, chaostypes.PodModeInject)
			if err := controllerutil.SetControllerReference(instance, pod, r.Scheme); err != nil {
				return ctrl.Result{}, err
			}

			// Check if the pod already exists
			found := &corev1.Pod{}
			err = r.Get(context.Background(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
			if err != nil && errors.IsNotFound(err) {
				// Ensure the node hasn't been injected before (possibly by another pod running on the same node as the targeted pod)
				found := false
				for _, injected := range instance.Status.Injected {
					if injected.Node == p.Spec.NodeName {
						found = true
						break
					}
				}
				if found {
					continue
				}

				// Set instance status before creating the pod, otherwise the node name might disappear because of the failure
				statusInjectedEntry := chaosv1beta1.NodeFailureInjectionStatusInjectedEntry{
					Node: p.Spec.NodeName,
					Pod:  p.Name,
				}
				instance.Status.Injected = append(instance.Status.Injected, statusInjectedEntry)

				// Create the injection pod
				r.Log.Info("Creating node failure injection pod", "namespace", pod.Namespace, "name", pod.Name)
				if err = r.Create(context.Background(), pod); err != nil {
					r.Log.Error(err, "Failed to create injection pod", "instance", instance.Name)
					r.Recorder.Event(instance, "Warning", "Create failed", fmt.Sprintf("Failure injection pod for nodefailureinjection \"%s\" failed to be created", instance.Name))
					return ctrl.Result{}, err
				}
				r.Recorder.Event(instance, "Normal", "Created", fmt.Sprintf("Created failure injection pod for nodefailureinjection \"%s\"", instance.Name))
				datadog.EventWithTags("New Injected NodeFailureInjection Pod", fmt.Sprintf("Created failure injection pod for nodefailureinjection \"%s\"", instance.Name), p.Spec.NodeName, []string{"phase:inject", "target_pod:" + statusInjectedEntry.Pod, "target_node:" + statusInjectedEntry.Node, "name:" + instance.Name, "namespace:" + instance.Namespace})
				datadog.GetInstance().Incr(metricPrefix+".pods.created", []string{"phase:inject", "target_pod:" + statusInjectedEntry.Pod, "target_node:" + statusInjectedEntry.Node, "name:" + instance.Name, "namespace:" + instance.Namespace}, 1)

				// Update the instance
				if err = r.Update(context.Background(), instance); err != nil {
					r.Log.Error(err, "Failed to update instance status", "instance", instance.Name)
					return ctrl.Result{}, err
				}
			} else if err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager setups the current reconciler with the given manager
func (r *NodeFailureInjectionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chaosv1beta1.NodeFailureInjection{}).
		Complete(r)
}
