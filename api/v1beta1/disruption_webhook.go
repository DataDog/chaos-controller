/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	"errors"
	"fmt"

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

//+kubebuilder:webhook:webhookVersions={v1beta1},path=/mutate-chaos-datadoghq-com-v1beta1-disruption,mutating=true,failurePolicy=fail,sideEffects=None,groups=chaos.datadoghq.com,resources=disruptions,verbs=create;update,versions=v1beta1,name=mdisruption.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &Disruption{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Disruption) Default() {
	logger.Infow("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:webhookVersions={v1beta1},path=/validate-chaos-datadoghq-com-v1beta1-disruption,mutating=false,failurePolicy=fail,sideEffects=None,groups=chaos.datadoghq.com,resources=disruptions,verbs=create;update,versions=v1beta1,name=vdisruption.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &Disruption{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Disruption) ValidateCreate() error {
	logger.Infow("validating created disruption", "instance", r.Name, "namespace", r.Namespace)

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
