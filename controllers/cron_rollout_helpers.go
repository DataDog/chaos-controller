// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.
package controllers

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ScheduledAtAnnotation          = chaosv1beta1.GroupName + "/scheduled-at"
	DisruptionCronNameLabel        = chaosv1beta1.GroupName + "/disruption-cron-name"
	DisruptionRolloutNameLabel     = chaosv1beta1.GroupName + "/disruption-rollout-name"
	TargetResourceMissingThreshold = time.Hour * 24
)

// GetChildDisruptions retrieves disruptions associated with a resource by its label.
// Most of the time, this will return an empty list as disruptions are typically short-lived objects.
func GetChildDisruptions(ctx context.Context, cl client.Client, log *zap.SugaredLogger, namespace, labelKey, labelVal string) (*chaosv1beta1.DisruptionList, error) {
	disruptions := &chaosv1beta1.DisruptionList{}
	labelSelector := labels.SelectorFromSet(labels.Set{labelKey: labelVal})

	if err := cl.List(ctx, disruptions, client.InNamespace(namespace), &client.ListOptions{LabelSelector: labelSelector}); err != nil {
		log.Errorw("unable to list Disruptions", "err", err)
		return disruptions, err
	}

	return disruptions, nil
}

// GetTargetResource retrieves the specified target resource (Deployment or StatefulSet).
// It returns the target resource object and any error encountered during retrieval.
func GetTargetResource(ctx context.Context, cl client.Client, targetResource *chaosv1beta1.TargetResourceSpec, namespace string) (client.Object, error) {
	var targetObj client.Object

	switch targetResource.Kind {
	case "deployment":
		targetObj = &appsv1.Deployment{}
	case "statefulset":
		targetObj = &appsv1.StatefulSet{}
	}

	err := cl.Get(ctx, types.NamespacedName{
		Name:      targetResource.Name,
		Namespace: namespace,
	}, targetObj)

	return targetObj, err
}

// CheckTargetResourceExists determines if the target resource exists.
// Returns a boolean indicating presence and an error if one occurs.
func CheckTargetResourceExists(ctx context.Context, cl client.Client, targetResource *chaosv1beta1.TargetResourceSpec, namespace string) (bool, error) {
	_, err := GetTargetResource(ctx, cl, targetResource, namespace)

	if apierrors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

// GetSelectors retrieves the labels of the specified target resource (Deployment or StatefulSet).
// Returns a set of labels to be used as Disruption selectors and an error if retrieval fails.
func GetSelectors(ctx context.Context, cl client.Client, targetResource *chaosv1beta1.TargetResourceSpec, namespace string) (labels.Set, error) {
	targetObj, err := GetTargetResource(ctx, cl, targetResource, namespace)
	if err != nil {
		return nil, err
	}

	// retrieve pod template spec from targeted resource
	podSpec := corev1.PodTemplateSpec{}
	switch o := targetObj.(type) {
	case *appsv1.Deployment:
		podSpec = o.Spec.Template
	case *appsv1.StatefulSet:
		podSpec = o.Spec.Template
	default:
		return nil, errors.New("error getting target resource pod template labels")
	}

	labels := podSpec.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	return labels, nil
}

// createBaseDisruption generates a basic Disruption object using the provided owner and disruptionSpec.
// The returned Disruption object has its basic details set, but it's not saved or stored anywhere yet.
func createBaseDisruption(owner metav1.Object, disruptionSpec *chaosv1beta1.DisruptionSpec) *chaosv1beta1.Disruption {
	name := generateDisruptionName(owner)

	disruption := &chaosv1beta1.Disruption{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   owner.GetNamespace(),
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
		},
		Spec: *disruptionSpec.DeepCopy(),
	}

	switch d := owner.(type) {
	case *chaosv1beta1.DisruptionCron:
		disruption.Spec.Reporting = d.Spec.Reporting
	default:
	}

	return disruption
}

