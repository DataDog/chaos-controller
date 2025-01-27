// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package config

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"

	cloudtypes "github.com/DataDog/chaos-controller/cloudservice/types"
	"github.com/DataDog/chaos-controller/eventnotifier"
)

type config struct {
	Controller controllerConfig `json:"controller" yaml:"controller"`
	Injector   injectorConfig   `json:"injector" yaml:"injector"`
	Handler    handlerConfig    `json:"handler" yaml:"handler"`
}

type controllerConfig struct {
	HealthProbeBindAddr              string                          `json:"healthProbeBindAddr" yaml:"healthProbeBindAddr"`
	MetricsBindAddr                  string                          `json:"metricsBindAddr" yaml:"metricsBindAddr"`
	MetricsSink                      string                          `json:"metricsSink" yaml:"metricsSink"`
	ExpiredDisruptionGCDelay         time.Duration                   `json:"expiredDisruptionGCDelay" yaml:"expiredDisruptionGCDelay"`
	MaxDuration                      time.Duration                   `json:"maxDuration,omitempty" yaml:"maxDuration,omitempty"`
	DefaultDuration                  time.Duration                   `json:"defaultDuration" yaml:"defaultDuration"`
	DefaultCronDelayedStartTolerance time.Duration                   `json:"defaultCronDelayedStartTolerance" yaml:"defaultCronDelayedStartTolerance"`
	MinimumCronFrequency             time.Duration                   `json:"minimumCronFrequency" yaml:"minimumCronFrequency"`
	DeleteOnly                       bool                            `json:"deleteOnly" yaml:"deleteOnly"`
	EnableSafeguards                 bool                            `json:"enableSafeguards" yaml:"enableSafeguards"`
	EnableObserver                   bool                            `json:"enableObserver" yaml:"enableObserver"`
	LeaderElection                   bool                            `json:"leaderElection" yaml:"leaderElection"`
	Webhook                          controllerWebhookConfig         `json:"webhook" yaml:"webhook"`
	Notifiers                        eventnotifier.NotifiersConfig   `json:"notifiers" yaml:"notifiers"`
	CloudProviders                   cloudtypes.CloudProviderConfigs `json:"cloudProviders" yaml:"cloudProviders"`
	UserInfoHook                     bool                            `json:"userInfoHook" yaml:"userInfoHook"`
	SafeMode                         safeModeConfig                  `json:"safeMode" yaml:"safeMode"`
	ProfilerSink                     string                          `json:"profilerSink" yaml:"profilerSink"`
	TracerSink                       string                          `json:"tracerSink" yaml:"tracerSink"`
	DisruptionCronEnabled            bool                            `json:"disruptionCronEnabled" yaml:"disruptionCronEnabled"`
	DisruptionRolloutEnabled         bool                            `json:"disruptionRolloutEnabled" yaml:"disruptionRolloutEnabled"`
	DisruptionDeletionTimeout        time.Duration                   `json:"disruptionDeletionTimeout" yaml:"disruptionDeletionTimeout"`
	FinalizerDeletionDelay           time.Duration                   `json:"finalizerDeletionDelay" yaml:"finalizerDeletionDelay"`
	TargetResourceMissingThreshold   time.Duration                   `json:"targetResourceMissingThreshold" yaml:"targetResourceMissingThreshold"`
	DisabledDisruptions              []string                        `json:"disabledDisruptions" yaml:"disabledDisruptions"`
}

type controllerWebhookConfig struct {
	CertDir string `json:"certDir" yaml:"certDir"`
	Host    string `json:"host" yaml:"host"`
	Port    int    `json:"port" yaml:"port"`
}

type safeModeConfig struct {
	Environment         string   `json:"environment" yaml:"environment"`
	PermittedUserGroups []string `json:"permittedUserGroups" yaml:"permittedUserGroups"`
	Enable              bool     `json:"enable" yaml:"enable"`
	NamespaceThreshold  int      `json:"namespaceThreshold" yaml:"namespaceThreshold"`
	ClusterThreshold    int      `json:"clusterThreshold" yaml:"clusterThreshold"`
	AllowNodeFailure    bool     `json:"allowNodeFailure" yaml:"allowNodeFailure"`
	AllowNodeLevel      bool     `json:"allowNodeLevel" yaml:"allowNodeLevel"`
}

