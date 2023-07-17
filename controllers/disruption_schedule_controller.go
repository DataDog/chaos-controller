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

	disruptions, err := r.getChildDisruptions(ctx, instance)
	if err != nil {
		return ctrl.Result{}, nil
	}

	if err := r.updateLastScheduleTime(ctx, instance, disruptions); err != nil {
		r.log.Errorw("unable to update LastScheduleTime of DisruptionSchedule status", "err", err)
		return ctrl.Result{}, err
	}

	// _ is targetResourceNotFound, which will be used later to requeue according to the schedule
	_, instanceDeleted, err := r.updateTargetResourcePreviouslyMissing(ctx, instance)
	if err != nil {
		// Error occurred during status update or deletion, requeue
		r.log.Errorw("failed to handle target resource status", "err", err)
		return ctrl.Result{}, err
	}

	if instanceDeleted {
		// The instance has been deleted, reconciliation can be skipped
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
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
	} else {
		instance.Status.LastScheduleTime = nil
	}

	return r.Client.Status().Update(ctx, instance)
}

// getMostRecentScheduleTime returns the pointer to the most recent scheduled time among a list of disruptions
func (r *DisruptionScheduleReconciler) getMostRecentScheduleTime(disruptions *chaosv1beta1.DisruptionList) *time.Time {
	var mostRecentScheduleTime *time.Time

	for _, disruption := range disruptions.Items {
		scheduledTimeForDisruption, err := r.getScheduledTimeForDisruption(&disruption)
		if err != nil {
			r.log.Errorw("unable to parse schedule time for child disruption", "err", err, "disruption", &disruption)
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

// checkTargetResourceExists checks whether the target resource exists.
// It returns two values:
// - 'targetResourceNotFound' (bool): Indicates whether the target resource is currently not found.
// - 'error': Represents any error that occurred during the execution of the function.
func (r *DisruptionScheduleReconciler) checkTargetResourceExists(ctx context.Context, instance *chaosv1beta1.DisruptionSchedule) (bool, error) {
	var targetObj client.Object

	switch instance.Spec.TargetResource.Kind {
	case "Deployment":
		targetObj = &appsv1.Deployment{}
	case "StatefulSet":
		targetObj = &appsv1.StatefulSet{}
	}

	err := r.Client.Get(ctx, types.NamespacedName{
		Name:      instance.Spec.TargetResource.Name,
		Namespace: instance.Namespace,
	}, targetObj)

	return errors.IsNotFound(err), err
}

// updateTargetResourcePreviouslyMissing is responsible for updating the status when the target resource was previously missing.
// The function returns three values:
// - 'targetResourceNotFound' (bool): Indicates whether the target resource is currently not found.
// - 'disruptionScheduleDeleted' (bool): Indicates whether the disruption schedule was deleted due to the target resource being missing for more than the expiration duration.
// - 'error': Represents any error that occurred during the execution of the function.
func (r *DisruptionScheduleReconciler) updateTargetResourcePreviouslyMissing(ctx context.Context, instance *chaosv1beta1.DisruptionSchedule) (bool, bool, error) {
	disruptionScheduleDeleted := false
	targetResourceNotFound, err := r.checkTargetResourceExists(ctx, instance)

	if targetResourceNotFound {
		r.log.Warnw("target does not exist, this schedule will be deleted if that continues", "error", err)

		if instance.Status.TargetResourcePreviouslyMissing == nil {
			r.log.Warnw("target is missing for the first time, updating status")

			return targetResourceNotFound, disruptionScheduleDeleted, r.handleTargetResourceFirstMissing(ctx, instance)
		}

		if time.Since(instance.Status.TargetResourcePreviouslyMissing.Time) > TargetResourceMissingThreshold {
			r.log.Errorw("target has been missing for over one day, deleting this schedule",
				"timeMissing", time.Since(instance.Status.TargetResourcePreviouslyMissing.Time))

			disruptionScheduleDeleted = true

			return targetResourceNotFound, disruptionScheduleDeleted, r.handleTargetResourceMissingPastExpiration(ctx, instance)
		}
	} else if instance.Status.TargetResourcePreviouslyMissing != nil {
		r.log.Infow("target was previously missing, but now present. updating the status accordingly")

		return targetResourceNotFound, disruptionScheduleDeleted, r.handleTargetResourceNowPresent(ctx, instance)
	}

	return targetResourceNotFound, disruptionScheduleDeleted, nil
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

// SetupWithManager setups the current reconciler with the given manager
func (r *DisruptionScheduleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chaosv1beta1.DisruptionSchedule{}).
		Complete(r)
}
