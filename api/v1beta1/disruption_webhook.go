// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package v1beta1

import (
	"errors"
	"fmt"

	"github.com/DataDog/chaos-controller/metrics"
	"go.uber.org/zap"
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

func (r *Disruption) SetupWebhookWithManager(mgr ctrl.Manager, l *zap.SugaredLogger, ms metrics.Sink, deleteOnlyFlag, handlerEnabledFlag bool) error {
	logger = &zap.SugaredLogger{}
	*logger = *l.With("source", "admission-controller")
	k8sClient = mgr.GetClient()
	metricsSink = ms
	deleteOnly = deleteOnlyFlag
	handlerEnabled = handlerEnabledFlag

	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

var _ webhook.Validator = &Disruption{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Disruption) ValidateCreate() error {
	logger.Infow("validating created disruption", "instance", r.Name, "namespace", r.Namespace)

	if deleteOnly {
		return errors.New("the controller is currently in delete-only mode, you can't create new disruptions for now")
	}

	// handle a disruption using the onInit feature without the handler being enabled
	if !handlerEnabled && r.Spec.OnInit {
		return errors.New("the chaos handler is disabled but the disruption onInit field is set to true, please enable the handler by specifying the --handler-enabled flag to the controller if you want to use the onInit feature (requires Kubernetes >= 1.15)")
	}

	if err := r.Spec.Validate(); err != nil {
		mErr := metricsSink.MetricFailedValidation()

		if mErr != nil {
			logger.Errorw("error sending a metric", "error", mErr)
		}

		return err
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Disruption) ValidateUpdate(old runtime.Object) error {
	logger.Infow("validating updated disruption", "instance", r.Name, "namespace", r.Namespace)

	// compare old and new disruption hashes and deny any spec changes
	oldHash, err := old.(*Disruption).Spec.Hash()
	if err != nil {
		return fmt.Errorf("error getting old disruption hash: %w", err)
	}

	newHash, err := r.Spec.Hash()
	if err != nil {
		return fmt.Errorf("error getting new disruption hash: %w", err)
	}

	logger.Infow("comparing disruption spec hashes", "instance", r.Name, "namespace", r.Namespace, "oldHash", oldHash, "newHash", newHash)

	if oldHash != newHash {
		return fmt.Errorf("a disruption spec can't be edited, please delete and recreate it if needed")
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Disruption) ValidateDelete() error {
	return nil
}
