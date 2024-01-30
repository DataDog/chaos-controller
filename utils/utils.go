// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

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

package utils

import (
	"time"

	"github.com/DataDog/chaos-controller/cloudservice"
	"github.com/DataDog/chaos-controller/o11y/metrics"
	"github.com/DataDog/chaos-controller/o11y/tracer"
	"go.uber.org/zap"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Contains returns true when the given string is present in the given slice
func Contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

type SetupWebhookWithManagerConfig struct {
	Manager                       ctrl.Manager
	Logger                        *zap.SugaredLogger
	MetricsSink                   metrics.Sink
	TracerSink                    tracer.Sink
	Recorder                      record.EventRecorder
	NamespaceThresholdFlag        int
	ClusterThresholdFlag          int
	EnableSafemodeFlag            bool
	DeleteOnlyFlag                bool
	HandlerEnabledFlag            bool
	DefaultDurationFlag           time.Duration
	MaxDurationFlag               time.Duration
	ChaosNamespace                string
	CloudServicesProvidersManager cloudservice.CloudServicesProvidersManager
	Environment                   string
}
