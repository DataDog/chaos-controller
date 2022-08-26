// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package v1beta1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/DataDog/chaos-controller/cloudservice"
	"github.com/DataDog/chaos-controller/ddmark"
	"github.com/DataDog/chaos-controller/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cloudtypes "github.com/DataDog/chaos-controller/cloudservice/types"
	"github.com/DataDog/chaos-controller/metrics"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
	v1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var logger *zap.SugaredLogger
var k8sClient client.Client
var metricsSink metrics.Sink
var recorder record.EventRecorder
var deleteOnly bool
var enableSafemode bool
var namespaceThreshold float64
var clusterThreshold float64
var handlerEnabled bool
var defaultDuration time.Duration
var cloudProviderManager *cloudservice.CloudProviderManager

func (r *Disruption) SetupWebhookWithManager(setupWebhookConfig utils.SetupWebhookWithManagerConfig) error {
	if err := ddmark.InitLibrary(EmbeddedChaosAPI, chaostypes.DDMarkChaoslibPrefix); err != nil {
		return err
	}

	logger = &zap.SugaredLogger{}
	*logger = *setupWebhookConfig.Logger.With("source", "admission-controller")
	k8sClient = setupWebhookConfig.Manager.GetClient()
	metricsSink = setupWebhookConfig.MetricsSink
	recorder = setupWebhookConfig.Recorder
	deleteOnly = setupWebhookConfig.DeleteOnlyFlag
	enableSafemode = setupWebhookConfig.EnableSafemodeFlag
	namespaceThreshold = float64(setupWebhookConfig.NamespaceThresholdFlag) / 100.0
	clusterThreshold = float64(setupWebhookConfig.ClusterThresholdFlag) / 100.0
	handlerEnabled = setupWebhookConfig.HandlerEnabledFlag
	defaultDuration = setupWebhookConfig.DefaultDurationFlag
	cloudProviderManager = setupWebhookConfig.CloudProviderManager

	return ctrl.NewWebhookManagedBy(setupWebhookConfig.Manager).
		For(r).
		Complete()
}

//+kubebuilder:webhook:webhookVersions={v1},path=/mutate-chaos-datadoghq-com-v1beta1-disruption,mutating=true,failurePolicy=fail,sideEffects=None,groups=chaos.datadoghq.com,resources=disruptions,verbs=create;update,versions=v1beta1,name=mdisruption.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &Disruption{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Disruption) Default() {
	if r.Spec.Duration.Duration() == 0 {
		logger.Infow(fmt.Sprintf("setting default duration of %s in disruption", defaultDuration), "instance", r.Name, "namespace", r.Namespace)
		r.Spec.Duration = DisruptionDuration(defaultDuration.String())
	}
}

