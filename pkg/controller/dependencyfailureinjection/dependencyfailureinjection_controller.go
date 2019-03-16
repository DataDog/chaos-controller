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
	"reflect"

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

	// Define the desired pod object
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name + "-pod",
			Namespace: instance.Namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx",
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
		log.Info("Creating chaos pod", "namespace", pod.Namespace, "name", pod.Name)
		err = r.Create(context.TODO(), pod)
		return reconcile.Result{}, err
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Update the found object and write the result back if there are any changes
	if !reflect.DeepEqual(pod.Spec, found.Spec) {
		found.Spec = pod.Spec
		log.Info("Updating chaos pod", "namespace", pod.Namespace, "name", pod.Name)
		err = r.Update(context.TODO(), found)
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{}, nil
}
