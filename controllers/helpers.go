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
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// getContainerIDs gets the IDs of the targeted containers or all container IDs found in a Pod
func getContainerIDs(pod *corev1.Pod, targets []string) ([]string, error) {
	if len(pod.Status.ContainerStatuses) < 1 {
		return []string{}, fmt.Errorf("missing container ids for pod '%s'", pod.Name)
	}

	ctns := []string{}
	// if no containers are specified or found in the list of targeted containers, all containers are targeted using the fallback
	fallback := []string{}

	for _, container := range pod.Status.ContainerStatuses {
		for _, target := range targets {
			if container.Name == target {
				ctns = append(ctns, container.ContainerID)
			}
		}

		fallback = append(fallback, container.ContainerID)
	}

	if len(ctns) == 0 {
		// targets were expected, most likely a typo in container name, erroring
		if len(targets) != 0 {
			return ctns, fmt.Errorf("could not find specified targets in pod spec, most likly a typo")
		}

		return fallback, nil
	}

	return ctns, nil
}

// This function returns a scaled value from an IntOrString type. If the IntOrString
// is a percentage string value it's treated as a percentage and scaled appropriately
// in accordance to the total, if it's an int value it's treated as a a simple value and
// if it is a string value which is either non-numeric or numeric but lacking a trailing '%' it returns an error.
func getScaledValueFromIntOrPercent(intOrPercent *intstr.IntOrString, total int, roundUp bool) (int, error) {
	if intOrPercent == nil {
		return 0, errors.NewBadRequest("nil value for IntOrString")
	}

	value, isPercent, err := getIntOrPercentValueSafely(intOrPercent)

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

func getIntOrPercentValueSafely(intOrStr *intstr.IntOrString) (int, bool, error) {
	switch intOrStr.Type {
	case intstr.Int:
		return intOrStr.IntValue(), false, nil
	case intstr.String:
		isPercent := false
		s := intOrStr.StrVal

		if strings.HasSuffix(s, "%") {
			isPercent = true
			s = strings.TrimSuffix(intOrStr.StrVal, "%")
		} else {
			return 0, false, fmt.Errorf("invalid type: string is not a percentage")
		}

		v, err := strconv.Atoi(s)

		if err != nil {
			return 0, false, fmt.Errorf("invalid value %q: %v", intOrStr.StrVal, err)
		}

		return v, isPercent, nil
	}

	return 0, false, fmt.Errorf("invalid type: neither int nor percentage")
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