type injectorConfig struct {
	Image             string                          `json:"image" yaml:"image"`
	Annotations       map[string]string               `json:"annotations" yaml:"annotations"`
	Labels            map[string]string               `json:"labels" yaml:"labels"`
	ChaosNamespace    string                          `json:"chaosNamespace" yaml:"chaosNamespace"`
	ServiceAccount    string                          `json:"serviceAccount" yaml:"serviceAccount"`
	DNSDisruption     injectorDNSDisruptionConfig     `json:"dnsDisruption" yaml:"dnsDisruption"`
	NetworkDisruption injectorNetworkDisruptionConfig `json:"networkDisruption" yaml:"networkDisruption"`
	ImagePullSecrets  string                          `json:"imagePullSecrets" yaml:"imagePullSecrets"`
	Tolerations       []Toleration                    `json:"tolerations" yaml:"tolerations,omitempty"`
	LogLevel          string                          `json:"logLevel" yaml:"logLevel"`
}

type Toleration struct {
	Key               string `json:"key" yaml:"key"`
	Operator          string `json:"operator" yaml:"operator"`
	Value             string `json:"value" yaml:"value"`
	Effect            string `json:"effect" yaml:"effect"`
	TolerationSeconds *int64 `json:"tolerationSeconds,omitempty" yaml:"tolerationSeconds,omitempty"`
}

type injectorDNSDisruptionConfig struct {
	DNSServer string `json:"dnsServer" yaml:"dnsServer"`
	KubeDNS   string `json:"kubeDns" yaml:"kubeDns"`
}

type injectorNetworkDisruptionConfig struct {
	AllowedHosts        []string      `json:"allowedHosts" yaml:"allowedHosts"`
	HostResolveInterval time.Duration `json:"hostResolveInterval" yaml:"hostResolveInterval"`
}

type handlerConfig struct {
	Enabled    bool          `json:"enabled" yaml:"enabled"`
	Image      string        `json:"image" yaml:"image"`
	Timeout    time.Duration `json:"timeout" yaml:"timeout"`
	MaxTimeout time.Duration `json:"maxTimeout" yaml:"maxTimeout"`
	CPU        string        `json:"cpu" yaml:"cpu"`
	Memory     string        `json:"memory" yaml:"memory"`
}

const DefaultDisruptionDeletionTimeout = time.Minute * 15
const DefaultFinalizerDeletionDelay = time.Second * 20

