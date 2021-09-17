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
	"fmt"
	"math"
	"regexp"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// filtering container meta data containing a list of filtered out containers (by name) and the reasoning for
// the filter.
type filteredContainer struct {
	removed string
}

// filterContainerIDs filters out any containers that may not be intended for disruption in a multi-container disruption
// this function returns 1 string clice of containers and 1 filteredContainer slice:
// containers: The updated list of containers with the filtered out containers removed
// filteredContainer.removed: A container name that was removed from the list of containers, used later for logging purposes
func filterContainerIDs(pod *corev1.Pod, containers []string, spec v1beta1.DisruptionSpec) ([]string, []filteredContainer) {
	var fContainers []filteredContainer

	if spec.DiskPressure != nil {
		// validate that each container has the volume to be disrupted
		fContainers = getNoValidVolumeCtns(pod, spec, fContainers)
		containers = removeCtnsFromConsideration(pod, fContainers, containers)
	}

	return containers, fContainers
}

// removeCtnsFromConsideration removes containers from the removal list that do not comply with multi-container
// disruption rules. Returns the updated list of containers
func removeCtnsFromConsideration(pod *corev1.Pod, fContainers []filteredContainer, containers []string) []string {
	if len(fContainers) == 0 {
		return containers
	}

	for _, ctn := range pod.Status.ContainerStatuses {
		for _, fcont := range fContainers {
			remove := fcont.removed
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
func getNoValidVolumeCtns(pod *corev1.Pod, spec v1beta1.DisruptionSpec, fContainers []filteredContainer) []filteredContainer {
	for _, ctn := range pod.Spec.Containers {
		if len(ctn.VolumeMounts) == 0 {
			newfcont := filteredContainer{
				removed: ctn.Name,
			}
			fContainers = append(fContainers, newfcont)
		} else {
			found := false
			for _, volume := range ctn.VolumeMounts {
				if volume.MountPath == spec.DiskPressure.Path {
					found = true
					break
				}
			}
			if !found {
				newfcont := filteredContainer{
					removed: ctn.Name,
				}
				fContainers = append(fContainers, newfcont)
			}
		}
	}

	return fContainers
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

// contains returns true when the given string is present in the given slice
func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}
