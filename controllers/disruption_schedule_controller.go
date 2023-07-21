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

type DisruptionScheduleReconciler struct {
	Client  client.Client
	Scheme  *runtime.Scheme
	BaseLog *zap.SugaredLogger
	log     *zap.SugaredLogger
}

const (
	ScheduledAtAnnotation          = chaosv1beta1.GroupName + "/scheduled-at"
	DisruptionScheduleNameLabel    = chaosv1beta1.GroupName + "/disruption-schedule-name"
	TargetResourceMissingThreshold = time.Hour * 24
)

func (r *DisruptionScheduleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	r.log = r.BaseLog.With("scheduleNamespace", req.Namespace, "scheduleName", req.Name)
	r.log.Info("Reconciling DisruptionSchedule")

	instance := &chaosv1beta1.DisruptionSchedule{}

	// retrieve schedule
	if err := r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !instance.DeletionTimestamp.IsZero() {
		// NOTE: add a finalizer if anything needs to be written here
		// The DisruptionSchedule is being deleted.
		return ctrl.Result{}, nil
	}

	targetResourceExists, instanceDeleted, err := r.updateTargetResourcePreviouslyMissing(ctx, instance)
	if err != nil {
		// Error occurred during status update or deletion, requeue
		r.log.Errorw("failed to handle target resource status", "err", err)
		return ctrl.Result{}, err
	}

	if instanceDeleted {
		// The instance has been deleted, reconciliation can be skipped
		return ctrl.Result{}, nil
	}

	disruptions, err := r.getChildDisruptions(ctx, instance)
	if err != nil {
		return ctrl.Result{}, nil
	}

	if err := r.updateLastScheduleTime(ctx, instance, disruptions); err != nil {
		r.log.Errorw("unable to update LastScheduleTime of DisruptionSchedule status", "err", err)
		return ctrl.Result{}, err
	}

	// Get the next scheduled run, or the time of the unproccessed run
	// If there are multiple unmet start times, only start last one
	missedRun, nextRun, err := r.getNextSchedule(instance, time.Now())
	if err != nil {
		r.log.Errorw("unable to figure out disruption schedule", "err", err)
		// we don't really care about requeuing until we get an update that
		// fixes the schedule, so don't return an error
		return ctrl.Result{}, nil
	}

	// Calculate the requeue time
	scheduledResult := ctrl.Result{RequeueAfter: time.Until(nextRun)}
	requeueTime := scheduledResult.RequeueAfter.Round(time.Second)

	r.log.Infow("calculated next scheduled run", "nextRun", nextRun.Format(time.UnixDate), "now", time.Now().Format(time.UnixDate))

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

	r.log.Infow("processing current run", "currentRun", missedRun.Format(time.UnixDate))

	tooLate := false
	if instance.Spec.StartingDeadlineSeconds != nil {
		tooLate = missedRun.Add(time.Duration(*instance.Spec.StartingDeadlineSeconds) * time.Second).Before(time.Now())
	}

	if tooLate {
		r.log.Infow(fmt.Sprintf("missed schedule to start a disruption at %s, scheduling next check in %s", missedRun, requeueTime))
		return scheduledResult, nil
	}

	disruptionCreated, err := r.createDisruption(ctx, instance, missedRun)

	if err != nil {
		r.log.Warnw(fmt.Sprintf("failed to create a disruption, scheduling next check in %s", requeueTime), "err", err)
		return scheduledResult, err
	}

	if disruptionCreated {
		r.log.Infow("created Disruption")

		instance.Status.LastScheduleTime = &metav1.Time{Time: missedRun}

		if err := r.Client.Status().Update(ctx, instance); err != nil {
			r.log.Errorw("unable to update LastScheduleTime of DisruptionSchedule status, requeue", "err", err)
			return ctrl.Result{}, err
		}
	}

	return scheduledResult, nil
}