//+kubebuilder:webhook:webhookVersions={v1},path=/validate-chaos-datadoghq-com-v1beta1-disruption,mutating=false,failurePolicy=fail,sideEffects=None,groups=chaos.datadoghq.com,resources=disruptions,verbs=create;update;delete,versions=v1beta1,name=vdisruption.kb.io,admissionReviewVersions={v1,v1beta1}

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

	if r.Spec.Network.Cloud != nil {
		clouds := map[cloudtypes.CloudProviderName][]string{
			cloudtypes.CloudProviderAWS:     r.Spec.Network.Cloud.AWS,
			cloudtypes.CloudProviderDatadog: r.Spec.Network.Cloud.Datadog,
			cloudtypes.CloudProviderGCP:     r.Spec.Network.Cloud.GCP,
		}

		for cloudName, serviceList := range clouds {
			for _, service := range serviceList {
				if !cloudProviderManager.ServiceExists(cloudName, service) {
					return fmt.Errorf("service %s of %s does not exist. Available are: %s", service, cloudName, strings.Join(cloudProviderManager.GetServiceList(cloudName), ", "))
				}
			}
		}
	}

	if err := r.Spec.Validate(); err != nil {
		if mErr := metricsSink.MetricValidationFailed(r.getMetricsTags()); mErr != nil {
			logger.Errorw("error sending a metric", "error", mErr)
		}

		return err
	}

	multiErr := ddmark.ValidateStructMultierror(r.Spec, "validation_webhook", chaostypes.DDMarkChaoslibPrefix)
	if multiErr.ErrorOrNil() != nil {
		return multierror.Prefix(multiErr, "ddmark: ")
	}

	// handle initial safety nets
	if enableSafemode {
		if responses, err := r.initialSafetyNets(); err != nil {
			return err
		} else if len(responses) > 0 {
			retErr := errors.New("at least one of the initial safety nets caught an issue")
			for _, response := range responses {
				retErr = multierror.Append(retErr, errors.New(response))
			}
			return retErr
		}
	}

	// send validation metric
	if err := metricsSink.MetricValidationCreated(r.getMetricsTags()); err != nil {
		logger.Errorw("error sending a metric", "error", err)
	}

	// send informative event to disruption to broadcast
	recorder.Event(r, Events[EventDisruptionCreated].Type, EventDisruptionCreated, Events[EventDisruptionCreated].OnDisruptionTemplateMessage)

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Disruption) ValidateUpdate(old runtime.Object) error {
	logger.Debugw("validating updated disruption", "instance", r.Name, "namespace", r.Namespace)

	// compare old and new disruption hashes and deny any spec changes
	var oldHash, newHash string

	var err error

	if r.Spec.StaticTargeting {
		oldHash, err = old.(*Disruption).Spec.Hash()
		if err != nil {
			return fmt.Errorf("error getting old disruption hash: %w", err)
		}

		newHash, err = r.Spec.Hash()

		if err != nil {
			return fmt.Errorf("error getting new disruption hash: %w", err)
		}
	} else {
		oldHash, err = old.(*Disruption).Spec.HashNoCount()
		if err != nil {
			return fmt.Errorf("error getting old disruption hash: %w", err)
		}
		newHash, err = r.Spec.HashNoCount()
		if err != nil {
			return fmt.Errorf("error getting new disruption hash: %w", err)
		}
	}

	logger.Debugw("comparing disruption spec hashes", "instance", r.Name, "namespace", r.Namespace, "oldHash", oldHash, "newHash", newHash)

	if oldHash != newHash {
		logger.Errorw("error when comparing disruption spec hashes", "instance", r.Name, "namespace", r.Namespace, "oldHash", oldHash, "newHash", newHash)

		if r.Spec.StaticTargeting {
			return fmt.Errorf("[StaticTargeting: true] a disruption spec cannot be updated, please delete and recreate it if needed")
		}

		return fmt.Errorf("[StaticTargeting: false] only a disruption spec's Count field can be updated, please delete and recreate it if needed")
	}

	if err := r.Spec.Validate(); err != nil {
		if mErr := metricsSink.MetricValidationFailed(r.getMetricsTags()); mErr != nil {
			logger.Errorw("error sending a metric", "error", mErr)
		}

		return err
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
	}

	if _, ok := r.Annotations["UserInfo"]; ok {
		var annotation v1.UserInfo

		err := json.Unmarshal([]byte(r.Annotations["UserInfo"]), &annotation)
		if err != nil {
			logger.Errorw("Error decoding annotation", err)
		}

		tags = append(tags, "username:"+annotation.Username)

		// add groups
		for _, group := range annotation.Groups {
			tags = append(tags, "group:"+group)
		}
	}

	// add selectors
	for key, val := range r.Spec.Selector {
		tags = append(tags, fmt.Sprintf("selector:%s:%s", key, val))
	}

	for _, lsr := range r.Spec.AdvancedSelector {
		value := ""
		if lsr.Operator == metav1.LabelSelectorOpIn || lsr.Operator == metav1.LabelSelectorOpNotIn {
			value = fmt.Sprintf(":%s", lsr.Values)
		}

		tags = append(tags, fmt.Sprintf("selector:%s:%s%s", lsr.Key, lsr.Operator, value))
	}

	// add kinds
	for _, kind := range r.Spec.GetKindNames() {
		tags = append(tags, "kind:"+string(kind))
	}

	return tags
}

// initialSafetyNets runs the initial safety nets for any new disruption
// returns a list of responses related to safety net catches if any safety net were caught and returns any errors when attempting to run the safety nets
func (r *Disruption) initialSafetyNets() ([]string, error) {
	responses := []string{}
	// handle initial safety nets if safemode is enabled
	if r.Spec.Unsafemode == nil || !r.Spec.Unsafemode.DisableAll {
		if caught, response, err := safetyNetCountNotTooLarge(*r); err != nil {
			return nil, fmt.Errorf("error checking for countNotTooLarge safetynet: %w", err)
		} else if caught {
			logger.Debugw("the specified count represents a large percentage of targets in either the namespace or the kubernetes cluster", r.Name, "SafetyNet Catch", "Generic")

			responses = append(responses, response)
		}

		if r.Spec.Network != nil {
			if caught := safetyNetNeitherHostNorPort(*r); caught {
				logger.Debugw("The specified disruption either contains no Hosts or contains a Host which has neither a port or a host. The more ambiguous, the larger the blast radius.", r.Name, "SafetyNet Catch", "Network")

				responses = append(responses, "The specified disruption either contains no Hosts or contains a Host which has neither a port or a host. The more ambiguous, the larger the blast radius.")
			}
		}
	}

	return responses, nil
}

