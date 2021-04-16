/*

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
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
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

// +kubebuilder:webhook:verbs=create;update,path=/validate-chaos-datadoghq-com-v1beta1-disruption,mutating=false,failurePolicy=fail,groups=chaos.datadoghq.com,resources=disruptions,versions=v1beta1,name=vdisruption.kb.io

var _ webhook.Validator = &Disruption{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Disruption) ValidateCreate() error {
	logger.Infow("validating created disruption", "instance", r.Name, "namespace", r.Namespace)

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Disruption) ValidateUpdate(old runtime.Object) error {
	logger.Infow("validating updated disruption", "instance", r.Name, "namespace", r.Namespace)

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Disruption) ValidateDelete() error {
	return nil
}
