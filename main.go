// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

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
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/controllers"
	"github.com/DataDog/chaos-controller/metrics"
	"github.com/DataDog/chaos-controller/metrics/types"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
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
		metricsAddr          string
		enableLeaderElection bool
		podTemplate          string
		podTemplateSpec      corev1.Pod
		sink                 string
	)

	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&podTemplate, "pod-template", "/etc/manager/pod-template.json", "The template file to use to generate injection pods.")
	flag.StringVar(&sink, "metrics-sink", "noop", "Metrics sink (datadog, or noop)")
	flag.Parse()

	// configure logger
	loggerConfig := zap.NewProductionConfig()
	loggerConfig.Level.SetLevel(zapcore.InfoLevel)
	loggerConfig.EncoderConfig.MessageKey = "message"
	loggerConfig.EncoderConfig.EncodeTime = zapcore.EpochMillisTimeEncoder

	// generate logger
	logger, err := loggerConfig.Build()
	if err != nil {
		setupLog.Error(err, "error creating controller logger")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "75ec2fa4.datadoghq.com",
		Port:               9443,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// metrics sink
	ms, err := metrics.GetSink(types.SinkDriver(sink), types.SinkAppController)
	if err != nil {
		ctrl.Log.Error(err, "error while creating metric sink")
	}

	// handle metrics sink client close on exit
	defer func() {
		ctrl.Log.Info("closing metrics sink client before exiting", "sink", ms.GetSinkName())

		if err := ms.Close(); err != nil {
			ctrl.Log.Error(err, "error closing metrics sink client", "sink", ms.GetSinkName())
		}
	}()

	// load pod template
	bytes, err := ioutil.ReadFile(podTemplate) //nolint:gosec
	if err != nil {
		ctrl.Log.Error(err, "unable to read pod template file")
		os.Exit(1)
	}

	err = json.Unmarshal(bytes, &podTemplateSpec)
	if err != nil {
		ctrl.Log.Error(err, "unable to load pod template spec")
		os.Exit(1)
	}

	ctrl.Log.Info("generated pod template", "template", podTemplateSpec)

	// create reconciler
	r := &controllers.DisruptionReconciler{
		Client:          mgr.GetClient(),
		Log:             logger.Sugar(),
		Scheme:          mgr.GetScheme(),
		Recorder:        mgr.GetEventRecorderFor("disruption-controller"),
		MetricsSink:     ms,
		PodTemplateSpec: podTemplateSpec,
		TargetSelector:  controllers.RunningTargetSelector{},
	}

	if err := r.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Disruption")
		os.Exit(1)
	}

	go r.WatchStuckOnRemoval()
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
