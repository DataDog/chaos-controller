// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package v1beta1

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/hashicorp/go-multierror"
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
			return 0, false, fmt.Errorf("invalid value %q: %w", intOrStr.StrVal, err)
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

func newGoValidator() (*validator.Validate, ut.Translator, error) {
	englishLocale := en.New()
	uni := ut.New(englishLocale, englishLocale)

	translator, _ := uni.GetTranslator("en")

	validate := validator.New(validator.WithRequiredStructEnabled())

	if err := validate.RegisterTranslation("required", translator, func(ut ut.Translator) error {
		return ut.Add("required", "{0} is a required field, and must be set", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("required", fe.Field())

		return t
	}); err != nil {
		return nil, nil, err
	}

	if err := validate.RegisterTranslation("gte", translator, func(ut ut.Translator) error {
		return ut.Add("gte", "{0} is set to {1}, but must be greater or equal to {2}", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		i, ok := fe.Value().(int)
		if !ok {
			iPtr, k := fe.Value().(*int)
			if !k {
				unsignedVal, k3 := fe.Value().(uint)
				if !k3 {
					return fmt.Sprintf("could not determine value of field %s %v", fe.Field(), fe.Value())
				} else {
					i = int(unsignedVal)
				}
			} else {
				if iPtr == nil {
					i = 0
				} else {
					i = *iPtr
				}
			}
		}

		iStr := strconv.Itoa(i)
		t, _ := ut.T("gte", fe.Namespace(), iStr, fe.Param())

		return t
	}); err != nil {
		return nil, nil, err
	}

	if err := validate.RegisterTranslation("lte", translator, func(ut ut.Translator) error {
		return ut.Add("lte", "{0} is set to {1}, but must be less or equal to {2}", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		i, ok := fe.Value().(int)
		if !ok {
			iPtr, k := fe.Value().(*int)
			if !k {
				unsignedVal, k3 := fe.Value().(uint)
				if !k3 {
					return fmt.Sprintf("could not determine value of field %s %v", fe.Field(), fe.Value())
				} else {
					i = int(unsignedVal)
				}
			} else {
				if iPtr == nil {
					i = 0
				} else {
					i = *iPtr
				}
			}
		}

		iStr := strconv.Itoa(i)
		t, _ := ut.T("lte", fe.Namespace(), iStr, fe.Param())

		return t
	}); err != nil {
		return nil, nil, err
	}

	return validate, translator, nil
}

func ValidateStructTags(s DisruptionSpec) error {
	var retErr *multierror.Error

	validate, translator, err := newGoValidator()
	if err != nil {
		return fmt.Errorf("could not validate struct tags: %w", err)
	}

	err = validate.Struct(s)

	if err != nil {
		// this check is only needed when your code could produce
		// an invalid value for validation such as interface with nil
		// value most including myself do not usually have code like this.
		var invalidValidationError *validator.InvalidValidationError
		if errors.As(err, &invalidValidationError) {
			return err
		}

		for _, err := range err.(validator.ValidationErrors) {
			retErr = multierror.Append(retErr,
				multierror.Prefix(errors.New(err.Translate(translator)), "validate:"),
			)
		}
	}

	if retErr != nil {
		return retErr.ErrorOrNil()
	}

	return nil
}

// IsUpdateConflictError tells us if this error is of the forms:
// "Operation cannot be fulfilled on disruptions.chaos.datadoghq.com "chaos-network-drop": the object has been modified; please apply your changes to the latest version and try again"
// "Operation cannot be fulfilled on disruptions.chaos.datadoghq.com "name": StorageError: invalid object, Code: 4, Key: /registry/chaos.datadoghq.com/disruptions/namespace/name, ResourceVersion: 0, AdditionalErrorMsg: Precondition failed: UID in precondition: 3534199c-2597-443e-ae59-92e003310d64, UID in object meta:"
// Sadly this doesn't seem to be one of the errors checkable with a function from "k8s.io/apimachinery/pkg/api/errors"
// So we parse the error message directly
func IsUpdateConflictError(err error) bool {
	return strings.Contains(err.Error(), "please apply your changes to the latest version and try again") || strings.Contains(err.Error(), "Precondition failed: UID in precondition")
}
