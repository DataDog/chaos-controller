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

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/controllers"
	"github.com/DataDog/chaos-controller/log"
	"github.com/DataDog/chaos-controller/metrics"
	"github.com/DataDog/chaos-controller/metrics/types"
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

func main() {
	var (
		metricsAddr             string
		enableLeaderElection    bool
		deleteOnly              bool
		sink                    string
		injectorAnnotations     map[string]string
		injectorServiceAccount  string
		injectorImage           string
		admissionWebhookCertDir string
		admissionWebhookHost    string
		admissionWebhookPort    int
	)

	pflag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	pflag.BoolVar(&enableLeaderElection, "enable-leader-election", false, "Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	pflag.BoolVar(&deleteOnly, "delete-only", false,
		"Enable delete only mode which will not allow new disruption to start and will only continue to clean up and remove existing disruptions.")
	pflag.StringToStringVar(&injectorAnnotations, "injector-annotations", map[string]string{}, "Annotations added to the generated injector pods")
	pflag.StringVar(&injectorServiceAccount, "injector-service-account", "chaos-injector", "Service account to use for the generated injector pods")
	pflag.StringVar(&injectorImage, "injector-image", "chaos-injector", "Service account to use for the generated injector pods")
	pflag.StringVar(&sink, "metrics-sink", "noop", "Metrics sink (datadog, or noop)")
	pflag.StringVar(&admissionWebhookCertDir, "admission-webhook-cert-dir", "", "Admission webhook certificate directory to search for tls.crt and tls.key files")
	pflag.StringVar(&admissionWebhookHost, "admission-webhook-host", "", "Host used by the admission controller to serve requests")
	pflag.IntVar(&admissionWebhookPort, "admission-webhook-port", 9443, "Port used by the admission controller to serve requests")
	pflag.Parse()

	logger, err := log.NewZapLogger()
	if err != nil {
		setupLog.Error(err, "error creating controller logger")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "75ec2fa4.datadoghq.com",
		Host:               admissionWebhookHost,
		Port:               admissionWebhookPort,
		CertDir:            admissionWebhookCertDir,
	})
	if err != nil {
		logger.Errorw("unable to start manager", "error", err)
		os.Exit(1)
	}

	// metrics sink
	ms, err := metrics.GetSink(types.SinkDriver(sink), types.SinkAppController)
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

	// create reconciler
	r := &controllers.DisruptionReconciler{
		Client:                 mgr.GetClient(),
		BaseLog:                logger,
		Scheme:                 mgr.GetScheme(),
		Recorder:               mgr.GetEventRecorderFor("disruption-controller"),
		MetricsSink:            ms,
		TargetSelector:         controllers.RunningTargetSelector{},
		DeleteOnly:             deleteOnly,
		InjectorAnnotations:    injectorAnnotations,
		InjectorServiceAccount: injectorServiceAccount,
		InjectorImage:          injectorImage,
	}

	if err := r.SetupWithManager(mgr); err != nil {
		logger.Errorw("unable to create controller", "controller", "Disruption", "error", err)
		os.Exit(1)
	}

	go r.ReportMetrics()

	if err = (&chaosv1beta1.Disruption{}).SetupWebhookWithManager(mgr, logger, ms); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Disruption")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	logger.Infow("restarting chaos-controller")

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		logger.Errorw("problem running manager", "error", err)
		os.Exit(1)
	}
}
