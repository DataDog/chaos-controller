// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package v1beta1

import (
	"errors"
	"strings"

	"github.com/DataDog/chaos-controller/utils"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
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

	// send informative event to disruption cron to broadcast
	disruptionCronWebhookRecorder.AnnotatedEventf(d, d.Annotations, Events[EventDisruptionCronCreated].Type, string(EventDisruptionCronCreated), Events[EventDisruptionCronCreated].OnDisruptionTemplateMessage)

	return nil, nil
}

func (d *DisruptionCron) ValidateUpdate(oldObject runtime.Object) (warnings admission.Warnings, err error) {
	log := logger.With("disruptionCronName", d.Name, "disruptionCronNamespace", d.Namespace)

	log.Infow("validating updated disruption cron", "spec", d.Spec)

	if err := validateUserInfoImmutable(oldObject.(*DisruptionCron), d); err != nil {
		return nil, err
	}

	// send informative event to disruption cron to broadcast
	disruptionCronWebhookRecorder.Event(d, Events[EventDisruptionCronUpdated].Type, string(EventDisruptionCronUpdated), Events[EventDisruptionCronUpdated].OnDisruptionTemplateMessage)

	return nil, nil
}

func (d *DisruptionCron) ValidateDelete() (warnings admission.Warnings, err error) {
	log := disruptionCronWebhookLogger.With("disruptionCronName", d.Name, "disruptionCronNamespace", d.Namespace)

	log.Infow("validating deleted disruption cron", "spec", d.Spec)

	// send informative event to disruption cron to broadcast
	disruptionCronWebhookRecorder.Event(d, Events[EventDisruptionCronDeleted].Type, string(EventDisruptionCronDeleted), Events[EventDisruptionCronDeleted].OnDisruptionTemplateMessage)

	return nil, nil
}
