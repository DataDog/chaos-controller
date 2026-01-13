// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package v1beta1

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	validationutils "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/DataDog/chaos-controller/o11y/metrics"
	tagutil "github.com/DataDog/chaos-controller/o11y/tags"
	"github.com/DataDog/chaos-controller/utils"
)

const (
	EventDisruptionCronAnnotation = "disruption_cron"
	EventDisruptionAnnotation     = "disruption"
)

var (
	disruptionCronWebhookLogger            *zap.SugaredLogger
	disruptionCronWebhookRecorder          record.EventRecorder
	disruptionCronWebhookDeleteOnly        bool
	disruptionCronPermittedUserGroups      map[string]struct{}
	disruptionCronPermittedUserGroupString string
	disruptionCronMetricsSink              metrics.Sink
	defaultCronDelayedStartTolerance       time.Duration
	minimumCronFrequency                   time.Duration
)

func (d *DisruptionCron) SetupWebhookWithManager(setupWebhookConfig utils.SetupWebhookWithManagerConfig) error {
	disruptionCronWebhookRecorder = setupWebhookConfig.Recorder
	disruptionCronWebhookDeleteOnly = setupWebhookConfig.DeleteOnlyFlag
	disruptionCronWebhookLogger = setupWebhookConfig.Logger.With(
		tagutil.SourceKey, "admission-controller",
		tagutil.AdmissionControllerKey, "disruption-cron-webhook",
	)
	disruptionCronMetricsSink = setupWebhookConfig.MetricsSink

	disruptionCronPermittedUserGroups = map[string]struct{}{}

	for _, group := range setupWebhookConfig.PermittedUserGroups {
		disruptionCronPermittedUserGroups[group] = struct{}{}
	}

	disruptionCronPermittedUserGroupString = strings.Join(setupWebhookConfig.PermittedUserGroups, ",")
	defaultCronDelayedStartTolerance = setupWebhookConfig.DefaultCronDelayedStartTolerance
	minimumCronFrequency = setupWebhookConfig.MinimumCronFrequency
	defaultDuration = setupWebhookConfig.DefaultDurationFlag

	return ctrl.NewWebhookManagedBy(setupWebhookConfig.Manager).
		WithDefaulter(&DisruptionCron{}).
		WithValidator(&DisruptionCron{}).
		For(d).
		Complete()
}

//+kubebuilder:webhook:webhookVersions={v1},path=/mutate-chaos-datadoghq-com-v1beta1-disruptioncron,mutating=true,failurePolicy=fail,sideEffects=None,groups=chaos.datadoghq.com,resources=disruptioncrons,verbs=create;update,versions=v1beta1,name=mdisruptioncron.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.CustomDefaulter = &DisruptionCron{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (d *DisruptionCron) Default(_ context.Context, obj runtime.Object) error {
	disruptionCronObj, ok := obj.(*DisruptionCron)
	if !ok {
		return fmt.Errorf("expected *DisruptionCron, got %T", obj)
	}

	log := disruptionCronWebhookLogger.With(
		tagutil.DisruptionCronNameKey, disruptionCronObj.Name,
		tagutil.DisruptionCronNamespaceKey, disruptionCronObj.Namespace,
	)

	log.Infow("defaulting disruption cron", tagutil.SpecKey, disruptionCronObj.Spec)

	if disruptionCronObj.Spec.DelayedStartTolerance.Duration() == 0 {
		log.Infow(fmt.Sprintf("setting default delayedStartTolerance of %s in disruptionCron", defaultCronDelayedStartTolerance),
			tagutil.DisruptionCronNameKey, disruptionCronObj.Name,
			tagutil.DisruptionCronNamespaceKey, disruptionCronObj.Namespace,
		)

		disruptionCronObj.Spec.DelayedStartTolerance = DisruptionDuration(defaultCronDelayedStartTolerance.String())
	}

	return nil
}

