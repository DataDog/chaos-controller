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
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DisruptionCronReconciler struct {
	Client  client.Client
	Scheme  *runtime.Scheme
	BaseLog *zap.SugaredLogger
	log     *zap.SugaredLogger
}

const (
	ScheduledAtAnnotation          = chaosv1beta1.GroupName + "/scheduled-at"
	DisruptionCronNameLabel        = chaosv1beta1.GroupName + "/disruption-cron-name"
	TargetResourceMissingThreshold = time.Hour * 24
)

func (r *DisruptionCronReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	r.log = r.BaseLog.With("disruptionCronNamespace", req.Namespace, "disruptionCronName", req.Name)
	r.log.Info("Reconciling DisruptionCron")

	instance := &chaosv1beta1.DisruptionCron{}

	// Fetch DisruptionCron instance
	if err := r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

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

	disruptions, err := r.getChildDisruptions(ctx, instance)
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
	if instance.Spec.StartingDeadlineSeconds != nil {
		tooLate = missedRun.Add(time.Duration(*instance.Spec.StartingDeadlineSeconds) * time.Second).Before(time.Now())
	}

	if tooLate {
		r.log.Infow(fmt.Sprintf("missed schedule to start a disruption at %s, scheduling next check in %s", missedRun, requeueTime))
		return scheduledResult, nil
	}

	r.log.Infow("processing current run", "currentRun", missedRun.Format(time.UnixDate))

	// Create disruption for current run
	disruption, err := r.getDisruptionFromTemplate(ctx, instance, missedRun)
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

	if err := r.Client.Status().Update(ctx, instance); err != nil {
		r.log.Warnw("unable to update LastScheduleTime of DisruptionCron status", "err", err)
		return ctrl.Result{}, err
	}

	return scheduledResult, nil
}

// getChildDisruptions fetches all disruptions associated with the given DisruptionCron instance.
// Most of the time, this will return an empty list as disruptions are typically short-lived objects.
func (r *DisruptionCronReconciler) getChildDisruptions(ctx context.Context, instance *chaosv1beta1.DisruptionCron) (*chaosv1beta1.DisruptionList, error) {
	disruptions := &chaosv1beta1.DisruptionList{}
	labelSelector := labels.SelectorFromSet(labels.Set{DisruptionCronNameLabel: instance.Name})

	if err := r.Client.List(ctx, disruptions, client.InNamespace(instance.Namespace), &client.ListOptions{LabelSelector: labelSelector}); err != nil {
		r.log.Errorw("unable to list Disruptions", "err", err)
		return disruptions, err
	}

	return disruptions, nil
}

// updateLastScheduleTime updates the LastScheduleTime in the status of a DisruptionCron instance
// based on the most recent schedule time among the given disruptions.
func (r *DisruptionCronReconciler) updateLastScheduleTime(ctx context.Context, instance *chaosv1beta1.DisruptionCron, disruptions *chaosv1beta1.DisruptionList) error {
	mostRecentScheduleTime := r.getMostRecentScheduleTime(disruptions) // find the last run so we can update the status
	if mostRecentScheduleTime != nil {
		instance.Status.LastScheduleTime = &metav1.Time{Time: *mostRecentScheduleTime}
		return r.Client.Status().Update(ctx, instance)
	}

	return nil // No need to update if mostRecentScheduleTime is nil
}

// getMostRecentScheduleTime returns the most recent scheduled time from a list of disruptions.
func (r *DisruptionCronReconciler) getMostRecentScheduleTime(disruptions *chaosv1beta1.DisruptionList) *time.Time {
	var mostRecentScheduleTime *time.Time

	for _, disruption := range disruptions.Items {
		scheduledTimeForDisruption, err := r.getScheduledTimeForDisruption(&disruption)
		if err != nil {
			r.log.Errorw("unable to parse schedule time for child disruption", "err", err, "disruption", disruption.Name)
			continue
		}

		if scheduledTimeForDisruption != nil {
			if mostRecentScheduleTime == nil {
				mostRecentScheduleTime = scheduledTimeForDisruption
			} else if mostRecentScheduleTime.Before(*scheduledTimeForDisruption) {
				mostRecentScheduleTime = scheduledTimeForDisruption
			}
		}
	}

	return mostRecentScheduleTime
}

// getScheduledTimeForDisruption returns the scheduled time for a particular disruption
func (r *DisruptionCronReconciler) getScheduledTimeForDisruption(disruption *chaosv1beta1.Disruption) (*time.Time, error) {
	timeRaw := disruption.Annotations[ScheduledAtAnnotation]
	if len(timeRaw) == 0 {
		return nil, nil
	}

	timeParsed, err := time.Parse(time.RFC3339, timeRaw)
	if err != nil {
		return nil, err
	}

	return &timeParsed, nil
}

