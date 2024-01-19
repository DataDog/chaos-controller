// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package v1beta1

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/DataDog/chaos-controller/cloudservice"
	cloudtypes "github.com/DataDog/chaos-controller/cloudservice/types"
	"github.com/DataDog/chaos-controller/ddmark"
	"github.com/DataDog/chaos-controller/utils"
	"go.opentelemetry.io/otel/trace"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/DataDog/chaos-controller/o11y/metrics"
	"github.com/DataDog/chaos-controller/o11y/tracer"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
	"k8s.io/api/authentication/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var (
	logger                        *zap.SugaredLogger
	k8sClient                     client.Client
	metricsSink                   metrics.Sink
	tracerSink                    tracer.Sink
	recorder                      record.EventRecorder
	deleteOnly                    bool
	enableSafemode                bool
	defaultNamespaceThreshold     float64
	defaultClusterThreshold       float64
	handlerEnabled                bool
	maxDuration                   time.Duration
	defaultDuration               time.Duration
	cloudServicesProvidersManager cloudservice.CloudServicesProvidersManager
	chaosNamespace                string
	ddmarkClient                  ddmark.Client
	safemodeEnvironment           string
)

const SafemodeEnvironmentAnnotation = GroupName + "/environment"

func (r *Disruption) SetupWebhookWithManager(setupWebhookConfig utils.SetupWebhookWithManagerConfig) error {
	var err error
	ddmarkClient, err = ddmark.NewClient(EmbeddedChaosAPI)

	if err != nil {
		return err
	}

	logger = &zap.SugaredLogger{}
	*logger = *setupWebhookConfig.Logger.With("source", "admission-controller")
	k8sClient = setupWebhookConfig.Manager.GetClient()
	metricsSink = setupWebhookConfig.MetricsSink
	tracerSink = setupWebhookConfig.TracerSink
	recorder = setupWebhookConfig.Recorder
	deleteOnly = setupWebhookConfig.DeleteOnlyFlag
	enableSafemode = setupWebhookConfig.EnableSafemodeFlag
	defaultNamespaceThreshold = float64(setupWebhookConfig.NamespaceThresholdFlag) / 100.0
	defaultClusterThreshold = float64(setupWebhookConfig.ClusterThresholdFlag) / 100.0
	handlerEnabled = setupWebhookConfig.HandlerEnabledFlag
	defaultDuration = setupWebhookConfig.DefaultDurationFlag
	maxDuration = setupWebhookConfig.MaxDurationFlag
	cloudServicesProvidersManager = setupWebhookConfig.CloudServicesProvidersManager
	chaosNamespace = setupWebhookConfig.ChaosNamespace
	safemodeEnvironment = setupWebhookConfig.Environment

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
	logger := logger.With("disruptionName", r.Name, "disruptionNamespace", r.Namespace)

	ctx, err := r.SpanContext(context.Background())
	if err != nil {
		logger.Errorw("did not find span context", "err", err)
	} else {
		logger = logger.With(tracerSink.GetLoggableTraceContext(trace.SpanFromContext(ctx))...)
	}

	logger.Infow("validating created disruption", "spec", r.Spec)

	// delete-only mode, reject everything trying to be created
	if deleteOnly {
		return errors.New("the controller is currently in delete-only mode, you can't create new disruptions for now")
	}

	// reject disruptions with a name which would not be a valid label value
	// according to https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set
	if _, err := labels.Parse(fmt.Sprintf("name=%s", r.Name)); err != nil {
		return fmt.Errorf("invalid disruption name: %w", err)
	}

	if safemodeEnvironment != "" {
		disruptionEnv, ok := r.Annotations[SafemodeEnvironmentAnnotation]
		if !ok {
			return fmt.Errorf("your disruption does not specify the environment it expects to run in, but this controller requires it to do. Set an annotation on this disruption with the key `%s` and the value `\"%s\"` to run in this kubernetes cluster. Be sure that you intend to run this disruption in %s",
				SafemodeEnvironmentAnnotation,
				safemodeEnvironment,
				safemodeEnvironment,
			)
		}

		if disruptionEnv != safemodeEnvironment {
			return fmt.Errorf("disruption is configured to run in \"%s\" but has been applied in \"%s\".  Set an annotation on this disruption with the key `%s` and the value `\"%s\"` to run in this kubernetes cluster, and double check your kubecontext is what you expect",
				disruptionEnv,
				safemodeEnvironment,
				SafemodeEnvironmentAnnotation,
				safemodeEnvironment,
			)
		}
	}

	// handle a disruption using the onInit feature without the handler being enabled
	if !handlerEnabled && r.Spec.OnInit {
		return errors.New("the chaos handler is disabled but the disruption onInit field is set to true, please enable the handler by specifying the --handler-enabled flag to the controller if you want to use the onInit feature (requires Kubernetes >= 1.15)")
	}

	if r.Spec.Network != nil {
		// this is the minimum estimated number of tc filters we could have for the disruption
		// knowing a service is filtered by both its service IP and the pod(s) IP where the service is
		// we don't count the number of Pods hosting the service here because this could be changing
		estimatedTcFiltersNb := len(r.Spec.Network.Hosts) + (len(r.Spec.Network.Services) * 2)

		if r.Spec.Network.Cloud != nil {
			clouds := r.Spec.Network.Cloud.TransformToCloudMap()

			for cloudName, serviceList := range clouds {
				serviceListNames := []string{}

				for _, service := range serviceList {
					serviceListNames = append(serviceListNames, service.ServiceName)
				}

				ipRangesPerService, err := cloudServicesProvidersManager.GetServicesIPRanges(cloudtypes.CloudProviderName(cloudName), serviceListNames)
				if err != nil {
					return err
				}

				for _, ipRanges := range ipRangesPerService {
					estimatedTcFiltersNb += len(ipRanges)
				}
			}
		}

		if estimatedTcFiltersNb > MaximumTCFilters {
			return fmt.Errorf("the number of resources (ips, ip ranges, single port) to filter is too high (%d). Please remove some hosts, services or cloud managed services to be affected in the disruption. Maximum resources (ips, ip ranges, single port) filterable is %d", estimatedTcFiltersNb, MaximumTCFilters)
		}
	}

	if err := r.Spec.Validate(); err != nil {
		if mErr := metricsSink.MetricValidationFailed(r.getMetricsTags()); mErr != nil {
			logger.Errorw("error sending a metric", "error", mErr)
		}

		return err
	}

	if r.Spec.Duration.Duration() > maxDuration {
		return fmt.Errorf("the maximum duration allowed is %s, please specify a duration lower or equal than this value", maxDuration)
	}

	multiErr := ddmarkClient.ValidateStructMultierror(r.Spec, "validation_webhook")
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
	recorder.Event(r, Events[EventDisruptionCreated].Type, string(EventDisruptionCreated), Events[EventDisruptionCreated].OnDisruptionTemplateMessage)

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Disruption) ValidateUpdate(old runtime.Object) error {
	logger := logger.With("disruptionName", r.Name, "disruptionNamespace", r.Namespace)
	logger.Debugw("validating updated disruption", "spec", r.Spec)

	var err error

	oldDisruption := old.(*Disruption)

	if err := r.validateUserInfo(oldDisruption); err != nil {
		return err
	}

	// ensure finalizer removal is only allowed if no related chaos pods exists
	// we should NOT always prevent finalizer removal because chaos controller reconcile loop will go through this mutating webhook when perfoming updates
	// and need to be able to remove the finalizer to enable the disruption to be garbage collected on successful removal
	if controllerutil.ContainsFinalizer(oldDisruption, chaostypes.DisruptionFinalizer) && !controllerutil.ContainsFinalizer(r, chaostypes.DisruptionFinalizer) {
		oldPods, err := GetChaosPods(context.Background(), logger, chaosNamespace, k8sClient, oldDisruption, nil)
		if err != nil {
			return fmt.Errorf("error getting disruption pods: %w", err)
		}

		if len(oldPods) != 0 {
			oldPodsInfos := []string{}
			for _, oldPod := range oldPods {
				oldPodsInfos = append(oldPodsInfos, fmt.Sprintf("%s/%s", oldPod.Namespace, oldPod.Name))
			}

			metricTags := append(r.getMetricsTags(), "prevent_finalizer_removal:true")
			if mErr := metricsSink.MetricValidationFailed(metricTags); mErr != nil {
				logger.Errorw("error sending a metric", "error", mErr)
			}

			return fmt.Errorf(`unable to remove disruption finalizer, disruption '%s/%s' still has associated pods:
- %s
You first need to remove those chaos pods (and potentially their finalizers) to be able to remove disruption finalizer`, oldDisruption.Namespace, oldDisruption.Name, strings.Join(oldPodsInfos, "\n- "))
		}
	}

	// compare old and new disruption hashes and deny any spec changes
	var oldHash, newHash string

	if oldDisruption.Spec.StaticTargeting {
		oldHash, err = oldDisruption.Spec.Hash()
		if err != nil {
			return fmt.Errorf("error getting old disruption hash: %w", err)
		}

		newHash, err = r.Spec.Hash()

		if err != nil {
			return fmt.Errorf("error getting new disruption hash: %w", err)
		}
	} else {
		oldHash, err = oldDisruption.Spec.HashNoCount()
		if err != nil {
			return fmt.Errorf("error getting old disruption hash: %w", err)
		}
		newHash, err = r.Spec.HashNoCount()
		if err != nil {
			return fmt.Errorf("error getting new disruption hash: %w", err)
		}
	}

	logger.Debugw("comparing disruption spec hashes", "oldHash", oldHash, "newHash", newHash)

	if oldHash != newHash {
		logger.Errorw("error when comparing disruption spec hashes", "oldHash", oldHash, "newHash", newHash)

		if oldDisruption.Spec.StaticTargeting {
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

func (r *Disruption) validateUserInfo(oldDisruption *Disruption) error {
	oldUserInfo, err := oldDisruption.UserInfo()
	if err != nil {
		return nil
	}

	emptyUserInfo := fmt.Sprintf("%v", v1beta1.UserInfo{})
	if fmt.Sprintf("%v", oldUserInfo) == emptyUserInfo {
		return nil
	}

	userInfo, err := r.UserInfo()
	if err != nil {
		return err
	}

	if fmt.Sprintf("%v", userInfo) != fmt.Sprintf("%v", oldUserInfo) {
		return fmt.Errorf("the user info annotation is immutable")
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
		"disruptionName:" + r.Name,
		"namespace:" + r.Namespace,
	}

	if userInfo, err := r.UserInfo(); !errors.Is(err, ErrNoUserInfo) {
		if err != nil {
			logger.Errorw("error retrieving user info from disruption, using empty user info", "error", err, "disruptionName", r.Name, "disruptionNamespace", r.Namespace)
		}

		tags = append(tags, "username:"+userInfo.Username)

		// add groups
		for _, group := range userInfo.Groups {
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
	for _, kind := range r.Spec.KindNames() {
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
		if caught, response, err := safetyNetCountNotTooLarge(r); err != nil {
			return nil, fmt.Errorf("error checking for countNotTooLarge safetynet: %w", err)
		} else if caught {
			logger.Debugw("the specified count represents a large percentage of targets in either the namespace or the kubernetes cluster", "SafetyNet Catch", "Generic")

			responses = append(responses, response)
		}

		if r.Spec.Network != nil {
			if caught := safetyNetNeitherHostNorPort(*r); caught {
				logger.Debugw("the specified disruption either contains no Hosts or contains a Host which has neither a port nor a host. The more ambiguous, the larger the blast radius.", "SafetyNet Catch", "Network")

				responses = append(responses, "the specified disruption either contains no Hosts or contains a Host which has neither a port nor a host. The more ambiguous, the larger the blast radius.")
			}
		}

		if r.Spec.DiskFailure != nil {
			if caught, response := safetyNetAllowRootDiskFailure(r); caught {
				logger.Debugw("the specified disruption contains an invalid path.", "SafetyNet Catch", "DiskFailure")

				responses = append(responses, response)
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
func safetyNetCountNotTooLarge(r *Disruption) (bool, string, error) {
	if r.Spec.Unsafemode != nil && r.Spec.Unsafemode.DisableCountTooLarge {
		return false, "", nil
	}

	userCount := r.Spec.Count
	totalCount := 0
	namespaceCount := 0
	targetCount := 0
	namespaceThreshold := defaultNamespaceThreshold
	clusterThreshold := defaultClusterThreshold

	if r.Spec.Unsafemode != nil {
		if r.Spec.Unsafemode.Config != nil && r.Spec.Unsafemode.Config.CountTooLarge != nil {
			if r.Spec.Unsafemode.Config.CountTooLarge.NamespaceThreshold != nil {
				namespaceThreshold = float64(*r.Spec.Unsafemode.Config.CountTooLarge.NamespaceThreshold) / 100.0
			}

			if r.Spec.Unsafemode.Config.CountTooLarge.ClusterThreshold != nil {
				clusterThreshold = float64(*r.Spec.Unsafemode.Config.CountTooLarge.ClusterThreshold) / 100.0
			}
		}
	}

	if r.Spec.Level == chaostypes.DisruptionLevelPod {
		pods := &corev1.PodList{}
		listOptions := &client.ListOptions{
			Namespace: r.ObjectMeta.Namespace,
			// In an effort not to fill up memory on huge list calls, limiting to 1000 objects per call
			Limit: 1000,
		}
		// we grab the number of pods in the specified namespace
		err := k8sClient.List(context.Background(), pods, listOptions)
		if err != nil {
			return false, "", fmt.Errorf("error listing namespace pods: %w", err)
		}

		for pods.Continue != "" {
			namespaceCount += len(pods.Items)
			listOptions.Continue = pods.Continue

			err = k8sClient.List(context.Background(), pods, listOptions)
			if err != nil {
				return false, "", fmt.Errorf("error listing target pods: %w", err)
			}
		}

		namespaceCount = len(pods.Items)

		listOptions = &client.ListOptions{
			LabelSelector: labels.SelectorFromValidatedSet(r.Spec.Selector),
		}
		// we grab the number of targets in the specified namespace
		err = k8sClient.List(context.Background(), pods, listOptions)
		if err != nil {
			return false, "", fmt.Errorf("error listing target pods: %w", err)
		}

		for pods.Continue != "" {
			targetCount += len(pods.Items)
			listOptions.Continue = pods.Continue

			err = k8sClient.List(context.Background(), pods, listOptions)
			if err != nil {
				return false, "", fmt.Errorf("error listing target pods: %w", err)
			}
		}

		targetCount = len(pods.Items)

		// we grab the number of pods in the entire cluster
		err = k8sClient.List(context.Background(), pods,
			client.Limit(1000))
		if err != nil {
			return false, "", fmt.Errorf("error listing cluster pods: %w", err)
		}

		for pods.Continue != "" {
			totalCount += len(pods.Items)

			err = k8sClient.List(context.Background(), pods, client.Limit(1000), client.Continue(pods.Continue))
			if err != nil {
				return false, "", fmt.Errorf("error listing target pods: %w", err)
			}
		}

		totalCount = len(pods.Items)
	} else {
		nodes := &corev1.NodeList{}

		err := k8sClient.List(context.Background(), nodes,
			client.Limit(1000))
		if err != nil {
			return false, "", fmt.Errorf("error listing target pods: %w", err)
		}

		for nodes.Continue != "" {
			totalCount += len(nodes.Items)

			err = k8sClient.List(context.Background(), nodes, client.Limit(1000), client.Continue(nodes.Continue))
			if err != nil {
				return false, "", fmt.Errorf("error listing target pods: %w", err)
			}
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

	// we check to see if the count represents > namespaceThreshold (default 80) percent of all pods in the existing namespace
	// or if the count represents > clusterThreshold (default 66) percent of all pods in the cluster
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

// safetyNetAllowRootDiskFailure is the safety net regarding missing path or invalid path values for a disk failure disruption.
func safetyNetAllowRootDiskFailure(r *Disruption) (bool, string) {
	if r.Spec.Unsafemode != nil && r.Spec.Unsafemode.AllowRootDiskFailure {
		return false, ""
	}

	if r.Spec.Level != chaostypes.DisruptionLevelNode {
		return false, ""
	}

	for _, path := range r.Spec.DiskFailure.Paths {
		if strings.TrimSpace(path) == "/" {
			return true, "the specified path for the disk failure disruption targeting a node must not be \"/\"."
		}
	}

	return false, ""
}