//+kubebuilder:webhook:webhookVersions={v1},path=/validate-chaos-datadoghq-com-v1beta1-disruptioncron,mutating=false,failurePolicy=fail,sideEffects=None,groups=chaos.datadoghq.com,resources=disruptioncrons,verbs=create;update;delete,versions=v1beta1,name=vdisruptioncron.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.CustomValidator = &DisruptionCron{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (d *DisruptionCron) ValidateCreate(_ context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	disruptionCronObj, ok := obj.(*DisruptionCron)
	if !ok {
		return nil, fmt.Errorf("expected *DisruptionCron, got %T", obj)
	}

	log := disruptionCronWebhookLogger.With(
		tagutil.DisruptionCronNameKey, disruptionCronObj.Name,
		tagutil.DisruptionCronNamespaceKey, disruptionCronObj.Namespace,
	)

	log.Infow("validating created disruption cron", tagutil.SpecKey, disruptionCronObj.Spec)

	metricTags := disruptionCronObj.getMetricsTags()

	defer func() {
		if err != nil {
			if mErr := disruptionCronMetricsSink.MetricValidationFailed(metricTags); mErr != nil {
				log.Errorw("error sending a metric", tagutil.ErrorKey, mErr)
			}
		}
	}()

	// delete-only mode, reject everything trying to be created
	if disruptionCronWebhookDeleteOnly {
		return nil, errors.New("the controller is currently in delete-only mode, you can't create new disruption cron for now")
	}

	if err = validateUserInfoGroup(disruptionCronObj, disruptionCronPermittedUserGroups, disruptionCronPermittedUserGroupString); err != nil {
		return nil, err
	}

	if err = disruptionCronObj.validateDisruptionCronName(); err != nil {
		return nil, err
	}

	if err = disruptionCronObj.validateDisruptionCronSpec(); err != nil {
		return nil, err
	}

	if err = disruptionCronObj.validateMinimumFrequency(minimumCronFrequency); err != nil {
		return nil, err
	}

	if mErr := metricsSink.MetricValidationCreated(metricTags); mErr != nil {
		log.Errorw("error sending a metric", tagutil.ErrorKey, mErr)
	}

	// send informative event to disruption cron to broadcast
	disruptionCronObj.emitEvent(EventDisruptionCronCreated)

	return nil, nil
}

func (d *DisruptionCron) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (warnings admission.Warnings, err error) {
	newDisruptionCronObj, ok := newObj.(*DisruptionCron)
	if !ok {
		return nil, fmt.Errorf("expected *DisruptionCron, got %T", newObj)
	}

	oldDisruptionCronObj, ok := oldObj.(*DisruptionCron)
	if !ok {
		return nil, fmt.Errorf("expected *DisruptionCron, got %T", oldObj)
	}

	log := disruptionCronWebhookLogger.With(
		tagutil.DisruptionCronNameKey, newDisruptionCronObj.Name,
		tagutil.DisruptionCronNamespaceKey, newDisruptionCronObj.Namespace,
	)

	log.Infow("validating updated disruption cron", tagutil.SpecKey, newDisruptionCronObj.Spec)

	metricTags := newDisruptionCronObj.getMetricsTags()

	defer func() {
		if err != nil {
			if mErr := disruptionCronMetricsSink.MetricValidationFailed(metricTags); mErr != nil {
				log.Errorw("error sending a metric", tagutil.ErrorKey, mErr)
			}
		}
	}()

	if err = validateUserInfoImmutable(oldDisruptionCronObj, newDisruptionCronObj); err != nil {
		return nil, err
	}

	if err = newDisruptionCronObj.validateDisruptionCronName(); err != nil {
		return nil, err
	}

	if err = newDisruptionCronObj.validateDisruptionCronSpec(); err != nil {
		return nil, err
	}

	// If a DisruptionCron is already more frequent than the minimum frequency, we don't want to
	// block updates on other parts of the spec or status. But if the schedule is being updated,
	// we do want to enforce that it happens more often than the minimum frequency
	if oldDisruptionCronObj.Spec.Schedule != newDisruptionCronObj.Spec.Schedule {
		if err := newDisruptionCronObj.validateMinimumFrequency(minimumCronFrequency); err != nil {
			return nil, err
		}
	}

	if mErr := metricsSink.MetricValidationUpdated(metricTags); mErr != nil {
		log.Errorw("error sending a metric", tagutil.ErrorKey, mErr)
	}

	// send informative event to disruption cron to broadcast
	newDisruptionCronObj.emitEvent(EventDisruptionCronUpdated)

	return nil, nil
}

func (d *DisruptionCron) ValidateDelete(_ context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	disruptionCronObj, ok := obj.(*DisruptionCron)
	if !ok {
		return nil, fmt.Errorf("expected *DisruptionCron, got %T", obj)
	}

	log := disruptionCronWebhookLogger.With(
		tagutil.DisruptionCronNameKey, disruptionCronObj.Name,
		tagutil.DisruptionCronNamespaceKey, disruptionCronObj.Namespace,
	)

	log.Infow("validating deleted disruption cron", tagutil.SpecKey, disruptionCronObj.Spec)

	// During the validation of the deletion the timestamp does not exist so we need to set it before emitting the event
	disruptionCronObj.DeletionTimestamp = &metav1.Time{Time: time.Now()}

	if mErr := metricsSink.MetricValidationDeleted(disruptionCronObj.getMetricsTags()); mErr != nil {
		log.Errorw("error sending a metric", tagutil.ErrorKey, mErr)
	}

	// send informative event to disruption cron to broadcast
	disruptionCronObj.emitEvent(EventDisruptionCronDeleted)

	return nil, nil
}