// safetyNetCountNotTooLarge is the safety net regarding the count of targets
// it will check against the number of targets being targeted and the number of targets in the k8s system
// > 66% of the k8s system being targeted warrants a safety check if we assume each of our targets are replicated
// at least twice. > 80% in a namespace also warrants a safety check as namespaces may be shared between services.
// returning true indicates the safety net caught something
func safetyNetCountNotTooLarge(r Disruption) (bool, string, error) {
	if r.Spec.Unsafemode != nil && r.Spec.Unsafemode.DisableCountTooLarge {
		return false, "", nil
	}

	userCount := r.Spec.Count
	totalCount := 0
	namespaceCount := 0
	targetCount := 0

	if r.Spec.Unsafemode != nil {
		if r.Spec.Unsafemode.Config != nil && r.Spec.Unsafemode.Config.CountTooLarge != nil {
			if r.Spec.Unsafemode.Config.CountTooLarge.NamespaceThreshold != 0 {
				namespaceThreshold = float64(r.Spec.Unsafemode.Config.CountTooLarge.NamespaceThreshold) / 100.0
			}

			if r.Spec.Unsafemode.Config.CountTooLarge.ClusterThreshold != 0 {
				clusterThreshold = float64(r.Spec.Unsafemode.Config.CountTooLarge.ClusterThreshold) / 100.0
			}
		}
	}

	if r.Spec.Level == chaostypes.DisruptionLevelPod {
		pods := &corev1.PodList{}
		listOptions := &client.ListOptions{
			Namespace: r.ObjectMeta.Namespace,
		}

		err := k8sClient.List(context.Background(), pods, listOptions)
		if err != nil {
			return false, "", fmt.Errorf("error listing namespace pods: %w", err)
		}

		namespaceCount = len(pods.Items)

		listOptions = &client.ListOptions{
			Namespace:     r.ObjectMeta.Namespace,
			LabelSelector: labels.SelectorFromValidatedSet(r.Spec.Selector),
		}

		err = k8sClient.List(context.Background(), pods, listOptions)
		if err != nil {
			return false, "", fmt.Errorf("error listing target pods: %w", err)
		}

		targetCount = len(pods.Items)

		err = k8sClient.List(context.Background(), pods)
		if err != nil {
			return false, "", fmt.Errorf("error listing cluster pods: %w", err)
		}

		totalCount = len(pods.Items)
	} else {
		nodes := &corev1.NodeList{}

		err := k8sClient.List(context.Background(), nodes)
		if err != nil {
			return false, "", fmt.Errorf("error listing target pods: %w", err)
		}

		totalCount = len(nodes.Items)
	}

	userCountVal := 0.0

	userCountInt, isPercent, err := GetIntOrPercentValueSafely(userCount)
	if err != nil {
		return false, "", fmt.Errorf("failed to get count: %w", err)
	}

	if targetCount == 0 {
		return false, "", nil
	}

	if isPercent {
		userCountVal = float64(userCountInt) / 100.0 * float64(targetCount)
	} else {
		userCountVal = float64(userCountInt)
	}

	// we check to see if the count represents > 80 percent of all pods in the existing namepsace
	// or if the count represents > 66 percent of all pods in the cluster
	if r.Spec.Level != chaostypes.DisruptionLevelNode {
		if userNamespacePercent := userCountVal / float64(namespaceCount); userNamespacePercent > namespaceThreshold {
			response := fmt.Sprintf("target selection represents %.2f %% of the total pods in the namespace while the threshold is %.2f %%", userNamespacePercent*100, namespaceThreshold*100)
			return true, response, nil
		}
	}

	if userTotalPercent := userCountVal / float64(totalCount); userTotalPercent > clusterThreshold {
		response := fmt.Sprintf("target selection represents %.2f %% of the total %ss in the cluster while the threshold is %.2f %%", userTotalPercent*100, r.Spec.Level, clusterThreshold*100)
		return true, response, nil
	}

	return false, "", nil
}

// safetyNetNeitherHostNorPort is the safety net regarding missing host and port values.
// it will check against all defined hosts in the network disruption spec to see if any of them have a host and a
// port missing. The more generic a hosts tuple is (Omitting fields such as port), the bigger the blast radius.
func safetyNetNeitherHostNorPort(r Disruption) bool {
	if r.Spec.Unsafemode != nil && r.Spec.Unsafemode.DisableNeitherHostNorPort {
		return false
	}

	// if hosts are not defined, this also falls into the safety net
	if r.Spec.Network.Hosts == nil || len(r.Spec.Network.Hosts) == 0 {
		return true
	}

	for _, host := range r.Spec.Network.Hosts {
		if host.Port == 0 && host.Host == "" {
			return true
		}
	}

	return false
}
