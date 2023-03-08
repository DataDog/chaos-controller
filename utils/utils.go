// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

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
	"fmt"
	"time"

	"github.com/DataDog/chaos-controller/cloudservice"
	"github.com/DataDog/chaos-controller/metrics"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
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
	Recorder                      record.EventRecorder
	NamespaceThresholdFlag        int
	ClusterThresholdFlag          int
	EnableSafemodeFlag            bool
	DeleteOnlyFlag                bool
	HandlerEnabledFlag            bool
	DefaultDurationFlag           time.Duration
	ChaosNamespace                string
	CloudServicesProvidersManager *cloudservice.CloudServicesProvidersManager
	Environment                   string
}

// GetTargetedContainersInfo gets the IDs of the targeted containers or all container IDs found in a Pod
func GetTargetedContainersInfo(pod *corev1.Pod, targets []string) (map[string]string, error) {
	if len(pod.Status.ContainerStatuses) < 1 {
		return map[string]string{}, fmt.Errorf("missing container ids for pod '%s'", pod.Name)
	}

	allContainers := map[string]string{}
	targetedContainers := map[string]string{}

	ctns := append(pod.Status.ContainerStatuses, pod.Status.InitContainerStatuses...) //nolint:gocritic

	if len(targets) == 0 {
		// get all running containers ID
		for _, c := range ctns {
			if c.State.Running != nil {
				targetedContainers[c.Name] = c.ContainerID
			}
		}
	} else {
		// populate containers name/ID map
		for _, c := range ctns {
			allContainers[c.Name] = c.ContainerID
		}

		// look for the target in the map
		for _, target := range targets {
			if id, found := allContainers[target]; found {
				targetedContainers[target] = id
			} else {
				return nil, fmt.Errorf("could not find specified container in pod (pod: %s, target: %s)", pod.ObjectMeta.Name, target)
			}
		}
	}

	return targetedContainers, nil
}
