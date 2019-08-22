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

package nodefailureinjection

import (
	"context"
	"fmt"
	"time"

	chaosv1beta1 "github.com/DataDog/chaos-fi-controller/pkg/apis/chaos/v1beta1"
	"github.com/DataDog/chaos-fi-controller/pkg/datadog"
	"github.com/DataDog/chaos-fi-controller/pkg/helpers"
	chaostypes "github.com/DataDog/chaos-fi-controller/pkg/types"
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
	metricPrefix = "chaos.controller.nofi"
)

var log = logf.Log.WithName("controller")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new NodeFailureInjection Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileNodeFailureInjection{
		Client:   mgr.GetClient(),
		scheme:   mgr.GetScheme(),
		recorder: mgr.GetRecorder("nodefailureinjection-controller"),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("nodefailureinjection-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to NodeFailureInjection
	err = c.Watch(&source.Kind{Type: &chaosv1beta1.NodeFailureInjection{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &chaosv1beta1.NodeFailureInjection{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileNodeFailureInjection{}

// ReconcileNodeFailureInjection reconciles a NodeFailureInjection object
type ReconcileNodeFailureInjection struct {
	client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

// Reconcile reads that state of the cluster for a NodeFailureInjection object and makes changes based on the state read
// and what is in the NodeFailureInjection.Spec
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=,resources=pods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=chaos.datadoghq.com,resources=nodefailureinjections,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=chaos.datadoghq.com,resources=nodefailureinjections/status,verbs=get;update;patch
func (r *ReconcileNodeFailureInjection) Reconcile(request reconcile.Request) (reconcile.Result, error) {
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
	err := r.Get(context.Background(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
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
			log.Error(err, "Failed to get pods matching the resource label selector", "instance", instance.Name)
			return reconcile.Result{}, err
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
			if err := controllerutil.SetControllerReference(instance, pod, r.scheme); err != nil {
				return reconcile.Result{}, err
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
				log.Info("Creating node failure injection pod", "namespace", pod.Namespace, "name", pod.Name)
				err = r.Create(context.Background(), pod)
				if err != nil {
					log.Error(err, "Failed to create injection pod", "instance", instance.Name)
					r.recorder.Event(instance, "Warning", "Create failed", fmt.Sprintf("Failure injection pod for nodefailureinjection \"%s\" failed to be created", instance.Name))
					return reconcile.Result{}, err
				}
				r.recorder.Event(instance, "Normal", "Created", fmt.Sprintf("Created failure injection pod for nodefailureinjection \"%s\"", instance.Name))
				datadog.GetInstance().Incr(metricPrefix+".pods.created", []string{"phase:inject", "target_pod:" + statusInjectedEntry.Pod, "target_node:" + statusInjectedEntry.Node, "name:" + instance.Name, "namespace:" + instance.Namespace}, 1)

				// Update the instance
				err = r.Update(context.Background(), instance)
				if err != nil {
					log.Error(err, "Failed to update instance status", "instance", instance.Name)
					return reconcile.Result{}, err
				}
			} else if err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	return reconcile.Result{}, nil
}
