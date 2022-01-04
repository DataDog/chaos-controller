// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package v1beta1

import (
	"errors"
	"fmt"
	"time"

	"github.com/DataDog/chaos-controller/metrics"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var logger *zap.SugaredLogger
var k8sClient client.Client
var metricsSink metrics.Sink
var deleteOnly bool
var handlerEnabled bool
var defaultDuration time.Duration

func (r *Disruption) SetupWebhookWithManager(mgr ctrl.Manager, l *zap.SugaredLogger, ms metrics.Sink, deleteOnlyFlag, handlerEnabledFlag bool, defaultDurationFlag time.Duration) error {
	logger = &zap.SugaredLogger{}
	*logger = *l.With("source", "admission-controller")
	k8sClient = mgr.GetClient()
	metricsSink = ms
	deleteOnly = deleteOnlyFlag
	handlerEnabled = handlerEnabledFlag
	defaultDuration = defaultDurationFlag

	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:webhookVersions={v1beta1},path=/mutate-chaos-datadoghq-com-v1beta1-disruption,mutating=true,failurePolicy=fail,sideEffects=None,groups=chaos.datadoghq.com,resources=disruptions,verbs=create;update,versions=v1beta1,name=mdisruption.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &Disruption{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Disruption) Default() {
	if r.Spec.Duration.Duration() == 0 {
		logger.Infow(fmt.Sprintf("setting default duration of %s in disruption", defaultDuration), "instance", r.Name, "namespace", r.Namespace)
		r.Spec.Duration = DisruptionDuration(defaultDuration.String())
	}
}

//+kubebuilder:webhook:webhookVersions={v1beta1},path=/validate-chaos-datadoghq-com-v1beta1-disruption,mutating=false,failurePolicy=fail,sideEffects=None,groups=chaos.datadoghq.com,resources=disruptions,verbs=create;update;delete,versions=v1beta1,name=vdisruption.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &Disruption{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Disruption) ValidateCreate() error {
	logger.Debugw("validating created disruption", "instance", r.Name, "namespace", r.Namespace)

	// delete-only mode, reject everything trying to be created
	if deleteOnly {
		return errors.New("the controller is currently in delete-only mode, you can't create new disruptions for now")
	}

	// reject disrputions with a name which would not be a valid label value
	// according to https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set
	if _, err := labels.Parse(fmt.Sprintf("name=%s", r.Name)); err != nil {
		return fmt.Errorf("invalid disruption name: %w", err)
	}

	// handle a disruption using the onInit feature without the handler being enabled
	if !handlerEnabled && r.Spec.OnInit {
		return errors.New("the chaos handler is disabled but the disruption onInit field is set to true, please enable the handler by specifying the --handler-enabled flag to the controller if you want to use the onInit feature (requires Kubernetes >= 1.15)")
	}

	if err := r.Spec.Validate(); err != nil {
		if mErr := metricsSink.MetricValidationFailed(r.getMetricsTags()); mErr != nil {
			logger.Errorw("error sending a metric", "error", mErr)
		}

		return err
	}

	// send validation metric
	if err := metricsSink.MetricValidationCreated(r.getMetricsTags()); err != nil {
		logger.Errorw("error sending a metric", "error", err)
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Disruption) ValidateUpdate(old runtime.Object) error {
	logger.Debugw("validating updated disruption", "instance", r.Name, "namespace", r.Namespace)

	// compare old and new disruption hashes and deny any spec changes
	oldHash, err := old.(*Disruption).Spec.Hash()
	if err != nil {
		return fmt.Errorf("error getting old disruption hash: %w", err)
	}

	newHash, err := r.Spec.Hash()
	if err != nil {
		return fmt.Errorf("error getting new disruption hash: %w", err)
	}

	logger.Debugw("comparing disruption spec hashes", "instance", r.Name, "namespace", r.Namespace, "oldHash", oldHash, "newHash", newHash)

	if oldHash != newHash {
		logger.Errorw("error when comparing disruption spec hashes", "instance", r.Name, "namespace", r.Namespace, "oldHash", oldHash, "newHash", newHash)
		return fmt.Errorf("a disruption spec can't be edited, please delete and recreate it if needed")
	}

	// send validation metric
	if err := metricsSink.MetricValidationUpdated(r.getMetricsTags()); err != nil {
		logger.Errorw("error sending a metric", "error", err)
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Disruption) ValidateDelete() error {
	// send validation metric
	if err := metricsSink.MetricValidationDeleted(r.getMetricsTags()); err != nil {
		logger.Errorw("error sending a metric", "error", err)
	}

	return nil
}

// getMetricsTags parses the disruption to generate metrics tags
func (r *Disruption) getMetricsTags() []string {
	tags := []string{
		"name:" + r.Name,
		"namespace:" + r.Namespace,
		"username:" + r.Status.UserInfo.Username,
	}

	// add groups
	for _, group := range r.Status.UserInfo.Groups {
		tags = append(tags, "group:"+group)
	}

	// add selectors
	for key, val := range r.Spec.Selector {
		tags = append(tags, fmt.Sprintf("selector:%s:%s", key, val))
	}

	// add kinds
	for _, kind := range r.Spec.GetKindNames() {
		tags = append(tags, "kind:"+string(kind))
	}

	return tags
}