func New(client corev1client.ConfigMapInterface, logger *zap.SugaredLogger, osArgs []string) (config, error) {
	var (
		configPath         string
		configMapOverrides string
		cfg                config
	)

	preConfigFS := pflag.NewFlagSet("pre-config", pflag.ContinueOnError)
	mainFS := pflag.NewFlagSet("main-config", pflag.ContinueOnError)

	preConfigFS.ParseErrorsWhitelist.UnknownFlags = true
	preConfigFS.StringVar(&configPath, "config", "", "Configuration file path")
	preConfigFS.StringVar(&configMapOverrides, "config-overrides", "", "Name of ConfigMap to provide config overrides")
	// we redefine configuration flag into main flag to avoid removing it manually from provided args
	// we also define it to avoid activating "UnknownFlags" for main flags so we'll return an error in case a flag is unknown
	mainFS.StringVar(&configPath, "config", "", "Configuration file path")
	mainFS.StringVar(&configMapOverrides, "config-overrides", "", "Name of ConfigMap to provide config overrides")

	mainFS.StringVar(&cfg.Controller.MetricsBindAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")

	mainFS.StringVar(&cfg.Controller.HealthProbeBindAddr, "health-probe-bind-address", "0.0.0.0:8081", "The address the health probe endpoint binds to.")

	if err := viper.BindPFlag("controller.metricsBindAddr", mainFS.Lookup("metrics-bind-address")); err != nil {
		return cfg, err
	}

	mainFS.BoolVar(&cfg.Controller.LeaderElection, "leader-elect", false, "Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")

	if err := viper.BindPFlag("controller.leaderElection", mainFS.Lookup("leader-elect")); err != nil {
		return cfg, err
	}

	mainFS.BoolVar(&cfg.Controller.DeleteOnly, "delete-only", false,
		"Enable delete only mode which will not allow new disruption to start and will clean up and remove existing disruptions.")

	if err := viper.BindPFlag("controller.deleteOnly", mainFS.Lookup("delete-only")); err != nil {
		return cfg, err
	}

	mainFS.BoolVar(&cfg.Controller.EnableSafeguards, "enable-safeguards", true, "Enable safeguards on target selection")

	if err := viper.BindPFlag("controller.enableSafeguards", mainFS.Lookup("enable-safeguards")); err != nil {
		return cfg, err
	}

	mainFS.BoolVar(&cfg.Controller.EnableObserver, "enable-observer", true, "Enable observer on targets")

	if err := viper.BindPFlag("controller.enableObserver", mainFS.Lookup("enable-observer")); err != nil {
		return cfg, err
	}

	mainFS.DurationVar(&cfg.Controller.ExpiredDisruptionGCDelay, "expired-disruption-gc-delay", time.Minute*(-1), "Duration after a disruption expires before being automatically deleted, leave unset to disable")

	if err := viper.BindPFlag("controller.expiredDisruptionGCDelay", mainFS.Lookup("expired-disruption-gc-delay")); err != nil {
		return cfg, err
	}

	mainFS.DurationVar(&cfg.Controller.DefaultDuration, "default-duration", time.Hour, "Default duration for a disruption with none specified")

	if err := viper.BindPFlag("controller.defaultDuration", mainFS.Lookup("default-duration")); err != nil {
		return cfg, err
	}

	mainFS.DurationVar(&cfg.Controller.MaxDuration, "max-duration", 0, "Max duration for a disruption to timeout")

	if err := viper.BindPFlag("controller.maxDuration", mainFS.Lookup("max-duration")); err != nil {
		return cfg, err
	}

	mainFS.DurationVar(&cfg.Controller.DefaultCronDelayedStartTolerance, "default-cron-delayed-start-tolerance", time.Minute*5, "Default deadline for starting a new disruption after the disruption cron's scheduled time")

	if err := viper.BindPFlag("controller.defaultCronDelayedStartTolerance", mainFS.Lookup("default-cron-delayed-start-tolerance")); err != nil {
		return cfg, err
	}

	mainFS.DurationVar(&cfg.Controller.MinimumCronFrequency, "minimum-cron-frequency", time.Minute, "Minimum frequency for a disruption cron schedule")

	if err := viper.BindPFlag("controller.minimumCronFrequency", mainFS.Lookup("minimum-cron-frequency")); err != nil {
		return cfg, err
	}

	mainFS.StringVar(&cfg.Controller.Notifiers.Common.ClusterName, "notifiers-common-clustername", "", "Cluster Name for notifiers output")

	if err := viper.BindPFlag("controller.notifiers.common.clusterName", mainFS.Lookup("notifiers-common-clustername")); err != nil {
		return cfg, err
	}

	mainFS.BoolVar(&cfg.Controller.Notifiers.Noop.Enabled, "notifiers-noop-enabled", false, "Enabler toggle for the NOOP notifier (defaulted to false)")

	if err := viper.BindPFlag("controller.notifiers.noop.enabled", mainFS.Lookup("notifiers-noop-enabled")); err != nil {
		return cfg, err
	}

	mainFS.BoolVar(&cfg.Controller.Notifiers.Slack.Enabled, "notifiers-slack-enabled", false, "Enabler toggle for the Slack notifier (defaulted to false)")

	if err := viper.BindPFlag("controller.notifiers.slack.enabled", mainFS.Lookup("notifiers-slack-enabled")); err != nil {
		return cfg, err
	}

	mainFS.StringVar(&cfg.Controller.Notifiers.Slack.TokenFilepath, "notifiers-slack-tokenfilepath", "", "File path of the API token for the Slack notifier (defaulted to empty string)")

	if err := viper.BindPFlag("controller.notifiers.slack.tokenFilepath", mainFS.Lookup("notifiers-slack-tokenfilepath")); err != nil {
		return cfg, err
	}

	mainFS.StringVar(&cfg.Controller.Notifiers.Slack.MirrorSlackChannelID, "notifiers-slack-mirrorslackchannelid", "", "Slack Channel ID to send all the slack notifier notifications in addition to the personal messages (defaulted to empty string)")

	if err := viper.BindPFlag("controller.notifiers.slack.mirrorSlackChannelId", mainFS.Lookup("notifiers-slack-mirrorslackchannelid")); err != nil {
		return cfg, err
	}

	mainFS.BoolVar(&cfg.Controller.Notifiers.Datadog.Enabled, "notifiers-datadog-enabled", false, "Enabler toggle for the Datadog notifier (defaulted to false)")

	if err := viper.BindPFlag("controller.notifiers.datadog.enabled", mainFS.Lookup("notifiers-datadog-enabled")); err != nil {
		return cfg, err
	}

	mainFS.StringArrayVar(&cfg.Controller.Notifiers.HTTP.Headers, "notifiers-http-headers", []string{}, "Additional headers to add to the request when sending the notification (defaulted to empty list)")

	if err := viper.BindPFlag("controller.notifiers.http.headers", mainFS.Lookup("notifiers-http-headers")); err != nil {
		return cfg, err
	}

	mainFS.StringVar(&cfg.Controller.Notifiers.HTTP.HeadersFilepath, "notifiers-http-headers-filepath", "", "Filepath to the additional headers to add to the request when sending the notification (defaulted to empty list)")

	if err := viper.BindPFlag("controller.notifiers.http.headersFilepath", mainFS.Lookup("notifiers-http-headers-filepath")); err != nil {
		return cfg, err
	}

	mainFS.StringVar(&cfg.Controller.Notifiers.HTTP.AuthURL, "notifiers-http-auth-url", "", "WARNING/ALPHA: First perform an HTTP request to dynamically retrieve auth information before sending http notification")

	if err := viper.BindPFlag("controller.notifiers.http.authURL", mainFS.Lookup("notifiers-http-auth-url")); err != nil {
		return cfg, err
	}

	mainFS.StringSliceVar(&cfg.Controller.Notifiers.HTTP.AuthHeaders, "notifiers-http-auth-headers", []string{}, "WARNING/ALPHA: HTTP headers to provide to auth request")

	if err := viper.BindPFlag("controller.notifiers.http.authHeaders", mainFS.Lookup("notifiers-http-auth-headers")); err != nil {
		return cfg, err
	}

	mainFS.StringVar(&cfg.Controller.Notifiers.HTTP.AuthTokenPath, "notifiers-http-auth-token-path", "", "WARNING/ALPHA: Extract bearer token from provided JSON path (using GJSON)")

	if err := viper.BindPFlag("controller.notifiers.http.authTokenPath", mainFS.Lookup("notifiers-http-auth-token-path")); err != nil {
		return cfg, err
	}

	mainFS.BoolVar(&cfg.Controller.Notifiers.HTTP.Disruption.Enabled, "notifiers-http-disruption-enabled", false, "Enabler toggle to send disruption notifications using the HTTP notifier (defaulted to false)")

	if err := viper.BindPFlag("controller.notifiers.http.disruption.enabled", mainFS.Lookup("notifiers-http-disruption-enabled")); err != nil {
		return cfg, err
	}

	mainFS.StringVar(&cfg.Controller.Notifiers.HTTP.Disruption.URL, "notifiers-http-disruption-url", "", "URL to send disruption notifications using the HTTP notifier(defaulted to \"\")")

	if err := viper.BindPFlag("controller.notifiers.http.disruption.url", mainFS.Lookup("notifiers-http-disruption-url")); err != nil {
		return cfg, err
	}

	mainFS.BoolVar(&cfg.Controller.Notifiers.HTTP.DisruptionCron.Enabled, "notifiers-http-disruptioncron-enabled", false, "Enabler toggle to send disruption cron notifications using the HTTP notifier (defaulted to false)")

	if err := viper.BindPFlag("controller.notifiers.http.disruptioncron.enabled", mainFS.Lookup("notifiers-http-disruptioncron-enabled")); err != nil {
		return cfg, err
	}

	mainFS.StringVar(&cfg.Controller.Notifiers.HTTP.Disruption.URL, "notifiers-http-disruptioncron-url", "", "URL to send disruption cron notifications using the HTTP notifier(defaulted to \"\")")

	if err := viper.BindPFlag("controller.notifiers.http.disruptioncron.url", mainFS.Lookup("notifiers-http-disruptioncron-url")); err != nil {
		return cfg, err
	}

	mainFS.StringToStringVar(&cfg.Injector.Annotations, "injector-annotations", map[string]string{}, "Annotations added to the generated injector pods")

	if err := viper.BindPFlag("injector.annotations", mainFS.Lookup("injector-annotations")); err != nil {
		return cfg, err
	}

	mainFS.StringToStringVar(&cfg.Injector.Labels, "injector-labels", map[string]string{}, "Labels added to the generated injector pods")

	if err := viper.BindPFlag("injector.labels", mainFS.Lookup("injector-labels")); err != nil {
		return cfg, err
	}

	mainFS.StringVar(&cfg.Injector.ServiceAccount, "injector-service-account", "chaos-injector", "Service account to use for the generated injector pods")

	if err := viper.BindPFlag("injector.serviceAccount.name", mainFS.Lookup("injector-service-account")); err != nil {
		return cfg, err
	}

	mainFS.StringVar(&cfg.Injector.ChaosNamespace, "chaos-namespace", "chaos-engineering", "Namespace of the service account to use for the generated injector pods. Must also host the controller.")

	if err := viper.BindPFlag("injector.chaosNamespace", mainFS.Lookup("chaos-namespace")); err != nil {
		return cfg, err
	}

	mainFS.StringVar(&cfg.Injector.Image, "injector-image", "chaos-injector", "Image to pull for the injector pods")

	if err := viper.BindPFlag("injector.image", mainFS.Lookup("injector-image")); err != nil {
		return cfg, err
	}

	mainFS.StringVar(&cfg.Injector.ImagePullSecrets, "image-pull-secrets", "", "Secrets used for pulling the Docker image from a private registry")

	if err := viper.BindPFlag("controller.imagePullSecrets", mainFS.Lookup("image-pull-secrets")); err != nil {
		return cfg, err
	}

	mainFS.StringVar(&cfg.Injector.LogLevel, "injector-log-level", "DEBUG", "The LOG_LEVEL used for the injector pods")

	if err := viper.BindPFlag("injector.logLevel", mainFS.Lookup("injector-log-level")); err != nil {
		return cfg, err
	}

	mainFS.StringVar(&cfg.Injector.DNSDisruption.DNSServer, "injector-dns-disruption-dns-server", "8.8.8.8", "IP address of the upstream DNS server")

	if err := viper.BindPFlag("injector.dnsDisruption.dnsServer", mainFS.Lookup("injector-dns-disruption-dns-server")); err != nil {
		return cfg, err
	}

	mainFS.StringVar(&cfg.Injector.DNSDisruption.KubeDNS, "injector-dns-disruption-kube-dns", "off", "Whether to use kube-dns for DNS resolution (off, internal, all)")

	if err := viper.BindPFlag("injector.dnsDisruption.kubeDns", mainFS.Lookup("injector-dns-disruption-kube-dns")); err != nil {
		return cfg, err
	}

	mainFS.StringSliceVar(&cfg.Injector.NetworkDisruption.AllowedHosts, "injector-network-disruption-allowed-hosts", []string{}, "List of hosts always allowed by network disruptions (format: <host>;<port>;<protocol>;<flow>)")

	if err := viper.BindPFlag("injector.networkDisruption.allowedHosts", mainFS.Lookup("injector-network-disruption-allowed-hosts")); err != nil {
		return cfg, err
	}

	mainFS.DurationVar(&cfg.Injector.NetworkDisruption.HostResolveInterval, "injector-network-disruption-host-resolve-interval", time.Minute, "How often to re-resolve hostnames specified in a network disruption")

	if err := viper.BindPFlag("injector.networkDisruption.hostResolveInterval", mainFS.Lookup("injector-network-disruption-host-resolve-interval")); err != nil {
		return cfg, err
	}

	mainFS.BoolVar(&cfg.Handler.Enabled, "handler-enabled", false, "Enable the chaos handler for on-init disruptions")

	if err := viper.BindPFlag("handler.enabled", mainFS.Lookup("handler-enabled")); err != nil {
		return cfg, err
	}

	mainFS.StringVar(&cfg.Handler.Image, "handler-image", "chaos-handler", "Image to pull for the handler containers")

	if err := viper.BindPFlag("handler.image", mainFS.Lookup("handler-image")); err != nil {
		return cfg, err
	}

	mainFS.DurationVar(&cfg.Handler.Timeout, "handler-timeout", time.Minute, "Handler init container timeout")

	if err := viper.BindPFlag("handler.timeout", mainFS.Lookup("handler-timeout")); err != nil {
		return cfg, err
	}

	mainFS.DurationVar(&cfg.Handler.MaxTimeout, "handler-max-timeout", time.Hour, "Handler init container maximum timeout")

	if err := viper.BindPFlag("handler.maxTimeout", mainFS.Lookup("handler-max-timeout")); err != nil {
		return cfg, err
	}

	mainFS.StringVar(&cfg.Handler.CPU, "handler-cpu", "100m", "CPU limit/requests for handler init container")

	if err := viper.BindPFlag("handler.cpu", mainFS.Lookup("handler-cpu")); err != nil {
		return cfg, err
	}

	if _, err := resource.ParseQuantity(cfg.Handler.CPU); err != nil {
		return cfg, err
	}

	mainFS.StringVar(&cfg.Handler.Memory, "handler-memory", "100Mi", "Memory limit/requests for handler init container")

	if err := viper.BindPFlag("handler.memory", mainFS.Lookup("handler-memory")); err != nil {
		return cfg, err
	}

	if _, err := resource.ParseQuantity(cfg.Handler.Memory); err != nil {
		return cfg, err
	}

	mainFS.StringVar(&cfg.Controller.Webhook.CertDir, "admission-webhook-cert-dir", "", "Admission webhook certificate directory to search for tls.crt and tls.key files")

	if err := viper.BindPFlag("controller.webhook.certDir", mainFS.Lookup("admission-webhook-cert-dir")); err != nil {
		return cfg, err
	}

	mainFS.StringVar(&cfg.Controller.Webhook.Host, "admission-webhook-host", "", "Host used by the admission controller to serve requests")

	if err := viper.BindPFlag("controller.webhook.host", mainFS.Lookup("admission-webhook-host")); err != nil {
		return cfg, err
	}

	mainFS.IntVar(&cfg.Controller.Webhook.Port, "admission-webhook-port", 9443, "Port used by the admission controller to serve requests")

	if err := viper.BindPFlag("controller.webhook.port", mainFS.Lookup("admission-webhook-port")); err != nil {
		return cfg, err
	}

	mainFS.BoolVar(&cfg.Controller.UserInfoHook, "user-info-webhook", true, "Enable the mutating webhook to inject user info into disruption status")

	if err := viper.BindPFlag("controller.userInfoHook", mainFS.Lookup("user-info-webhook")); err != nil {
		return cfg, err
	}

	mainFS.StringSliceVar(&cfg.Controller.DisabledDisruptions, "disabled-disruptions", []string{}, "List of Disruption Kinds to disable. These should match their kind names from types.go: e.g., `dns-disruption`, `container-failure`, etc. ")

	if err := viper.BindPFlag("controller.disabledDisruptions", mainFS.Lookup("disabled-disruptions")); err != nil {
		return cfg, err
	}

	mainFS.StringVar(&cfg.Controller.SafeMode.Environment, "safemode-environment", "", "Specify the 'location' this controller is run in. All disruptions must have an annotation of chaos.datadoghq.com/environment configured with this location to be allowed to create")

	if err := viper.BindPFlag("controller.safeMode.environment", mainFS.Lookup("safemode-environment")); err != nil {
		return cfg, err
	}

	mainFS.StringSliceVar(&cfg.Controller.SafeMode.PermittedUserGroups, "permitted-user-groups", []string{}, "Set of user groups which, if set, a user must belong to at least one in order to create a disruption")

	if err := viper.BindPFlag("controller.safeMode.permittedUserGroups", mainFS.Lookup("permitted-user-groups")); err != nil {
		return cfg, err
	}

	mainFS.BoolVar(&cfg.Controller.SafeMode.Enable, "safemode-enable", true,
		"Enable or disable the safemode functionality of the chaos-controller")

	if err := viper.BindPFlag("controller.safeMode.enable", mainFS.Lookup("safemode-enable")); err != nil {
		return cfg, err
	}

	mainFS.IntVar(&cfg.Controller.SafeMode.NamespaceThreshold, "safemode-namespace-threshold", 80,
		"Threshold which safemode checks against to see if the number of targets is over safety measures within a namespace.")

	if err := viper.BindPFlag("controller.safeMode.namespaceThreshold", mainFS.Lookup("safemode-namespace-threshold")); err != nil {
		return cfg, err
	}

	mainFS.IntVar(&cfg.Controller.SafeMode.ClusterThreshold, "safemode-cluster-threshold", 66,
		"Threshold which safemode checks against to see if the number of targets is over safety measures within a cluster")

	if err := viper.BindPFlag("controller.safeMode.clusterThreshold", mainFS.Lookup("safemode-cluster-threshold")); err != nil {
		return cfg, err
	}

	mainFS.BoolVar(&cfg.Controller.SafeMode.AllowNodeFailure, "safemode-allow-node-failure", true, "Boolean to determine if validation should prevent disruptions with node failure from being created. Relies on safemode-enable to be true.")

	if err := viper.BindPFlag("controller.safeMode.allowNodeFailure", mainFS.Lookup("safemode-allow-node-failure")); err != nil {
		return cfg, err
	}

	mainFS.BoolVar(&cfg.Controller.SafeMode.AllowNodeLevel, "safemode-allow-node-level", true, "Boolean to determine if validation should prevent disruptions with at the node level from being created. Relies on safemode-enable to be true.")

	if err := viper.BindPFlag("controller.safeMode.allowNodeLevel", mainFS.Lookup("safemode-allow-node-level")); err != nil {
		return cfg, err
	}

	mainFS.BoolVar(&cfg.Controller.CloudProviders.DisableAll, "cloud-providers-disable-all", false, "Disable all cloud providers disruptions (defaults to false, overrides all individual cloud providers configuration)")

	if err := viper.BindPFlag("controller.cloudProviders.disableAll", mainFS.Lookup("cloud-providers-disable-all")); err != nil {
		return cfg, err
	}

	mainFS.DurationVar(&cfg.Controller.CloudProviders.PullInterval, "cloud-providers-pull-interval", 24*time.Hour, "Interval of time to pull the ip ranges of all cloud providers' services (defaults to 1 day)")

	if err := viper.BindPFlag("controller.cloudProviders.pullinterval", mainFS.Lookup("cloud-providers-pull-interval")); err != nil {
		return cfg, err
	}

	mainFS.BoolVar(&cfg.Controller.CloudProviders.AWS.Enabled, "cloud-providers-aws-enabled", true, "Enable AWS cloud provider disruptions (defaults to true, is overridden by --cloud-providers-disable-all)")

	if err := viper.BindPFlag("controller.cloudProviders.aws.enabled", mainFS.Lookup("cloud-providers-aws-enabled")); err != nil {
		return cfg, err
	}

	mainFS.StringVar(&cfg.Controller.CloudProviders.AWS.IPRangesURL, "cloud-providers-aws-iprangesurl", "", "Configure the cloud provider URL to the IP ranges file used by the disruption")

	if err := viper.BindPFlag("controller.cloudProviders.aws.ipRangesURL", mainFS.Lookup("cloud-providers-aws-iprangesurl")); err != nil {
		return cfg, err
	}

	mainFS.BoolVar(&cfg.Controller.CloudProviders.GCP.Enabled, "cloud-providers-gcp-enabled", true, "Enable GCP cloud provider disruptions (defaults to true, is overridden by --cloud-providers-disable-all)")

	if err := viper.BindPFlag("controller.cloudProviders.gcp.enabled", mainFS.Lookup("cloud-providers-gcp-enabled")); err != nil {
		return cfg, err
	}

	mainFS.StringVar(&cfg.Controller.CloudProviders.GCP.IPRangesURL, "cloud-providers-gcp-iprangesurl", "", "Configure the cloud provider URL to the IP ranges file used by the disruption")

	if err := viper.BindPFlag("controller.cloudProviders.gcp.ipRangesURL", mainFS.Lookup("cloud-providers-gcp-iprangesurl")); err != nil {
		return cfg, err
	}

	mainFS.BoolVar(&cfg.Controller.CloudProviders.Datadog.Enabled, "cloud-providers-datadog-enabled", true, "Enable Datadog cloud provider disruptions (defaults to true, is overridden by --cloud-providers-disable-all)")

	if err := viper.BindPFlag("controller.cloudProviders.datadog.enabled", mainFS.Lookup("cloud-providers-datadog-enabled")); err != nil {
		return cfg, err
	}

	mainFS.StringVar(&cfg.Controller.CloudProviders.Datadog.IPRangesURL, "cloud-providers-datadog-iprangesurl", "", "Configure the cloud provider URL to the IP ranges file used by the disruption")

	if err := viper.BindPFlag("controller.cloudProviders.datadog.ipRangesURL", mainFS.Lookup("cloud-providers-datadog-iprangesurl")); err != nil {
		return cfg, err
	}

	mainFS.StringVar(&cfg.Controller.MetricsSink, "metrics-sink", "noop", "metrics sink (datadog, or noop)")

	if err := viper.BindPFlag("controller.metricsSink", mainFS.Lookup("metrics-sink")); err != nil {
		return cfg, err
	}

	mainFS.StringVar(&cfg.Controller.ProfilerSink, "profiler-sink", "noop", "profiler sink (datadog, or noop)")

	if err := viper.BindPFlag("controller.profilerSink", mainFS.Lookup("profiler-sink")); err != nil {
		return cfg, err
	}

	mainFS.StringVar(&cfg.Controller.TracerSink, "tracer-sink", "noop", "tracer sink (datadog, or noop)")

	if err := viper.BindPFlag("controller.tracerSink", mainFS.Lookup("tracer-sink")); err != nil {
		return cfg, err
	}

	mainFS.BoolVar(&cfg.Controller.DisruptionCronEnabled, "disruption-cron-enabled", false, "Enable the DisruptionCron CRD and its controller")

	if err := viper.BindPFlag("controller.disruptionCronEnabled", mainFS.Lookup("disruption-cron-enabled")); err != nil {
		return cfg, err
	}

	mainFS.BoolVar(&cfg.Controller.DisruptionRolloutEnabled, "disruption-rollout-enabled", false, "Enable the DisruptionRollout CRD and its controller")

	if err := viper.BindPFlag("controller.disruptionRolloutEnabled", mainFS.Lookup("disruption-rollout-enabled")); err != nil {
		return cfg, err
	}

	mainFS.DurationVar(&cfg.Controller.DisruptionDeletionTimeout, "disruption-deletion-timeout", DefaultDisruptionDeletionTimeout, "If the deletion time of the disruption is greater than the delete timeout, the disruption is marked as stuck on removal")

	if err := viper.BindPFlag("controller.disruptionDeletionTimeout", mainFS.Lookup("disruption-deletion-timeout")); err != nil {
		return cfg, err
	}

	mainFS.DurationVar(&cfg.Controller.FinalizerDeletionDelay, "finalizer-deletion-delay", DefaultFinalizerDeletionDelay, "Define a delay before we attempt at removing the finalizers (on disruption and disruptioncron only)")

	if err := viper.BindPFlag("controller.finalizerDeletionDelay", mainFS.Lookup("finalizer-deletion-delay")); err != nil {
		return cfg, err
	}

	mainFS.DurationVar(&cfg.Controller.TargetResourceMissingThreshold, "target-resource-missing-threshold", time.Hour*24, "Define the amount of time a cron or rollout will tolerate its target missing before self-deleting")

	if err := viper.BindPFlag("controller.targetResourceMissingThreshold", mainFS.Lookup("target-resource-missing-threshold")); err != nil {
		return cfg, err
	}

	if err := preConfigFS.Parse(osArgs); err != nil {
		return cfg, fmt.Errorf("unable to retrieve configuration parse from provided flag: %w", err)
	}

	// load configuration file if present first and add values to cfg struct
	if configPath != "" {
		logger.Infow("loading configuration file", "config", configPath)

		viper.SetConfigFile(configPath)

		if err := viper.ReadInConfig(); err != nil {
			return cfg, fmt.Errorf("error loading configuration file: %w", err)
		}

		if configMapOverrides != "" {
			var configMap *corev1.ConfigMap

			if backOffErr := backoff.Retry(func() error {
				var err error
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				configMap, err = client.Get(ctx, configMapOverrides, metav1.GetOptions{})
				if err != nil {
					logger.Debugw(fmt.Sprintf("failed to get %s configMap", configMapOverrides), "error", err)
				}
				return err
			},
				backoff.NewConstantBackOff(time.Second*5)); backOffErr != nil {
				return cfg, fmt.Errorf("unable to retry fetching %s: %w", configMapOverrides, backOffErr)
			}

			interfacedMap := make(map[string]interface{}, len(configMap.Data))
			for k, v := range configMap.Data {
				interfacedMap[k] = v
			}

			if err := viper.MergeConfigMap(interfacedMap); err != nil {
				return cfg, fmt.Errorf("unable to merge config map: %w", err)
			}

			go func(resourceVersion string) {
				ticker := time.NewTicker(time.Second * 30)

				for {
					<-ticker.C

					configMap, err := client.Get(context.Background(), configMapOverrides, metav1.GetOptions{})

					if err != nil {
						logger.Errorw(fmt.Sprintf("error getting %s configMap", configMapOverrides), "error", err)
						continue
					}

					if configMap.ResourceVersion != resourceVersion {
						logger.Info("override configmap has changed, restarting")
						os.Exit(0)
					}
				}
			}(configMap.ResourceVersion)
		}

		if err := viper.Unmarshal(&cfg); err != nil {
			return cfg, fmt.Errorf("error unmarshaling configuration: %w", err)
		}

		viper.WatchConfig()
		viper.OnConfigChange(func(in fsnotify.Event) {
			logger.Info("configuration has changed, restarting")
			os.Exit(0)
		})
	}

	// now that configuration file has been loaded, parse all remaining flags
	if err := mainFS.Parse(osArgs); err != nil {
		return cfg, fmt.Errorf("unable to parse main flags: %w", err)
	}

	if !cfg.Controller.UserInfoHook && cfg.Controller.Notifiers.Slack.Enabled {
		return cfg, fmt.Errorf("cannot enable slack notifier without enabling the user info webhook")
	}

	if cfg.Controller.DefaultDuration > 0 && cfg.Controller.MaxDuration > 0 && cfg.Controller.DefaultDuration > cfg.Controller.MaxDuration {
		return cfg, fmt.Errorf("defaultDuration of %s, must be less than or equal to the maxDuration %s", cfg.Controller.DefaultDuration, cfg.Controller.MaxDuration)
	}

	return cfg, nil
}
