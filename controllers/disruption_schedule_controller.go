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

type DisruptionScheduleReconciler struct {
	Client  client.Client
	Scheme  *runtime.Scheme
	BaseLog *zap.SugaredLogger
}

// +kubebuilder:rbac:groups=chaos.datadoghq.com,resources=disruptionschedules,verbs=list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=chaos.datadoghq.com,resources=disruptionschedules/status,verbs=update;patch
// +kubebuilder:rbac:groups=chaos.datadoghq.com,resources=disruptionschedules/finalizers,verbs=update
func (r *DisruptionScheduleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	r.BaseLog.Info("RECONCILING")

	return ctrl.Result{}, nil
}

// SetupWithManager setups the current reconciler with the given manager
func (r *DisruptionScheduleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chaosv1beta1.DisruptionSchedule{}).
		Complete(r)
}
