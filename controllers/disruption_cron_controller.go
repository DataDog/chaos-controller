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
	"github.com/robfig/cron"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DisruptionCronReconciler struct {
	Client  client.Client
	Scheme  *runtime.Scheme
	BaseLog *zap.SugaredLogger
	log     *zap.SugaredLogger
}

func (r *DisruptionCronReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	r.log = r.BaseLog.With("disruptionCronNamespace", req.Namespace, "disruptionCronName", req.Name)
	r.log.Info("Reconciling DisruptionCron")

	instance := &chaosv1beta1.DisruptionCron{}

	// Fetch DisruptionCron instance
	if err := r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	r.log.Infow("fetched last known history", "history", instance.Status.History)

	if !instance.DeletionTimestamp.IsZero() {
		// Add finalizer here if required
		return ctrl.Result{}, nil
	}

	// Update the DisruptionCron status based on the presence of the target resource
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

	disruptions, err := GetChildDisruptions(ctx, r.Client, r.log, instance.Namespace, DisruptionCronNameLabel, instance.Name)
	if err != nil {
		return ctrl.Result{}, nil
	}

	// Update the DisruptionCron status with the time when the last disruption was successfully scheduled
	if err := r.updateLastScheduleTime(ctx, instance, disruptions); err != nil {
		r.log.Errorw("unable to update LastScheduleTime of DisruptionCron status", "err", err)
		return ctrl.Result{}, err
	}

	// Get next scheduled run or time of unprocessed run
	// If multiple unmet start times exist, start the last one
	missedRun, nextRun, err := r.getNextSchedule(instance, time.Now())
	if err != nil {
		r.log.Errorw("unable to figure out DisruptionCron schedule", "err", err)
		// Don't requeue until schedule update is received
		return ctrl.Result{}, nil
	}

	// Calculate next requeue time
	scheduledResult := ctrl.Result{RequeueAfter: time.Until(nextRun)}
	requeueTime := scheduledResult.RequeueAfter.Round(time.Second)

	r.log.Infow("calculated next scheduled run", "nextRun", nextRun.Format(time.UnixDate), "now", time.Now().Format(time.UnixDate))

	// Run a new disruption if the following conditions are met:
	// 1. It's on schedule
	// 2. The target resource is available
	// 3. It's not blocked by another disruption already running
	// 4. It's not past the deadline
	if missedRun.IsZero() {
		r.log.Infow(fmt.Sprintf("no missed runs detected, scheduling next check in %s", requeueTime))
		return scheduledResult, nil
	}

	if !targetResourceExists {
		r.log.Infow(fmt.Sprintf("target resource is missing, scheduling next check in %s", requeueTime))
		return scheduledResult, nil
	}

	if len(disruptions.Items) > 0 {
		r.log.Infow(fmt.Sprintf("cannot start a new disruption as a prior one is still running, scheduling next check in %s", requeueTime), "numActiveDisruptions", len(disruptions.Items))
		return scheduledResult, nil
	}

	tooLate := false
	if instance.Spec.DelayedStartTolerance.Duration() > 0 {
		tooLate = missedRun.Add(instance.Spec.DelayedStartTolerance.Duration()).Before(time.Now())
	}

	if tooLate {
		r.log.Infow(fmt.Sprintf("missed schedule to start a disruption at %s, scheduling next check in %s", missedRun, requeueTime))
		return scheduledResult, nil
	}

	r.log.Infow("processing current run", "currentRun", missedRun.Format(time.UnixDate))

	// Create disruption for current run
	disruption, err := CreateDisruptionFromTemplate(ctx, r.Client, r.Scheme, instance, &instance.Spec.TargetResource, &instance.Spec.DisruptionTemplate, missedRun)

	if err != nil {
		r.log.Warnw("unable to construct disruption from template", "err", err)
		// Don't requeue until update to the spec is received
		return scheduledResult, nil
	}

	if err := r.Client.Create(ctx, disruption); err != nil {
		r.log.Warnw("unable to create Disruption for DisruptionCron", "disruption", disruption, "err", err)
		return ctrl.Result{}, err
	}

	r.log.Infow("created Disruption for DisruptionCron run", "disruptionName", disruption.Name)

	// ------------------------------------------------------------------ //
	// If this process restarts at this point (after posting a disruption, but
	// before updating the status), we might try to start the disruption again
	// the next time. To prevent this, we use the same disruption name for every
	// execution, acting as a lock to prevent creating the disruption twice.

	// Add the start time of the just initiated disruption to the status
	instance.Status.LastScheduleTime = &metav1.Time{Time: missedRun}
	instance.Status.History = append(instance.Status.History, chaosv1beta1.DisruptionRun{
		Name:      instance.ObjectMeta.Name,
		Kind:      instance.TypeMeta.Kind,
		CreatedAt: *instance.Status.LastScheduleTime,
	})
	if len(instance.Status.History) > chaosv1beta1.MaxHistoryLen {
		instance.Status.History = instance.Status.History[1:]
	}

	r.log.Debugw("updating instance Status lastScheduleTime and history",
		"lastScheduleTime", instance.Status.LastScheduleTime, "history", instance.Status.History)

	if err := r.Client.Status().Update(ctx, instance); err != nil {
		r.log.Warnw("unable to update LastScheduleTime of DisruptionCron status", "err", err)
		return ctrl.Result{}, err
	}

	return scheduledResult, nil
}