// getTargetResource retrieves the target resource specified in the DisruptionCron instance.
// It returns two values:
// - client.Object: Represents the target resource (Deployment or StatefulSet).
// - error: Any error encountered during retrieval.
func (r *DisruptionCronReconciler) getTargetResource(ctx context.Context, instance *chaosv1beta1.DisruptionCron) (client.Object, error) {
	var targetObj client.Object

	switch instance.Spec.TargetResource.Kind {
	case "deployment":
		targetObj = &appsv1.Deployment{}
	case "statefulset":
		targetObj = &appsv1.StatefulSet{}
	}

	err := r.Client.Get(ctx, types.NamespacedName{
		Name:      instance.Spec.TargetResource.Name,
		Namespace: instance.Namespace,
	}, targetObj)

	return targetObj, err
}

// checkTargetResourceExists checks whether the target resource exists.
// It returns two values:
// - bool: Indicates whether the target resource is currently found.
// - error: Represents any error that occurred during the execution of the function.
func (r *DisruptionCronReconciler) checkTargetResourceExists(ctx context.Context, instance *chaosv1beta1.DisruptionCron) (bool, error) {
	_, err := r.getTargetResource(ctx, instance)

	if errors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

// updateTargetResourcePreviouslyMissing updates the status when the target resource was previously missing.
// The function returns three values:
// - bool: Indicates whether the target resource is currently found.
// - bool: Indicates whether the disruptioncron was deleted due to the target resource being missing for more than the expiration duration.
// - error: Represents any error that occurred during the execution of the function.
func (r *DisruptionCronReconciler) updateTargetResourcePreviouslyMissing(ctx context.Context, instance *chaosv1beta1.DisruptionCron) (bool, bool, error) {
	disruptionCronDeleted := false
	targetResourceExists, err := r.checkTargetResourceExists(ctx, instance)

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

	if instance.Spec.StartingDeadlineSeconds != nil {
		// controller is not going to schedule anything below this point
		schedulingDeadline := now.Add(-time.Second * time.Duration(*instance.Spec.StartingDeadlineSeconds))

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

// getSelectors retrieves the labels of the target resource specified in the DisruptionCron instance.
// The function returns two values:
// - labels.Set: A set of labels of the target resource which will be used as the selectors for a Disruption.
// - error: An error if the target resource or labels retrieval fails.
func (r *DisruptionCronReconciler) getSelectors(ctx context.Context, instance *chaosv1beta1.DisruptionCron) (labels.Set, error) {
	targetObj, err := r.getTargetResource(ctx, instance)
	if err != nil {
		return nil, err
	}

	labels := targetObj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	return labels, nil
}

// getDisruptionFromTemplate creates a Disruption object based on a DisruptionCron instance and a scheduledTime.
// The selectors of the Disruption object are overwritten with the selectors of the target resource.
// The function returns two values:
// - *chaosv1beta1.Disruption: A pointer to the created Disruption object. The object is not created in the Kubernetes cluster.
// - error: An error if any operation fails, such as when selectors of the target resource retrieval fails.
func (r *DisruptionCronReconciler) getDisruptionFromTemplate(ctx context.Context, instance *chaosv1beta1.DisruptionCron, scheduledTime time.Time) (*chaosv1beta1.Disruption, error) {
	// Disruption names are deterministic for a given nominal start time to avoid creating the same disruption more than once
	name := fmt.Sprintf("disruption-cron-%s", instance.Name)

	disruption := &chaosv1beta1.Disruption{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   instance.Namespace,
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
		},
		Spec: *instance.Spec.DisruptionTemplate.DeepCopy(),
	}

	for k, v := range instance.Annotations {
		disruption.Annotations[k] = v
	}

	disruption.Annotations[ScheduledAtAnnotation] = scheduledTime.Format(time.RFC3339)

	disruption.Labels[DisruptionCronNameLabel] = instance.Name

	if err := r.overwriteDisruptionSelectors(ctx, instance, disruption); err != nil {
		return nil, err
	}

	if err := ctrl.SetControllerReference(instance, disruption, r.Scheme); err != nil {
		return nil, err
	}

	return disruption, nil
}

// overwriteDisruptionSelectors replaces the Disruption's selectors with the ones from the target resource
func (r *DisruptionCronReconciler) overwriteDisruptionSelectors(ctx context.Context, instance *chaosv1beta1.DisruptionCron, disruption *chaosv1beta1.Disruption) error {
	// Get selectors from target resource
	selectors, err := r.getSelectors(ctx, instance)
	if err != nil {
		return err
	}

	if disruption.Spec.Selector == nil {
		disruption.Spec.Selector = make(map[string]string)
	}

	for k, v := range selectors {
		disruption.Spec.Selector[k] = v
	}

	return nil
}

// SetupWithManager setups the current reconciler with the given manager
func (r *DisruptionCronReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chaosv1beta1.DisruptionCron{}).
		Complete(r)
}