func (d *DisruptionCron) emitEvent(eventReason EventReason) {
	disruptionCronJSON, err := json.Marshal(d)
	if err != nil {
		disruptionCronWebhookLogger.Errorw("failed to marshal disruption cron", tagutil.ErrorKey, err)
		return
	}

	annotations := map[string]string{
		EventDisruptionCronAnnotation: string(disruptionCronJSON),
	}

	disruptionCronWebhookRecorder.AnnotatedEventf(d, annotations, Events[eventReason].Type, string(eventReason), Events[eventReason].OnDisruptionTemplateMessage)
}

func (d *DisruptionCron) validateDisruptionCronName() error {
	if len(d.ObjectMeta.Name) == 0 {
		return errors.New("disruption cron name must be specified")
	}

	if len(d.ObjectMeta.Name) > validationutils.DNS1035LabelMaxLength-16 {
		// The disruption name length is 63 character like all Kubernetes objects
		// (which must fit in a DNS subdomain). The disruption cron controller appends
		// a 16-character prefix (`disruption-cron-`) when creating
		// a disruption. The disruption name length limit is 63 characters.
		// Therefore disruption cron names must have length <= 63-16=47.
		// If we don't validate this here, then disruption creation will fail later.
		return fmt.Errorf("disruption cron name exceeds maximum length: must be no more than 47 characters, currently %d", len(d.ObjectMeta.Name))
	}

	// reject disruption crons with a name which would not be a valid label value
	// according to https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set
	if _, err := labels.Parse(fmt.Sprintf("name=%s", d.ObjectMeta.Name)); err != nil {
		return fmt.Errorf("disruption cron name must follow Kubernetes label format: %w", err)
	}

	return nil
}

func (d *DisruptionCron) validateDisruptionCronSpec() error {
	if _, err := cron.ParseStandard(d.Spec.Schedule); err != nil {
		return fmt.Errorf("spec.Schedule must follow the standard cron syntax: %w", err)
	}

	if d.Spec.DelayedStartTolerance.Duration() < 0 {
		return fmt.Errorf("spec.delayedStartTolerance must be a positive duration, currently %s", d.Spec.DelayedStartTolerance.Duration())
	}

	if err := d.Spec.DisruptionTemplate.ValidateSelectorsOptional(false); err != nil {
		return fmt.Errorf("spec.disruptionTemplate validation failed: %w", err)
	}

	return nil
}

func (d *DisruptionCron) validateMinimumFrequency(minFrequency time.Duration) error {
	schedule, err := cron.ParseStandard(d.Spec.Schedule)
	if err != nil {
		return fmt.Errorf("spec.Schedule must follow the standard cron syntax: %w", err)
	}

	specDuration := defaultDuration
	if d.Spec.DisruptionTemplate.Duration.Duration() > 0 {
		specDuration = d.Spec.DisruptionTemplate.Duration.Duration()
	}

	now := time.Now()
	nextDisruptionStarts := schedule.Next(now)
	nextDisruptionCompletes := nextDisruptionStarts.Add(specDuration)

	// Measure, "frequency", the time between when we would schedule the next two disruptions.
	// We don't want to measure from "now", because the cron standard will try to run at whole intervals, e.g.,
	// a schedule for "every 15 minutes", created at 1:05, will try to run the first disruption at 1:15. So we find the next two intervals,
	// which would be 1:15 and 1:30, and find the difference
	frequency := schedule.Next(nextDisruptionStarts).Sub(nextDisruptionStarts)

	// Measure, "interval", the time from when the next disruption completes, until the following disruption would start.
	// This lets us know how long the target will be undisrupted for, between two disruptions. If that's less than the minimum frequency,
	// we need to return an error
	interval := schedule.Next(nextDisruptionCompletes).Sub(nextDisruptionCompletes)
	if interval < minFrequency {
		return fmt.Errorf("this cron's spec.Schedule is \"%s\", which will create disruptions that last %s every %s. This leaves only %s between disruptions, but the minimum tolerated frequency is %s",
			d.Spec.Schedule, specDuration.String(), frequency.String(), interval.String(), minFrequency.String())
	}

	return nil
}
