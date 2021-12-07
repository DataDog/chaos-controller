// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

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

package main

import (
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/controllers"
	"github.com/DataDog/chaos-controller/eventbroadcaster"
	"github.com/DataDog/chaos-controller/eventnotifier"
	"github.com/DataDog/chaos-controller/log"
	"github.com/DataDog/chaos-controller/metrics"
	"github.com/DataDog/chaos-controller/metrics/types"
	"github.com/DataDog/chaos-controller/targetselector"
	chaoswebhook "github.com/DataDog/chaos-controller/webhook"
	"github.com/spf13/viper"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	// +kubebuilder:scaffold:scheme
	_ = clientgoscheme.AddToScheme(scheme)
	_ = chaosv1beta1.AddToScheme(scheme)
}

type config struct {
	Controller controllerConfig `json:"controller"`
	Injector   injectorConfig   `json:"injector"`
	Handler    handlerConfig    `json:"handler"`
}

type controllerConfig struct {
	MetricsBindAddr          string                        `json:"metricsBindAddr"`
	MetricsSink              string                        `json:"metricsSink"`
	ImagePullSecrets         string                        `json:"imagePullSecrets"`
	ExpiredDisruptionGCDelay time.Duration                 `json:"expiredDisruptionGCDelay"`
	DefaultDuration          time.Duration                 `json:"defaultDuration"`
	DeleteOnly               bool                          `json:"deleteOnly"`
	EnableSafeguards         bool                          `json:"enableSafeguards"`
	LeaderElection           bool                          `json:"leaderElection"`
	Webhook                  controllerWebhookConfig       `json:"webhook"`
	Notifiers                eventnotifier.NotifiersConfig `json:"notifiersConfig"`
}

