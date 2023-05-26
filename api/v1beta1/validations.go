// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package v1beta1

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func validateServices(k8sClient client.Client, services []NetworkDisruptionServiceSpec) error {
	// ensure given services exist and are compatible
	for _, service := range services {
		k8sService := corev1.Service{}
		serviceKey := types.NamespacedName{
			Namespace: service.Namespace,
			Name:      service.Name,
		}

		// try to get the service and throw an error if it does not exist
		if err := k8sClient.Get(context.Background(), serviceKey, &k8sService); err != nil {
			if client.IgnoreNotFound(err) == nil {
				if service.Namespace == "" || service.Name == "" {
					return fmt.Errorf("either service namespace or name have not been properly set for this service: %s/%s -> namespace/name", service.Namespace, service.Name)
				}
				
				return fmt.Errorf("the service specified in the network disruption (%s/%s) does not exist", service.Namespace, service.Name)
			}

			return fmt.Errorf("error retrieving the specified network disruption service: %w", err)
		}

		_, notFoundPorts := service.ExtractAffectedPortsInServicePorts(&k8sService)
		if len(notFoundPorts) > 0 {
			errorOnNotFoundPorts := []string{}

			for _, port := range notFoundPorts {
				displayedStringsForPort := []string{}

				if port.Name != "" {
					displayedStringsForPort = append(displayedStringsForPort, port.Name)
				}

				if port.Port != 0 {
					displayedStringsForPort = append(displayedStringsForPort, strconv.Itoa(port.Port))
				}

				errorOnNotFoundPorts = append(errorOnNotFoundPorts, strings.Join(displayedStringsForPort, "/"))
			}

			return fmt.Errorf("the ports (%s) specified for the service in the network disruption (%s/%s) do not exist", errorOnNotFoundPorts, service.Name, service.Namespace)
		}

		// check the service type
		if k8sService.Spec.Type != corev1.ServiceTypeClusterIP {
			return fmt.Errorf("the service specified in the network disruption (%s/%s) is of type %s, but only the following service types are supported: ClusterIP", service.Namespace, service.Name, k8sService.Spec.Type)
		}
	}

	return nil
}

// GetIntOrPercentValueSafely has three return values. The first is the int value of intOrStr, and the second is
// if that int value is a percentage (true) or simply an integer (false).
func GetIntOrPercentValueSafely(intOrStr *intstr.IntOrString) (int, bool, error) {
	if intOrStr == nil {
		return 0, false, fmt.Errorf("invalid type: pointer is nil")
	}

	switch intOrStr.Type {
	case intstr.Int:
		return intOrStr.IntValue(), false, nil
	case intstr.String:
		s := intOrStr.StrVal
		isPercent := false

		if strings.HasSuffix(s, "%") {
			s = strings.TrimSuffix(intOrStr.StrVal, "%")
			isPercent = true
		}

		v, err := strconv.Atoi(s)
		if err != nil {
			return 0, false, fmt.Errorf("invalid value %q: %v", intOrStr.StrVal, err)
		}

		return v, isPercent, nil
	}

	return 0, false, fmt.Errorf("invalid type: neither int nor percentage")
}

func ValidateCount(count *intstr.IntOrString) error {
	value, isPercent, err := GetIntOrPercentValueSafely(count)
	if err != nil {
		return fmt.Errorf("error determining value of spec.count: %w", err)
	}

	if isPercent {
		if value <= 0 || value > 100 {
			return fmt.Errorf("count must be a positive integer or a valid percentage value")
		}
	} else {
		if value <= 0 {
			return fmt.Errorf("count must be a positive integer or a valid percentage value")
		}
	}

	return nil
}
