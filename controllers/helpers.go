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

package controllers

import (
	"errors"
	"fmt"
	"math"
	"regexp"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	chaostypes "github.com/DataDog/chaos-controller/types"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// filterContainerIDs filters out any containers that may not be intended for disruption in a multi-container disruption
func filterContainerIDs(pod *corev1.Pod, containers []string, spec v1beta1.DisruptionSpec) ([]string, []string, []string) {
	removed := []string{}
	reasoning := []string{}

	if spec.DiskPressure != nil {
		// validate that each container has the volume to be disrupted
		removed, reasoning = getNoValidVolumeCtns(pod, spec)
		containers = removeCtnsFromConsideration(pod, removed, containers)
	}

	return containers, removed, reasoning
}

// removeCtnsFromConsideration removes containers from the removal list that do not comply with multi-container
// disruption rules. Returns the updated list of containers
func removeCtnsFromConsideration (pod *corev1.Pod, removal []string, containers []string ) []string {
	if len(removal) == 0 {
		return containers
	}
	for _, ctn := range pod.Status.ContainerStatuses {
		for _, remove := range removal {
			if ctn.Name == remove {
				for i, id := range containers {
					if id == ctn.ContainerID {
						containers[i] = containers[len(containers)-1]
						containers = containers[:len(containers)-1]
						break
					}
				}
				break
			}
		}
	}

	return containers
}

// getNoValidVolumeCtns returns a list of containers that do not have valid volumes according to the disruption spec
func getNoValidVolumeCtns(pod *corev1.Pod, spec v1beta1.DisruptionSpec) ([]string, []string) {
	removal := []string{}
	reasoning := []string{}
	for _, ctn := range pod.Spec.Containers {
		if len(ctn.VolumeMounts) == 0 {
			removal = append(removal, ctn.Name)
			reasoning = append(reasoning, "Disk Pressure Disruption; Message: Could not find valid volume specified in disruption.")
		} else {
			found := false
			for _, volume := range ctn.VolumeMounts {
				if volume.MountPath == spec.DiskPressure.Path {
					found = true
					break
				}
			}
			if !found {
				removal = append(removal, ctn.Name)
				reasoning = append(reasoning, "Disk Pressure Disruption; Message: Could not find valid volume specified in disruption.")
			}
		}
	}
	return removal, reasoning
}

// getContainerIDs gets the IDs of the targeted containers or all container IDs found in a Pod
func getContainerIDs(pod *corev1.Pod, targets []string) ([]string, error) {
	if len(pod.Status.ContainerStatuses) < 1 {
		return []string{}, fmt.Errorf("missing container ids for pod '%s'", pod.Name)
	}

	containersNameID := map[string]string{}
	containerIDs := []string{}

	ctns := append(pod.Status.ContainerStatuses, pod.Status.InitContainerStatuses...)

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

// assert label selector matches valid grammar, avoids CORE-414
func validateLabelSelector(selector labels.Selector) error {
	labelGrammar := "([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]"
	rgx := regexp.MustCompile(labelGrammar)

	if !rgx.MatchString(selector.String()) {
		return fmt.Errorf("given label selector is invalid, it does not match valid selector grammar: %s %s", selector.String(), labelGrammar)
	}

	return nil
}

// getLabelSelectorFromInstance crafts a label selector made of requirements from the given disruption instance
func getLabelSelectorFromInstance(instance *v1beta1.Disruption) (labels.Selector, error) {
	// we want to ensure we never run into the possibility of using an empty label selector
	if (len(instance.Spec.Selector) == 0 || instance.Spec.Selector == nil) && (len(instance.Spec.AdvancedSelector) == 0 || instance.Spec.AdvancedSelector == nil) {
		return nil, errors.New("selector can't be an empty set")
	}

	selector := labels.NewSelector()

	// add simple selectors by parsing them
	if instance.Spec.Selector != nil {
		req, err := labels.ParseToRequirements(instance.Spec.Selector.AsSelector().String())
		if err != nil {
			return nil, fmt.Errorf("error parsing given selector to requirements: %w", err)
		}

		selector = selector.Add(req...)
	}

	// add advanced selectors
	if instance.Spec.AdvancedSelector != nil {
		for _, req := range instance.Spec.AdvancedSelector {
			var op selection.Operator

			// parse the operator to convert it from one package to another
			switch req.Operator {
			case metav1.LabelSelectorOpIn:
				op = selection.In
			case metav1.LabelSelectorOpNotIn:
				op = selection.NotIn
			case metav1.LabelSelectorOpExists:
				op = selection.Exists
			case metav1.LabelSelectorOpDoesNotExist:
				op = selection.DoesNotExist
			default:
				return nil, fmt.Errorf("error parsing advanced selector operator %s: must be either In, NotIn, Exists or DoesNotExist", req.Operator)
			}

			// generate and add the requirement to the selector
			parsedReq, err := labels.NewRequirement(req.Key, op, req.Values)
			if err != nil {
				return nil, fmt.Errorf("error parsing given advanced selector to requirements: %w", err)
			}

			selector = selector.Add(*parsedReq)
		}
	}

	// if the disruption is supposed to be injected on pod init
	// then let's add a requirement to get pods having the matching label only
	if instance.Spec.OnInit {
		onInitRequirement, err := labels.NewRequirement(chaostypes.DisruptOnInitLabel, selection.Exists, []string{})
		if err != nil {
			return nil, fmt.Errorf("error adding the disrupt-on-init label requirement: %w", err)
		}

		selector.Add(*onInitRequirement)
	}

	return selector, nil
}

// contains returns true when the given string is present in the given slice
func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}
