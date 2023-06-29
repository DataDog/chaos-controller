// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.
package config

import (
	"fmt"
	"os"
	"time"

	cloudtypes "github.com/DataDog/chaos-controller/cloudservice/types"
	"github.com/DataDog/chaos-controller/eventnotifier"
	"go.uber.org/zap"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type config struct {
	Controller controllerConfig `json:"controller"`
	Injector   injectorConfig   `json:"injector"`
	Handler    handlerConfig    `json:"handler"`
}

type controllerConfig struct {
	MetricsBindAddr          string                          `json:"metricsBindAddr"`
	MetricsSink              string                          `json:"metricsSink"`
	ExpiredDisruptionGCDelay time.Duration                   `json:"expiredDisruptionGCDelay"`
	DefaultDuration          time.Duration                   `json:"defaultDuration"`
	DeleteOnly               bool                            `json:"deleteOnly"`
	EnableSafeguards         bool                            `json:"enableSafeguards"`
	EnableObserver           bool                            `json:"enableObserver"`
	LeaderElection           bool                            `json:"leaderElection"`
	Webhook                  controllerWebhookConfig         `json:"webhook"`
	Notifiers                eventnotifier.NotifiersConfig   `json:"notifiersConfig"`
	CloudProviders           cloudtypes.CloudProviderConfigs `json:"cloudProviders"`
	UserInfoHook             bool                            `json:"userInfoHook"`
	SafeMode                 safeModeConfig                  `json:"safeMode"`
	ProfilerSink             string                          `json:"profilerSink"`
}

type controllerWebhookConfig struct {
	CertDir string `json:"certDir"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
}

type safeModeConfig struct {
	Environment        string `json:"environment"`
	Enable             bool   `json:"enable"`
	NamespaceThreshold int    `json:"namespaceThreshold"`
	ClusterThreshold   int    `json:"clusterThreshold"`
}

type injectorConfig struct {
	Image             string                          `json:"image"`
	Annotations       map[string]string               `json:"annotations"`
	Labels            map[string]string               `json:"labels"`
	ChaosNamespace    string                          `json:"namespace"`
	ServiceAccount    string                          `json:"serviceAccount"`
	DNSDisruption     injectorDNSDisruptionConfig     `json:"dnsDisruption"`
	NetworkDisruption injectorNetworkDisruptionConfig `json:"networkDisruption"`
	ImagePullSecrets  string                          `json:"imagePullSecrets"`
}

type injectorDNSDisruptionConfig struct {
	DNSServer string `json:"dnsServer"`
	KubeDNS   string `json:"kubeDns"`
}

type injectorNetworkDisruptionConfig struct {
	AllowedHosts        []string      `json:"allowedHosts"`
	HostResolveInterval time.Duration `json:"hostResolveInterval"`
}

type handlerConfig struct {
	Enabled bool          `json:"enabled"`
	Image   string        `json:"image"`
	Timeout time.Duration `json:"timeout"`
}

func New(logger *zap.SugaredLogger, osArgs []string) (config, error) {
	var (
		configPath string
		cfg        config
	)

	preConfigFS := pflag.NewFlagSet("pre-config", pflag.ContinueOnError)
	mainFS := pflag.NewFlagSet("main-config", pflag.ContinueOnError)

	preConfigFS.ParseErrorsWhitelist.UnknownFlags = true
	preConfigFS.StringVar(&configPath, "config", "", "Configuration file path")
	// we redefine configuration flag into main flag to avoid removing it manually from provided args
	// we also define it to avoid activating "UnknownFlags" for main flags so we'll return an error in case a flag is unknown
	mainFS.StringVar(&configPath, "config", "", "Configuration file path")

	mainFS.StringVar(&cfg.Controller.MetricsBindAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")

	if err := viper.BindPFlag("controller.metricsBindAddr", mainFS.Lookup("metrics-bind-address")); err != nil {
		return cfg, err
	}

	mainFS.BoolVar(&cfg.Controller.LeaderElection, "leader-elect", false, "Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")

	if err := viper.BindPFlag("controller.leaderElection", mainFS.Lookup("leader-elect")); err != nil {
		return cfg, err
	}

	mainFS.BoolVar(&cfg.Controller.DeleteOnly, "delete-only", false,
		"Enable delete only mode which will not allow new disruption to start and will only continue to clean up and remove existing disruptions.")

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

	mainFS.BoolVar(&cfg.Controller.Notifiers.HTTP.Enabled, "notifiers-http-enabled", false, "Enabler toggle for the HTTP notifier (defaulted to false)")

	if err := viper.BindPFlag("controller.notifiers.http.enabled", mainFS.Lookup("notifiers-http-enabled")); err != nil {
		return cfg, err
	}

	mainFS.StringVar(&cfg.Controller.Notifiers.HTTP.URL, "notifiers-http-url", "", "URL to send the notification using the HTTP notifier(defaulted to \"\")")

	if err := viper.BindPFlag("controller.notifiers.http.url", mainFS.Lookup("notifiers-http-url")); err != nil {
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

	mainFS.StringVar(&cfg.Controller.SafeMode.Environment, "safemode-environment", "", "Specify the 'location' this controller is run in. All disruptions must have an annotation of chaos.datadoghq.com/environment configured with this location to be allowed to create")

	if err := viper.BindPFlag("conroller.safemode.environment", mainFS.Lookup("safemode-environment")); err != nil {
		return cfg, err
	}

	mainFS.BoolVar(&cfg.Controller.SafeMode.Enable, "safemode-enable", true,
		"Enable or disable the safemode functionality of the chaos-controller")

	if err := viper.BindPFlag("controller.safemode.enable", mainFS.Lookup("safemode-enable")); err != nil {
		return cfg, err
	}

	mainFS.IntVar(&cfg.Controller.SafeMode.NamespaceThreshold, "safemode-namespace-threshold", 80,
		"Threshold which safemode checks against to see if the number of targets is over safety measures within a namespace.")

	if err := viper.BindPFlag("controller.safemode.namespaceThreshold", mainFS.Lookup("safemode-namespace-threshold")); err != nil {
		return cfg, err
	}

	mainFS.IntVar(&cfg.Controller.SafeMode.ClusterThreshold, "safemode-cluster-threshold", 66,
		"Threshold which safemode checks against to see if the number of targets is over safety measures within a cluster")

	if err := viper.BindPFlag("controller.safemode.clusterThreshold", mainFS.Lookup("safemode-cluster-threshold")); err != nil {
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

	err := preConfigFS.Parse(osArgs)
	if err != nil {
		return cfg, fmt.Errorf("unable to retrieve configuration parse from provided flag: %w", err)
	}

	// load configuration file if present first and add values to cfg struct
	if configPath != "" {
		logger.Infow("loading configuration file", "config", configPath)

		viper.SetConfigFile(configPath)

		if err := viper.ReadInConfig(); err != nil {
			return cfg, fmt.Errorf("error loading configuration file: %w", err)
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

	return cfg, nil
}
