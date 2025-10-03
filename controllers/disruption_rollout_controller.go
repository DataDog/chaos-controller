// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package controllers

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	cLog "github.com/DataDog/chaos-controller/log"
	"github.com/DataDog/chaos-controller/o11y/metrics"
	tagutil "github.com/DataDog/chaos-controller/o11y/tags"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var DisruptionRolloutTags = []string{}

type DisruptionRolloutReconciler struct {
	Client                         client.Client
	Scheme                         *runtime.Scheme
	BaseLog                        *zap.SugaredLogger
	log                            *zap.SugaredLogger
	MetricsSink                    metrics.Sink
	TargetResourceMissingThreshold time.Duration
}

func (r *DisruptionRolloutReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	r.log = r.BaseLog.With("disruptionRolloutNamespace", req.Namespace, "disruptionRolloutName", req.Name)
	r.log.Info("Reconciling DisruptionRollout")

	instance := &chaosv1beta1.DisruptionRollout{}
	randSource := rand.New(rand.NewSource(time.Now().UnixNano()))

	// reconcile metrics
	r.handleMetricSinkError(r.MetricsSink.MetricReconcile())

	defer func(tsStart time.Time) {
		tags := []string{}
		if instance.Name != "" {
			tags = append(tags,
				tagutil.FormatTag(cLog.DisruptionRolloutNameKey, instance.Name),
				tagutil.FormatTag(cLog.DisruptionRolloutNamespaceKey, instance.Namespace),
			)
		}

		r.handleMetricSinkError(r.MetricsSink.MetricReconcileDuration(time.Since(tsStart), tags))
	}(time.Now())

	// Fetch DisruptionRollout instance
	if err := r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	DisruptionRolloutTags = []string{
		tagutil.FormatTag(cLog.DisruptionRolloutNameKey, instance.Name),
		tagutil.FormatTag(cLog.DisruptionRolloutNamespaceKey, instance.Namespace),
		tagutil.FormatTag(cLog.TargetNameKey, instance.Spec.TargetResource.Name),
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

	// Update the DisruptionRollout status with the time when the last disruption was successfully scheduled
	if err := r.updateLastScheduleTime(ctx, instance, disruptions); err != nil {
		r.log.Errorw("unable to update LastScheduleTime of DisruptionCron status", "err", err)
		return ctrl.Result{}, err
	}

	// Calculate next requeue time
	requeueAfter := time.Duration(randSource.Intn(5)+15) * time.Second //nolint:gosec
	requeueTime := requeueAfter.Round(time.Second)
	scheduledResult := ctrl.Result{RequeueAfter: requeueAfter}

	// Run a new disruption if the following conditions are met:
	// 1. The target resource is available
	// 2. The target resource has been updated
	// 3. The target resource update has not been tested
	// 4. It's not blocked by another disruption already running
	// 5. It's not past the deadline
	if !targetResourceExists {
		r.log.Infow(fmt.Sprintf("target resource is missing, scheduling next check in %s", requeueTime))
		return scheduledResult, nil
	}

	if !r.targetResourceUpdated(&instance.Status) {
		r.log.Infow("target resource hasn't been modified yet, sleeping")
		return ctrl.Result{}, nil
	}

	if instance.Status.LastContainerChangeTime.Before(instance.Status.LastScheduleTime) || instance.Status.LastContainerChangeTime.Equal(instance.Status.LastScheduleTime) {
		r.log.Debugw("target resource update has already been tested, sleeping",
			"LastContainerChangeTime", instance.Status.LastContainerChangeTime,
			"LastScheduleTime", instance.Status.LastScheduleTime)

		return ctrl.Result{}, nil
	}

	if len(disruptions.Items) > 0 {
		r.log.Infow(fmt.Sprintf("cannot start a new disruption as a prior one is still running, scheduling next check in %s", requeueTime), "numActiveDisruptions", len(disruptions.Items))
		return scheduledResult, nil
	}

	tooLate := false
	if instance.Spec.DelayedStartTolerance.Duration() > 0 && !instance.Status.LastContainerChangeTime.IsZero() {
		tooLate = instance.Status.LastContainerChangeTime.Add(instance.Spec.DelayedStartTolerance.Duration()).Before(time.Now())
	}

	if tooLate {
		r.handleMetricSinkError(r.MetricsSink.MetricTooLate(DisruptionRolloutTags))
		r.log.Infow("missed schedule to start a disruption, sleeping",
			"LastContainerChangeTime", instance.Status.LastContainerChangeTime,
			"DelayedStartTolerance", instance.Spec.DelayedStartTolerance)

		return ctrl.Result{}, nil
	}

	// Create disruption
	scheduledTime := time.Now()
	disruption, err := CreateDisruptionFromTemplate(ctx, r.Client, r.Scheme, instance, &instance.Spec.TargetResource, &instance.Spec.DisruptionTemplate, scheduledTime, r.log)

	if err != nil {
		r.log.Warnw("unable to construct disruption from template", "err", err)
		return scheduledResult, nil
	}

	if err := r.Client.Create(ctx, disruption); err != nil {
		r.log.Warnw("unable to create Disruption for DisruptionRollout", "disruption", disruption, "err", err)
		return ctrl.Result{}, err
	}

	r.handleMetricSinkError(r.MetricsSink.MetricDisruptionScheduled(append(DisruptionRolloutTags, tagutil.FormatTag(cLog.DisruptionNameKey, disruption.Name))))

	r.log.Infow("created Disruption for DisruptionRollout run", cLog.DisruptionNameKey, disruption.Name)

	// ------------------------------------------------------------------ //
	// If this process restarts at this point (after posting a disruption, but
	// before updating the status), we might try to start the disruption again
	// the next time. To prevent this, we use the same disruption name for every
	// execution, acting as a lock to prevent creating the disruption twice.

	// Add the start time of the just initiated disruption to the status
	instance.Status.LastScheduleTime = &metav1.Time{Time: scheduledTime}
	if err := r.Client.Status().Update(ctx, instance); err != nil {
		r.log.Warnw("unable to update LastScheduleTime of DisruptionCron status", "err", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// updateLastScheduleTime updates the LastScheduleTime in the status of a DisruptionRollout instance
// based on the most recent schedule time among the given disruptions.
func (r *DisruptionRolloutReconciler) updateLastScheduleTime(ctx context.Context, instance *chaosv1beta1.DisruptionRollout, disruptions *chaosv1beta1.DisruptionList) error {
	mostRecentScheduleTime := GetMostRecentScheduleTime(r.log, disruptions) // find the last run so we can update the status
	if !mostRecentScheduleTime.IsZero() {
		instance.Status.LastScheduleTime = &metav1.Time{Time: mostRecentScheduleTime}
		return r.Client.Status().Update(ctx, instance)
	}

	return nil // No need to update if mostRecentScheduleTime is nil
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

		if time.Since(instance.Status.TargetResourcePreviouslyMissing.Time) > r.TargetResourceMissingThreshold {
			r.log.Errorw("target has been missing for over one day, deleting this schedule",
				"timeMissing", time.Since(instance.Status.TargetResourcePreviouslyMissing.Time))

			disruptionRolloutDeleted = true

			return targetResourceExists, disruptionRolloutDeleted, r.handleTargetResourceMissingPastExpiration(ctx, instance)
		}

		r.handleMetricSinkError(r.MetricsSink.MetricTargetMissing(time.Since(instance.Status.TargetResourcePreviouslyMissing.Time), DisruptionRolloutTags))
	} else if instance.Status.TargetResourcePreviouslyMissing != nil {
		r.log.Infow("target was previously missing, but now present. updating the status accordingly")
		r.handleMetricSinkError(r.MetricsSink.MetricMissingTargetFound(DisruptionRolloutTags))

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

	r.handleMetricSinkError(r.MetricsSink.MetricMissingTargetDeleted(DisruptionRolloutTags))

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

// handleMetricSinkError logs the given metric sink error if it is not nil
func (r *DisruptionRolloutReconciler) handleMetricSinkError(err error) {
	if err != nil {
		r.log.Errorw("error sending a metric", "error", err)
	}
}

// targetResourceUpdated checks whether the target resource has been updated or not.
func (r *DisruptionRolloutReconciler) targetResourceUpdated(status *chaosv1beta1.DisruptionRolloutStatus) bool {
	if status == nil {
		return false
	}

	if status.LatestInitContainersHash == nil &&
		status.LatestContainersHash == nil &&
		status.LastContainerChangeTime == nil {
		return false
	}

	return true
}

// SetupWithManager setups the current reconciler with the given manager
func (r *DisruptionRolloutReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chaosv1beta1.DisruptionRollout{}).
		Complete(r)
}