// updateLastScheduleTime updates the LastScheduleTime in the status of a DisruptionCron instance
// based on the most recent schedule time among the given disruptions.
func (r *DisruptionCronReconciler) updateLastScheduleTime(ctx context.Context, instance *chaosv1beta1.DisruptionCron, disruptions *chaosv1beta1.DisruptionList) error {
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
// - bool: Indicates whether the disruptioncron was deleted due to the target resource being missing for more than the expiration duration.
// - error: Represents any error that occurred during the execution of the function.
func (r *DisruptionCronReconciler) updateTargetResourcePreviouslyMissing(ctx context.Context, instance *chaosv1beta1.DisruptionCron) (bool, bool, error) {
	disruptionCronDeleted := false
	targetResourceExists, err := CheckTargetResourceExists(ctx, r.Client, &instance.Spec.TargetResource, instance.Namespace)

	if err != nil {
		return targetResourceExists, disruptionCronDeleted, err
	}

	if !targetResourceExists {
		r.log.Warnw("target does not exist, this schedule will be deleted if that continues", "error", err)

		if instance.Status.TargetResourcePreviouslyMissing == nil {
			r.log.Warnw("target is missing for the first time, updating status")

			return targetResourceExists, disruptionCronDeleted, r.handleTargetResourceFirstMissing(ctx, instance)
		}

		if time.Since(instance.Status.TargetResourcePreviouslyMissing.Time) > TargetResourceMissingThreshold {
			r.log.Errorw("target has been missing for over one day, deleting this schedule",
				"timeMissing", time.Since(instance.Status.TargetResourcePreviouslyMissing.Time))

			disruptionCronDeleted = true

			return targetResourceExists, disruptionCronDeleted, r.handleTargetResourceMissingPastExpiration(ctx, instance)
		}
	} else if instance.Status.TargetResourcePreviouslyMissing != nil {
		r.log.Infow("target was previously missing, but now present. updating the status accordingly")

		return targetResourceExists, disruptionCronDeleted, r.handleTargetResourceNowPresent(ctx, instance)
	}

	return targetResourceExists, disruptionCronDeleted, nil
}

// handleTargetResourceFirstMissing handles the scenario when the target resource is missing for the first time.
// It updates the status of the DisruptionCron instance.
func (r *DisruptionCronReconciler) handleTargetResourceFirstMissing(ctx context.Context, instance *chaosv1beta1.DisruptionCron) error {
	instance.Status.TargetResourcePreviouslyMissing = &metav1.Time{Time: time.Now()}
	if err := r.Client.Status().Update(ctx, instance); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	return nil
}

// handleTargetResourceMissingPastExpiration handles the scenario when the target resource has been missing for more than the expiration period.
// It deletes the DisruptionCron instance.
func (r *DisruptionCronReconciler) handleTargetResourceMissingPastExpiration(ctx context.Context, instance *chaosv1beta1.DisruptionCron) error {
	if err := r.Client.Delete(ctx, instance); err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	return nil
}

// handleTargetResourceNowPresent handles the scenario when the target resource was previously missing but is now present.
// It updates the status of the DisruptionCron instance.
func (r *DisruptionCronReconciler) handleTargetResourceNowPresent(ctx context.Context, instance *chaosv1beta1.DisruptionCron) error {
	instance.Status.TargetResourcePreviouslyMissing = nil
	if err := r.Client.Status().Update(ctx, instance); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	return nil
}

// getNextSchedule calculates the next scheduled time for a DisruptionCron instance based on its cron schedule and the current time.
// It returns the last missed schedule time, the next scheduled time, and any error encountered during parsing the schedule.
func (r *DisruptionCronReconciler) getNextSchedule(instance *chaosv1beta1.DisruptionCron, now time.Time) (lastMissed time.Time, next time.Time, err error) {
	starts := 0
	sched, err := cron.ParseStandard(instance.Spec.Schedule)

	if err != nil {
		r.log.Errorw("Unparseable schedule", "schedule", instance.Spec.Schedule, "err", err)
		return time.Time{}, time.Time{}, err
	}

	var earliestTime time.Time
	if instance.Status.LastScheduleTime != nil {
		earliestTime = instance.Status.LastScheduleTime.Time
	} else {
		earliestTime = instance.ObjectMeta.CreationTimestamp.Time
	}

	if instance.Spec.DelayedStartTolerance.Duration() > 0 {
		// controller is not going to schedule anything below this point
		schedulingDeadline := now.Add(-instance.Spec.DelayedStartTolerance.Duration())

		if schedulingDeadline.After(earliestTime) {
			earliestTime = schedulingDeadline
		}
	}

	if earliestTime.After(now) {
		return time.Time{}, sched.Next(now), nil
	}

	for t := sched.Next(earliestTime); !t.After(now); t = sched.Next(t) {
		lastMissed = t
		starts++

		if starts > 100 {
			// We can't get the most recent times so just return an empty slice
			return time.Time{}, time.Time{}, fmt.Errorf("too many missed start times (> 100)")
		}
	}

	return lastMissed, sched.Next(now), nil
}

// SetupWithManager setups the current reconciler with the given manager
func (r *DisruptionCronReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chaosv1beta1.DisruptionCron{}).
		Complete(r)
}