func (r *DisruptionScheduleReconciler) getChildDisruptions(ctx context.Context, instance *chaosv1beta1.DisruptionSchedule) (*chaosv1beta1.DisruptionList, error) {
	disruptions := &chaosv1beta1.DisruptionList{}
	labelSelector := labels.SelectorFromSet(labels.Set{DisruptionScheduleNameLabel: instance.Name})

	if err := r.Client.List(ctx, disruptions, client.InNamespace(instance.Namespace), &client.ListOptions{LabelSelector: labelSelector}); err != nil {
		r.log.Errorw("unable to list Disruptions", "err", err)
		return disruptions, err
	}

	return disruptions, nil
}

// updateLastScheduleTime updates the LastScheduleTime in the status of a DisruptionSchedule instance based on the most recent schedule time among the given disruptions
func (r *DisruptionScheduleReconciler) updateLastScheduleTime(ctx context.Context, instance *chaosv1beta1.DisruptionSchedule, disruptions *chaosv1beta1.DisruptionList) error {
	mostRecentScheduleTime := r.getMostRecentScheduleTime(disruptions) // find the last run so we can update the status
	if mostRecentScheduleTime != nil {
		instance.Status.LastScheduleTime = &metav1.Time{Time: *mostRecentScheduleTime}
		return r.Client.Status().Update(ctx, instance)
	}

	return nil // No need to update if mostRecentScheduleTime is nil
}

