// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package v1beta1

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/DataDog/chaos-controller/utils"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
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
)

func (d *DisruptionCron) SetupWebhookWithManager(setupWebhookConfig utils.SetupWebhookWithManagerConfig) error {
	disruptionCronWebhookRecorder = setupWebhookConfig.Recorder
	disruptionCronWebhookDeleteOnly = setupWebhookConfig.DeleteOnlyFlag
	disruptionCronWebhookLogger = setupWebhookConfig.Logger.With(
		"source", "admission-controller",
		"admission-controller", "disruption-cron-webhook",
	)

	disruptionCronPermittedUserGroups = map[string]struct{}{}

	for _, group := range setupWebhookConfig.PermittedUserGroups {
		disruptionCronPermittedUserGroups[group] = struct{}{}
	}

	disruptionCronPermittedUserGroupString = strings.Join(setupWebhookConfig.PermittedUserGroups, ",")

	return ctrl.NewWebhookManagedBy(setupWebhookConfig.Manager).
		For(d).
		Complete()
}

//+kubebuilder:webhook:webhookVersions={v1},path=/validate-chaos-datadoghq-com-v1beta1-disruptioncron,mutating=false,failurePolicy=fail,sideEffects=None,groups=chaos.datadoghq.com,resources=disruptioncrons,verbs=create;update;delete,versions=v1beta1,name=vdisruptioncron.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &DisruptionCron{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (d *DisruptionCron) ValidateCreate() (admission.Warnings, error) {
	log := disruptionCronWebhookLogger.With("disruptionCronName", d.Name, "disruptionCronNamespace", d.Namespace)

	log.Infow("validating created disruption cron", "spec", d.Spec)

	// delete-only mode, reject everything trying to be created
	if disruptionCronWebhookDeleteOnly {
		return nil, errors.New("the controller is currently in delete-only mode, you can't create new disruption cron for now")
	}

	if err := validateUserInfoGroup(d, disruptionCronPermittedUserGroups, disruptionCronPermittedUserGroupString); err != nil {
		return nil, err
	}

	if err := d.Spec.DisruptionTemplate.ValidateSelectorsOptional(false); err != nil {
		return nil, fmt.Errorf("error while validating the spec.disruptionTemplate: %w", err)
	}

	if d.Spec.DisruptionTemplate.Reporting != nil {
		return nil, fmt.Errorf("disruptions created by DisruptionCrons inherit the spec.reporting field of the parent, do not set spec.disruptionTemplate.reporting")
	}

	// send informative event to disruption cron to broadcast
	d.emitEvent(EventDisruptionCronCreated)

	return nil, nil
}

func (d *DisruptionCron) ValidateUpdate(oldObject runtime.Object) (admission.Warnings, error) {
	log := logger.With("disruptionCronName", d.Name, "disruptionCronNamespace", d.Namespace)

	log.Infow("validating updated disruption cron", "spec", d.Spec)

	if err := validateUserInfoImmutable(oldObject.(*DisruptionCron), d); err != nil {
		return nil, err
	}

	if err := d.Spec.DisruptionTemplate.ValidateSelectorsOptional(false); err != nil {
		return nil, err
	}

	// send informative event to disruption cron to broadcast
	d.emitEvent(EventDisruptionCronUpdated)

	return nil, nil
}

func (d *DisruptionCron) ValidateDelete() (warnings admission.Warnings, err error) {
	log := disruptionCronWebhookLogger.With("disruptionCronName", d.Name, "disruptionCronNamespace", d.Namespace)

	log.Infow("validating deleted disruption cron", "spec", d.Spec)

	// During the validation of the deletion the timestamp does not exist so we need to set it before emitting the event
	d.DeletionTimestamp = &metav1.Time{Time: time.Now()}

	// send informative event to disruption cron to broadcast
	d.emitEvent(EventDisruptionCronDeleted)

	return nil, nil
}

func (d *DisruptionCron) emitEvent(eventReason EventReason) {
	disruptionCronJSON, err := json.Marshal(d)
	if err != nil {
		disruptionCronWebhookLogger.Errorw("failed to marshal disruption cron", "error", err)
		return
	}

	annotations := map[string]string{
		EventDisruptionCronAnnotation: string(disruptionCronJSON),
	}

	disruptionCronWebhookRecorder.AnnotatedEventf(d, annotations, Events[eventReason].Type, string(eventReason), Events[eventReason].OnDisruptionTemplateMessage)
}
