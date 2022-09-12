// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

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

package controllers

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/cloudservice"
	cloudtypes "github.com/DataDog/chaos-controller/cloudservice/types"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// getContainerIDs gets the IDs of the targeted containers or all container IDs found in a Pod
func getContainerIDs(pod *corev1.Pod, targets []string) ([]string, error) {
	if len(pod.Status.ContainerStatuses) < 1 {
		return []string{}, fmt.Errorf("missing container ids for pod '%s'", pod.Name)
	}

	containersNameID := map[string]string{}
	containerIDs := []string{}

	ctns := append(pod.Status.ContainerStatuses, pod.Status.InitContainerStatuses...) //nolint:gocritic

	if len(targets) == 0 {
		// get all running containers ID
		for _, c := range ctns {
			if c.State.Running != nil {
				containerIDs = append(containerIDs, c.ContainerID)
			}
		}
	} else {
		// populate containers name/ID map
		for _, c := range ctns {
			containersNameID[c.Name] = c.ContainerID
		}

		// look for the target in the map
		for _, target := range targets {
			if id, found := containersNameID[target]; found {
				containerIDs = append(containerIDs, id)
			} else {
				return nil, fmt.Errorf("could not find specified container in pod (pod: %s, target: %s)", pod.ObjectMeta.Name, target)
			}
		}
	}

	return containerIDs, nil
}

// This function returns a scaled value from an IntOrString type. If the IntOrString
// is a percentage string value it's treated as a percentage and scaled appropriately
// in accordance to the total, if it's an int value it's treated as a a simple value and
// if it is a string value which is either non-numeric or numeric but lacking a trailing '%' it returns an error.
func getScaledValueFromIntOrPercent(intOrPercent *intstr.IntOrString, total int, roundUp bool) (int, error) {
	if intOrPercent == nil {
		return 0, k8serrors.NewBadRequest("nil value for IntOrString")
	}

	value, isPercent, err := v1beta1.GetIntOrPercentValueSafely(intOrPercent)

	if err != nil {
		return 0, fmt.Errorf("invalid value for IntOrString: %v", err)
	}

	if isPercent {
		if roundUp {
			value = int(math.Ceil(float64(value) * (float64(total)) / 100))
		} else {
			value = int(math.Floor(float64(value) * (float64(total)) / 100))
		}
	}

	return value, nil
}

func calculateRemainingDuration(instance v1beta1.Disruption) time.Duration {
	return calculateDeadline(
		instance.Spec.Duration.Duration(),
		instance.ObjectMeta.CreationTimestamp.Time,
	)
}

// returned value can be negative if deadline is in the past
func calculateDeadline(duration time.Duration, creationTime time.Time) time.Duration {
	// first we must calculate the timout from when the disruption was created, not from now
	timeout := creationTime.Add(duration)
	now := time.Now() // rather not take the risk that the time changes by a second during this function

	// return the number of seconds between now and the deadline
	return timeout.Sub(now)
}

// assert label selector matches valid grammar, avoids CORE-414
func validateLabelSelector(selector labels.Selector) error {
	labelGrammar := "([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]"
	rgx := regexp.MustCompile(labelGrammar)

	if !rgx.MatchString(selector.String()) {
		return fmt.Errorf("given label selector is invalid, it does not match valid selector grammar: %s %s", selector.String(), labelGrammar)
	}

	return nil
}

// transformCloudSpecToHostsSpec from a cloud spec disruption, get all ip ranges of services provided and transform them into a list of hosts spec
func transformCloudSpecToHostsSpec(log *zap.SugaredLogger, cloudManager *cloudservice.CloudServicesProvidersManager, cloudSpec *v1beta1.NetworkDisruptionCloudSpec) ([]v1beta1.NetworkDisruptionHostSpec, error) {
	hosts := []v1beta1.NetworkDisruptionHostSpec{}
	clouds := map[cloudtypes.CloudProviderName]*[]v1beta1.NetworkDisruptionCloudServiceSpec{
		cloudtypes.CloudProviderAWS: cloudSpec.AWSServiceList,
	}

	for cloudName, serviceList := range clouds {
		serviceListNames := []string{}
		for _, service := range *serviceList {
			serviceListNames = append(serviceListNames, service.ServiceName)
		}

		ipRangesPerService, err := cloudManager.GetServicesIPRanges(cloudName, serviceListNames)
		if err != nil {
			return nil, err
		}

		for _, serviceSpec := range *serviceList {
			for _, ipRange := range ipRangesPerService[serviceSpec.ServiceName] {
				hosts = append(hosts, v1beta1.NetworkDisruptionHostSpec{
					Host:     ipRange,
					Protocol: serviceSpec.Protocol,
					Flow:     serviceSpec.Flow,
				})
			}
		}
	}

	return hosts, nil
}

// isModifiedError tells us if this error is of the form:
// "Operation cannot be fulfilled on disruptions.chaos.datadoghq.com "chaos-network-drop": the object has been modified; please apply your changes to the latest version and try again"
// Sadly this doesn't seem to be one of the errors checkable with a function from "k8s.io/apimachinery/pkg/api/errors"
// So we parse the error message directly
func isModifiedError(err error) bool {
	return strings.Contains(err.Error(), "please apply your changes to the latest version and try again")
}