// setDisruptionAnnotations updates the annotations of a given Disruption object with those of its owner.
// Additionally, it sets a scheduled time annotation using the provided scheduledTime.
func setDisruptionAnnotations(disruption *chaosv1beta1.Disruption, owner metav1.Object, scheduledTime time.Time) {
	for k, v := range owner.GetAnnotations() {
		disruption.Annotations[k] = v
	}

	disruption.Annotations[ScheduledAtAnnotation] = scheduledTime.Format(time.RFC3339)
}

// overwriteDisruptionSelectors updates the selectors of a given Disruption object based on the provided targetResource.
// Returns an error if fetching selectors from the target resource fails.
func overwriteDisruptionSelectors(ctx context.Context, cl client.Client, disruption *chaosv1beta1.Disruption, targetResource *chaosv1beta1.TargetResourceSpec, namespace string) error {
	// Get selectors from target resource
	selectors, err := GetSelectors(ctx, cl, targetResource, namespace)
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

// CreateDisruptionFromTemplate constructs a Disruption object based on the provided owner, disruptionSpec, and targetResource.
// The function sets annotations, overwrites selectors, and associates the Disruption with its owner.
// It returns the constructed Disruption or an error if any step fails.
func CreateDisruptionFromTemplate(ctx context.Context, cl client.Client, scheme *runtime.Scheme, owner metav1.Object, targetResource *chaosv1beta1.TargetResourceSpec, disruptionSpec *chaosv1beta1.DisruptionSpec, scheduledTime time.Time) (*chaosv1beta1.Disruption, error) {
	disruption := createBaseDisruption(owner, disruptionSpec)

	ownerNameLabel := getOwnerNameLabel(owner)
	disruption.Labels[ownerNameLabel] = owner.GetName()

	setDisruptionAnnotations(disruption, owner, scheduledTime)

	if err := overwriteDisruptionSelectors(ctx, cl, disruption, targetResource, owner.GetNamespace()); err != nil {
		return nil, err
	}

	if err := ctrl.SetControllerReference(owner, disruption, scheme); err != nil {
		return nil, err
	}

	return disruption, nil
}

// getScheduledTimeForDisruption returns the scheduled time for a particular disruption.
func getScheduledTimeForDisruption(log *zap.SugaredLogger, disruption *chaosv1beta1.Disruption) time.Time {
	timeRaw := disruption.Annotations[ScheduledAtAnnotation]
	if len(timeRaw) == 0 {
		return time.Time{}
	}

	timeParsed, err := time.Parse(time.RFC3339, timeRaw)
	if err != nil {
		log.Errorw("unable to parse schedule time for child disruption", "err", err, "disruptionName", disruption.Name)
		return time.Time{}
	}

	return timeParsed
}

// GetMostRecentScheduleTime returns the most recent scheduled time from a list of disruptions.
func GetMostRecentScheduleTime(log *zap.SugaredLogger, disruptions *chaosv1beta1.DisruptionList) time.Time {
	length := len(disruptions.Items)
	if length == 0 {
		return time.Time{}
	}

	sort.Slice(disruptions.Items, func(i, j int) bool {
		scheduleTime1 := getScheduledTimeForDisruption(log, &disruptions.Items[i])
		scheduleTime2 := getScheduledTimeForDisruption(log, &disruptions.Items[j])

		return scheduleTime1.Before(scheduleTime2)
	})

	return getScheduledTimeForDisruption(log, &disruptions.Items[length-1])
}

// generateDisruptionName produces a disruption name based on the specific CR controller, that's invoking it.
// It returns a formatted string name.
func generateDisruptionName(owner metav1.Object) string {
	switch typedOwner := owner.(type) {
	case *chaosv1beta1.DisruptionCron:
		return fmt.Sprintf("disruption-cron-%s", typedOwner.GetName())
	case *chaosv1beta1.DisruptionRollout:
		return fmt.Sprintf("disruption-rollout-%s", typedOwner.GetName())
	}

	return ""
}

// getOwnerNameLabel derives the appropriate label for the owner CR.
// It returns the label string.
func getOwnerNameLabel(owner metav1.Object) string {
	switch owner.(type) {
	case *chaosv1beta1.DisruptionCron:
		return DisruptionCronNameLabel
	case *chaosv1beta1.DisruptionRollout:
		return DisruptionRolloutNameLabel
	}

	return ""
}