type controllerWebhookConfig struct {
	CertDir string `json:"certDir"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
}

type injectorConfig struct {
	Image             string                          `json:"image"`
	Annotations       map[string]string               `json:"annotations"`
	ServiceAccount    injectorServiceAccountConfig    `json:"serviceAccount"`
	DNSDisruption     injectorDNSDisruptionConfig     `json:"dnsDisruption"`
	NetworkDisruption injectorNetworkDisruptionConfig `json:"networkDisruption"`
}

type injectorServiceAccountConfig struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type injectorDNSDisruptionConfig struct {
	DNSServer string `json:"dnsServer"`
	KubeDNS   string `json:"kubeDns"`
}

type injectorNetworkDisruptionConfig struct {
	AllowedHosts []string `json:"allowedHosts"`
}

type handlerConfig struct {
	Enabled bool          `json:"enabled"`
	Image   string        `json:"image"`
	Timeout time.Duration `json:"timeout"`
}

func main() {
	var (
		configPath string
		cfg        config
	)

	// parse flags
	pflag.StringVar(&configPath, "config", "", "Configuration file path")

	pflag.StringVar(&cfg.Controller.MetricsBindAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	handleFatalError(viper.BindPFlag("controller.metrics.addr", pflag.Lookup("metrics-bind-address")))

	pflag.BoolVar(&cfg.Controller.LeaderElection, "leader-elect", false, "Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	handleFatalError(viper.BindPFlag("controller.leaderElection", pflag.Lookup("leader-elect")))

	pflag.BoolVar(&cfg.Controller.DeleteOnly, "delete-only", false,
		"Enable delete only mode which will not allow new disruption to start and will only continue to clean up and remove existing disruptions.")
	handleFatalError(viper.BindPFlag("controller.deleteOnly", pflag.Lookup("delete-only")))

	pflag.BoolVar(&cfg.Controller.EnableSafeguards, "enable-safeguards", true, "Enable safeguards on target selection")
	handleFatalError(viper.BindPFlag("controller.enableSafeguards", pflag.Lookup("enable-safeguards")))

	pflag.StringVar(&cfg.Controller.ImagePullSecrets, "image-pull-secrets", "", "Secrets used for pulling the Docker image from a private registry")
	handleFatalError(viper.BindPFlag("controller.imagePullSecrets", pflag.Lookup("image-pull-secrets")))

	pflag.DurationVar(&cfg.Controller.ExpiredDisruptionGCDelay, "expired-disruption-gc-delay", time.Minute*15, "Seconds after a disruption expires before being automatically deleted")
	handleFatalError(viper.BindPFlag("controller.expiredDisruptionGCDelay", pflag.Lookup("expired-disruption-gc-delay")))

	pflag.DurationVar(&cfg.Controller.DefaultDuration, "default-duration", time.Hour, "Default duration for a disruption with none specified")
	handleFatalError(viper.BindPFlag("controller.defaultDuration", pflag.Lookup("default-duration")))

	pflag.StringVar(&cfg.Controller.MetricsSink, "metrics-sink", "noop", "Metrics sink (datadog, or noop)")
	handleFatalError(viper.BindPFlag("controller.metricsSink", pflag.Lookup("metrics-sink")))

	pflag.StringVar(&cfg.Controller.Notifiers.Common.ClusterName, "notifiers-common-clustername", "", "Cluster Name for notifiers output")
	handleFatalError(viper.BindPFlag("controller.notifiers.common.clusterName", pflag.Lookup("notifiers-common-clustername")))

	pflag.BoolVar(&cfg.Controller.Notifiers.Noop.Enabled, "notifiers-noop-enabled", false, "Enabler toggle for the NOOP notifier (defaulted to false)")
	handleFatalError(viper.BindPFlag("controller.notifiers.noop.enabled", pflag.Lookup("notifiers-noop-enabled")))

	pflag.BoolVar(&cfg.Controller.Notifiers.Slack.Enabled, "notifiers-slack-enabled", false, "Enabler toggle for the Slack notifier (defaulted to false)")
	handleFatalError(viper.BindPFlag("controller.notifiers.slack.enabled", pflag.Lookup("notifiers-slack-enabled")))

	pflag.StringVar(&cfg.Controller.Notifiers.Slack.TokenFilepath, "notifiers-slack-tokenfilepath", "", "File path of the API token for the Slack notifier (defaulted to empty string)")
	handleFatalError(viper.BindPFlag("controller.notifiers.slack.tokenFilepath", pflag.Lookup("notifiers-slack-tokenfilepath")))

	pflag.StringToStringVar(&cfg.Injector.Annotations, "injector-annotations", map[string]string{}, "Annotations added to the generated injector pods")
	handleFatalError(viper.BindPFlag("injector.annotations", pflag.Lookup("injector-annotations")))

	pflag.StringVar(&cfg.Injector.ServiceAccount.Name, "injector-service-account", "chaos-injector", "Service account to use for the generated injector pods")
	handleFatalError(viper.BindPFlag("injector.serviceAccount.name", pflag.Lookup("injector-service-account")))

	pflag.StringVar(&cfg.Injector.ServiceAccount.Namespace, "injector-service-account-namespace", "chaos-engineering", "Namespace of the service account to use for the generated injector pods. Should also host the controller.")
	handleFatalError(viper.BindPFlag("injector.serviceAccount.namespace", pflag.Lookup("injector-service-account-namespace")))

	pflag.StringVar(&cfg.Injector.Image, "injector-image", "chaos-injector", "Image to pull for the injector pods")
	handleFatalError(viper.BindPFlag("injector.image", pflag.Lookup("injector-image")))

	pflag.StringVar(&cfg.Injector.DNSDisruption.DNSServer, "injector-dns-disruption-dns-server", "8.8.8.8", "IP address of the upstream DNS server")
	handleFatalError(viper.BindPFlag("injector.dnsDisruption.dnsServer", pflag.Lookup("injector-dns-disruption-dns-server")))

	pflag.StringVar(&cfg.Injector.DNSDisruption.KubeDNS, "injector-dns-disruption-kube-dns", "off", "Whether to use kube-dns for DNS resolution (off, internal, all)")
	handleFatalError(viper.BindPFlag("injector.dnsDisruption.kubeDns", pflag.Lookup("injector-dns-disruption-kube-dns")))

	pflag.StringSliceVar(&cfg.Injector.NetworkDisruption.AllowedHosts, "injector-network-disruption-allowed-hosts", []string{}, "List of hosts always allowed by network disruptions (format: <host>;<port>;<protocol>)")
	handleFatalError(viper.BindPFlag("injector.networkDisruption.allowedHosts", pflag.Lookup("injector-network-disruption-allowed-hosts")))

	pflag.BoolVar(&cfg.Handler.Enabled, "handler-enabled", false, "Enable the chaos handler for on-init disruptions")
	handleFatalError(viper.BindPFlag("handler.enabled", pflag.Lookup("handler-enabled")))

	pflag.StringVar(&cfg.Handler.Image, "handler-image", "chaos-handler", "Image to pull for the handler containers")
	handleFatalError(viper.BindPFlag("handler.image", pflag.Lookup("handler-image")))

	pflag.DurationVar(&cfg.Handler.Timeout, "handler-timeout", time.Minute, "Handler init container timeout")
	handleFatalError(viper.BindPFlag("handler.timeout", pflag.Lookup("handler-timeout")))

	pflag.StringVar(&cfg.Controller.Webhook.CertDir, "admission-webhook-cert-dir", "", "Admission webhook certificate directory to search for tls.crt and tls.key files")
	handleFatalError(viper.BindPFlag("controller.webhook.certDir", pflag.Lookup("admission-webhook-cert-dir")))

	pflag.StringVar(&cfg.Controller.Webhook.Host, "admission-webhook-host", "", "Host used by the admission controller to serve requests")
	handleFatalError(viper.BindPFlag("controller.webhook.host", pflag.Lookup("admission-webhook-host")))

	pflag.IntVar(&cfg.Controller.Webhook.Port, "admission-webhook-port", 9443, "Port used by the admission controller to serve requests")
	handleFatalError(viper.BindPFlag("controller.webhook.port", pflag.Lookup("admission-webhook-port")))

	pflag.Parse()

	logger, err := log.NewZapLogger()
	if err != nil {
		setupLog.Error(err, "error creating controller logger")
		os.Exit(1)
	}

	// get controller node name
	controllerNodeName, exists := os.LookupEnv("CONTROLLER_NODE_NAME")
	if !exists {
		logger.Fatal("missing required CONTROLLER_NODE_NAME environment variable")
	}

	// load configuration file if present
	if configPath != "" {
		logger.Infow("loading configuration file", "config", configPath)

		viper.SetConfigFile(configPath)

		if err := viper.ReadInConfig(); err != nil {
			logger.Fatalw("error loading configuration file", "error", err)
		}

		if err := viper.Unmarshal(&cfg); err != nil {
			logger.Fatalw("error unmarshaling configuration", "error", err)
		}

		viper.WatchConfig()
		viper.OnConfigChange(func(in fsnotify.Event) {
			logger.Info("configuration has changed, restarting")
			os.Exit(0)
		})
	}

	broadcaster := eventbroadcaster.EventBroadcaster()
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: cfg.Controller.MetricsBindAddr,
		LeaderElection:     cfg.Controller.LeaderElection,
		LeaderElectionID:   "75ec2fa4.datadoghq.com",
		EventBroadcaster:   broadcaster,
		Host:               cfg.Controller.Webhook.Host,
		Port:               cfg.Controller.Webhook.Port,
		CertDir:            cfg.Controller.Webhook.CertDir,
	})

	if err != nil {
		logger.Errorw("unable to start manager", "error", err)
		os.Exit(1)
	}

	// event notifiers
	err = eventbroadcaster.RegisterNotifierSinks(mgr, broadcaster, cfg.Controller.Notifiers, logger)
	if err != nil {
		logger.Errorw("error(s) while creating notifiers", "error", err)
	}

	// metrics sink
	ms, err := metrics.GetSink(types.SinkDriver(cfg.Controller.MetricsSink), types.SinkAppController)
	if err != nil {
		logger.Errorw("error while creating metric sink", "error", err)
	}

	if ms.MetricRestart() != nil {
		logger.Errorw("error sending MetricRestart", "sink", ms.GetSinkName())
	}

	// handle metrics sink client close on exit
	defer func() {
		logger.Infow("closing metrics sink client before exiting", "sink", ms.GetSinkName())

		if err := ms.Close(); err != nil {
			logger.Errorw("error closing metrics sink client", "sink", ms.GetSinkName(), "error", err)
		}
	}()

	// target selector
	targetSelector := targetselector.NewRunningTargetSelector(cfg.Controller.EnableSafeguards, controllerNodeName)

	// create reconciler
	r := &controllers.DisruptionReconciler{
		Client:                                mgr.GetClient(),
		BaseLog:                               logger,
		Scheme:                                mgr.GetScheme(),
		Recorder:                              mgr.GetEventRecorderFor("disruption-controller"),
		MetricsSink:                           ms,
		TargetSelector:                        targetSelector,
		InjectorAnnotations:                   cfg.Injector.Annotations,
		InjectorServiceAccount:                cfg.Injector.ServiceAccount.Name,
		InjectorImage:                         cfg.Injector.Image,
		InjectorServiceAccountNamespace:       cfg.Injector.ServiceAccount.Namespace,
		InjectorDNSDisruptionDNSServer:        cfg.Injector.DNSDisruption.DNSServer,
		InjectorDNSDisruptionKubeDNS:          cfg.Injector.DNSDisruption.KubeDNS,
		InjectorNetworkDisruptionAllowedHosts: cfg.Injector.NetworkDisruption.AllowedHosts,
		ImagePullSecrets:                      cfg.Controller.ImagePullSecrets,
		ExpiredDisruptionGCDelay:              cfg.Controller.ExpiredDisruptionGCDelay,
	}

	informerClient := kubernetes.NewForConfigOrDie(ctrl.GetConfigOrDie())
	kubeInformerFactory := kubeinformers.NewSharedInformerFactoryWithOptions(informerClient, time.Minute*5, kubeinformers.WithNamespace(cfg.Injector.ServiceAccount.Namespace))

	if err := r.SetupWithManager(mgr, kubeInformerFactory); err != nil {
		logger.Errorw("unable to create controller", "controller", "Disruption", "error", err)
		os.Exit(1) //nolint:gocritic
	}

	stopCh := make(chan struct{})
	kubeInformerFactory.Start(stopCh)

	go r.ReportMetrics()

	// register disruption validating webhook
	if err = (&chaosv1beta1.Disruption{}).SetupWebhookWithManager(mgr, logger, ms, cfg.Controller.DeleteOnly, cfg.Handler.Enabled, cfg.Controller.DefaultDuration); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Disruption")
		os.Exit(1) //nolint:gocritic
	}

	// register chaos handler init container mutating webhook
	mgr.GetWebhookServer().Register("/mutate-v1-pod-chaos-handler-init-container", &webhook.Admission{
		Handler: &chaoswebhook.ChaosHandlerMutator{
			Client:  mgr.GetClient(),
			Log:     logger,
			Image:   cfg.Handler.Image,
			Timeout: cfg.Handler.Timeout,
		},
	})

	// register user info mutating webhook
	mgr.GetWebhookServer().Register("/mutate-chaos-datadoghq-com-v1beta1-disruption-user-info", &webhook.Admission{
		Handler: &chaoswebhook.UserInfoMutator{
			Client: mgr.GetClient(),
			Log:    logger,
		},
	})

	// +kubebuilder:scaffold:builder

	logger.Infow("restarting chaos-controller")

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		stopCh <- struct{}{} // stop the informer

		logger.Errorw("problem running manager", "error", err)
		os.Exit(1) //nolint:gocritic
	}
}

// handleFatalError logs the given error and exits if err is not nil
func handleFatalError(err error) {
	if err != nil {
		setupLog.Error(err, "fatal error occurred on setup")
		os.Exit(1)
	}
}
