// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package controllers

import (
	"context"
	"time"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
	ScheduledAtAnnotation       = chaosv1beta1.GroupName + "/scheduled-at"
	DisruptionScheduleNameLabel = chaosv1beta1.GroupName + "/disruption-schedule-name"
)

func (r *DisruptionScheduleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	r.log = r.BaseLog.With("scheduleNamespace", req.Namespace, "scheduleName", req.Name)
	r.log.Info("Reconciling DisruptionSchedule")

	instance := &chaosv1beta1.DisruptionSchedule{}

	// retrieve schedule
	if err := r.Client.Get(context.Background(), req.NamespacedName, instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !instance.DeletionTimestamp.IsZero() {
		// NOTE: add a finalizer if anything needs to be written here
	} else {
		disruptions := &chaosv1beta1.DisruptionList{}
		if err := r.Client.List(ctx, disruptions, client.InNamespace(instance.Namespace), client.MatchingFields{DisruptionScheduleNameLabel: instance.Name}); err != nil {
			r.log.Errorw("unable to list Disruptions", "DisruptionSchedule", instance.Name, "err", err)
			return ctrl.Result{}, err
		}

		if err := r.updateLastScheduleTime(ctx, instance, disruptions); err != nil {
			r.log.Errorw("unable to update LastScheduleTime of DisruptionSchedule status", "DisruptionSchedule", instance.Name, "err", err)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
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

// SetupWithManager setups the current reconciler with the given manager
func (r *DisruptionScheduleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chaosv1beta1.DisruptionSchedule{}).
		Complete(r)
}
