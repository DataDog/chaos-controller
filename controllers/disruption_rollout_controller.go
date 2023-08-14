// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.
package controllers

import (
	"context"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DisruptionRolloutReconciler struct {
	Client  client.Client
	Scheme  *runtime.Scheme
	BaseLog *zap.SugaredLogger
	log     *zap.SugaredLogger
}

func (r *DisruptionRolloutReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	r.log = r.BaseLog.With("disruptionRolloutNamespace", req.Namespace, "disruptionRolloutName", req.Name)
	r.log.Info("Reconciling DisruptionRollout")

	instance := &chaosv1beta1.DisruptionRollout{}

	// Fetch DisruptionCron instance
	if err := r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	r.log.Infow("Object", "DisruptionRollout", instance)

	return ctrl.Result{}, nil
}

// SetupWithManager setups the current reconciler with the given manager
func (r *DisruptionRolloutReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chaosv1beta1.DisruptionRollout{}).
		Complete(r)
}