// getMostRecentScheduleTime returns the pointer to the most recent scheduled time among a list of disruptions
func (r *DisruptionScheduleReconciler) getMostRecentScheduleTime(disruptions *chaosv1beta1.DisruptionList) *time.Time {
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
func (r *DisruptionScheduleReconciler) getScheduledTimeForDisruption(disruption *chaosv1beta1.Disruption) (*time.Time, error) {
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

// getTargetResource retrieves the target resource specified in the DisruptionSchedule.
// It returns two values:
// - 'client.Object': Represents the target resource (Deployment or StatefulSet).
// - 'error': Any error encountered during retrieval.
func (r *DisruptionScheduleReconciler) getTargetResource(ctx context.Context, instance *chaosv1beta1.DisruptionSchedule) (client.Object, error) {
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
// - 'bool': Indicates whether the target resource is currently found.
// - 'error': Represents any error that occurred during the execution of the function.
func (r *DisruptionScheduleReconciler) checkTargetResourceExists(ctx context.Context, instance *chaosv1beta1.DisruptionSchedule) (bool, error) {
	_, err := r.getTargetResource(ctx, instance)

	if errors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

// updateTargetResourcePreviouslyMissing is responsible for updating the status when the target resource was previously missing.
// The function returns three values:
// - 'targetResourceExists' (bool): Indicates whether the target resource is currently found.
// - 'disruptionScheduleDeleted' (bool): Indicates whether the disruption schedule was deleted due to the target resource being missing for more than the expiration duration.
// - 'error': Represents any error that occurred during the execution of the function.
func (r *DisruptionScheduleReconciler) updateTargetResourcePreviouslyMissing(ctx context.Context, instance *chaosv1beta1.DisruptionSchedule) (bool, bool, error) {
	disruptionScheduleDeleted := false
	targetResourceExists, err := r.checkTargetResourceExists(ctx, instance)

	if err != nil {
		return targetResourceExists, disruptionScheduleDeleted, err
	}

	if !targetResourceExists {
		r.log.Warnw("target does not exist, this schedule will be deleted if that continues", "error", err)

		if instance.Status.TargetResourcePreviouslyMissing == nil {
			r.log.Warnw("target is missing for the first time, updating status")

			return targetResourceExists, disruptionScheduleDeleted, r.handleTargetResourceFirstMissing(ctx, instance)
		}

		if time.Since(instance.Status.TargetResourcePreviouslyMissing.Time) > TargetResourceMissingThreshold {
			r.log.Errorw("target has been missing for over one day, deleting this schedule",
				"timeMissing", time.Since(instance.Status.TargetResourcePreviouslyMissing.Time))

			disruptionScheduleDeleted = true

			return targetResourceExists, disruptionScheduleDeleted, r.handleTargetResourceMissingPastExpiration(ctx, instance)
		}
	} else if instance.Status.TargetResourcePreviouslyMissing != nil {
		r.log.Infow("target was previously missing, but now present. updating the status accordingly")

		return targetResourceExists, disruptionScheduleDeleted, r.handleTargetResourceNowPresent(ctx, instance)
	}

	return targetResourceExists, disruptionScheduleDeleted, nil
}

// handleTargetResourceFirstMissing handles the scenario when the target resource is missing for the first time.
// It updates the status of the DisruptionSchedule instance.
func (r *DisruptionScheduleReconciler) handleTargetResourceFirstMissing(ctx context.Context, instance *chaosv1beta1.DisruptionSchedule) error {
	instance.Status.TargetResourcePreviouslyMissing = &metav1.Time{Time: time.Now()}
	if err := r.Client.Status().Update(ctx, instance); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	return nil
}

// handleTargetResourceMissingPastExpiration handles the scenario when the target resource has been missing for more than the expiration period.
// It deletes the DisruptionSchedule instance.
func (r *DisruptionScheduleReconciler) handleTargetResourceMissingPastExpiration(ctx context.Context, instance *chaosv1beta1.DisruptionSchedule) error {
	if err := r.Client.Delete(ctx, instance); err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	return nil
}

// handleTargetResourceNowPresent handles the scenario when the target resource was previously missing but is now present.
// It updates the status of the DisruptionSchedule instance.
func (r *DisruptionScheduleReconciler) handleTargetResourceNowPresent(ctx context.Context, instance *chaosv1beta1.DisruptionSchedule) error {
	instance.Status.TargetResourcePreviouslyMissing = nil
	if err := r.Client.Status().Update(ctx, instance); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	return nil
}

// getNextSchedule calculates the next scheduled time for a disruption instance based on its cron schedule and the current time.
// It returns the last missed schedule time, the next scheduled time, and any error encountered during parsing the schedule.
func (r *DisruptionScheduleReconciler) getNextSchedule(instance *chaosv1beta1.DisruptionSchedule, now time.Time) (lastMissed time.Time, next time.Time, err error) {
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

// getSelectors retrieves the labels of the target resource specified in the DisruptionSchedule.
// The function returns two values:
// - labels.Set: A set of labels of the target resource which will be used as the selectors for Disruption.
// - error: An error if the target resource or labels retrieval fails.
func (r *DisruptionScheduleReconciler) getSelectors(ctx context.Context, instance *chaosv1beta1.DisruptionSchedule) (labels.Set, error) {
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

// createDisruption creates a Disruption object based on the provided DisruptionSchedule,
// sets the necessary metadata and spec fields, and creates the object in the Kubernetes cluster.
// The function returns two values:
// - bool: A boolean indicating whether the creation was successful.
// - error: An error if any operation fails.
func (r *DisruptionScheduleReconciler) createDisruption(ctx context.Context, instance *chaosv1beta1.DisruptionSchedule, scheduledTime time.Time) (bool, error) {
	created := false
	name := fmt.Sprintf("disruption-schedule-%s", instance.Name)

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

	selectors, err := r.getSelectors(ctx, instance)
	if err != nil {
		return created, err
	}

	disruption.Labels[DisruptionScheduleNameLabel] = instance.Name

	// overwriting selectors
	if disruption.Spec.Selector == nil {
		disruption.Spec.Selector = make(map[string]string)
	}

	for k, v := range selectors {
		disruption.Spec.Selector[k] = v
	}

	if err := ctrl.SetControllerReference(instance, disruption, r.Scheme); err != nil {
		return created, err
	}

	if err := r.Client.Create(ctx, disruption); err != nil {
		return created, err
	}

	created = true

	return created, nil
}

// SetupWithManager setups the current reconciler with the given manager
func (r *DisruptionScheduleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chaosv1beta1.DisruptionSchedule{}).
		Complete(r)
}
