// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.
package controllers

import (
	"context"
	"fmt"
	"time"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	// Fetch DisruptionRollout instance
	if err := r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !instance.DeletionTimestamp.IsZero() {
		// Add finalizer here if required
		return ctrl.Result{}, nil
	}

	// Update the DisruptionRollout status based on the presence of the target resource
	// If the target resource has been missing for longer than the TargetResourceMissingThreshold, delete the instance
	targetResourceExists, instanceDeleted, err := r.updateTargetResourcePreviouslyMissing(ctx, instance)
	if err != nil {
		// Log error and requeue if status update or deletion fails
		r.log.Errorw("failed to handle target resource status", "err", err)
		return ctrl.Result{}, err
	}

	if instanceDeleted {
		// Skip reconciliation since the instance has been deleted
		return ctrl.Result{}, nil
	}

	disruptions, err := GetChildDisruptions(ctx, r.Client, r.log, instance.Namespace, DisruptionRolloutNameLabel, instance.Name)
	if err != nil {
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

// updateTargetResourcePreviouslyMissing updates the status when the target resource was previously missing.
// The function returns three values:
// - bool: Indicates whether the target resource is currently found.
// - bool: Indicates whether the disruptionrollout was deleted due to the target resource being missing for more than the expiration duration.
// - error: Represents any error that occurred during the execution of the function.
func (r *DisruptionRolloutReconciler) updateTargetResourcePreviouslyMissing(ctx context.Context, instance *chaosv1beta1.DisruptionRollout) (bool, bool, error) {
	disruptionRolloutDeleted := false
	targetResourceExists, err := CheckTargetResourceExists(ctx, r.Client, &instance.Spec.TargetResource, instance.Namespace)

	if err != nil {
		return targetResourceExists, disruptionRolloutDeleted, err
	}

	if !targetResourceExists {
		r.log.Warnw("target does not exist, this disruption rollout will be deleted if that continues", "error", err)

		if instance.Status.TargetResourcePreviouslyMissing == nil {
			r.log.Warnw("target is missing for the first time, updating status")

			return targetResourceExists, disruptionRolloutDeleted, r.handleTargetResourceFirstMissing(ctx, instance)
		}

		if time.Since(instance.Status.TargetResourcePreviouslyMissing.Time) > TargetResourceMissingThreshold {
			r.log.Errorw("target has been missing for over one day, deleting this schedule",
				"timeMissing", time.Since(instance.Status.TargetResourcePreviouslyMissing.Time))

			disruptionRolloutDeleted = true

			return targetResourceExists, disruptionRolloutDeleted, r.handleTargetResourceMissingPastExpiration(ctx, instance)
		}
	} else if instance.Status.TargetResourcePreviouslyMissing != nil {
		r.log.Infow("target was previously missing, but now present. updating the status accordingly")

		return targetResourceExists, disruptionRolloutDeleted, r.handleTargetResourceNowPresent(ctx, instance)
	}

	return targetResourceExists, disruptionRolloutDeleted, nil
}

// handleTargetResourceFirstMissing handles the scenario when the target resource is missing for the first time.
// It updates the status of the DisruptionRollout instance.
func (r *DisruptionRolloutReconciler) handleTargetResourceFirstMissing(ctx context.Context, instance *chaosv1beta1.DisruptionRollout) error {
	instance.Status.TargetResourcePreviouslyMissing = &metav1.Time{Time: time.Now()}
	if err := r.Client.Status().Update(ctx, instance); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	return nil
}

// handleTargetResourceMissingPastExpiration handles the scenario when the target resource has been missing for more than the expiration period.
// It deletes the DisruptionRollout instance.
func (r *DisruptionRolloutReconciler) handleTargetResourceMissingPastExpiration(ctx context.Context, instance *chaosv1beta1.DisruptionRollout) error {
	if err := r.Client.Delete(ctx, instance); err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	return nil
}

// handleTargetResourceNowPresent handles the scenario when the target resource was previously missing but is now present.
// It updates the status of the DisruptionRollout instance.
func (r *DisruptionRolloutReconciler) handleTargetResourceNowPresent(ctx context.Context, instance *chaosv1beta1.DisruptionRollout) error {
	instance.Status.TargetResourcePreviouslyMissing = nil
	if err := r.Client.Status().Update(ctx, instance); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	return nil
}

// SetupWithManager setups the current reconciler with the given manager
func (r *DisruptionRolloutReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chaosv1beta1.DisruptionRollout{}).
		Complete(r)
}
