// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package v1beta1

import (
	"fmt"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var logger *zap.SugaredLogger

func (r *Disruption) SetupWebhookWithManager(mgr ctrl.Manager, l *zap.SugaredLogger) error {
	logger = &zap.SugaredLogger{}
	*logger = *l.With("source", "admission-controller")

	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-chaos-datadoghq-com-v1beta1-disruption,mutating=false,failurePolicy=fail,groups=chaos.datadoghq.com,resources=disruptions,versions=v1beta1,name=chaos-controller-admission-webhook.chaos-engineering.svc

var _ webhook.Validator = &Disruption{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Disruption) ValidateCreate() error {
	logger.Infow("validating created disruption", "instance", r.Name, "namespace", r.Namespace)

	if r.Spec.Network != nil && r.Spec.Network.Flow == FlowIngress && len(r.Spec.Network.Hosts) > 0 {
		return fmt.Errorf("a network disruption should not specify a hosts list when targeting ingress packets")
	}

	if r.Spec.Count.Type == intstr.Int && r.Spec.Count.IntValue() < 0 {
		return fmt.Errorf("count must be a positive integer or a percentage value")
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
