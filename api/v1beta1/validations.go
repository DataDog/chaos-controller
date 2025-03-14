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

// newGoValidator instantiates a validator and translator which can be used to inspect a struct marked with `validate`
// tags, and then return an array of validator.ValidationErrors explaining which fields did not match which constraints.
// The returned translator can then be used to transform those errors into easy to understand, user-facing error messages.
// The returned validator and translator are prepared to translate the following tags: required, gte, lte, oneofci. Other tags
// will still be validated, but the error messages will be the defaults.
func newGoValidator() (*validator.Validate, ut.Translator, error) {
	englishLocale := en.New()
	uni := ut.New(englishLocale, englishLocale)

	translator, _ := uni.GetTranslator("en")

	validate := validator.New(validator.WithRequiredStructEnabled())
	validate.SetTagName("chaos_validate")

	// We need to register a translation for every tag we use
	// in order to control the error message returned to the users when
	// their specs are invalid
	if err := registerRequiredTranslation(validate, translator); err != nil {
		return nil, nil, err
	}

	if err := registerGteTranslation(validate, translator); err != nil {
		return nil, nil, err
	}

	if err := registerLteTranslation(validate, translator); err != nil {
		return nil, nil, err
	}

	if err := registerOneofciTranslation(validate, translator); err != nil {
		return nil, nil, err
	}

	return validate, translator, nil
}

func registerGteTranslation(validate *validator.Validate, translator ut.Translator) error {
	return validate.RegisterTranslation("gte", translator, func(ut ut.Translator) error {
		// The {idx} values are interpolated using the arguments to the ut.T("gte", ...) call in the function below
		return ut.Add("gte", "{0} is set to {1}, but must be greater or equal to {2}", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		iStr := fieldErrorToNumString(fe)
		t, _ := ut.T("gte", fe.Namespace(), iStr, fe.Param())

		return t
	})
}

func registerLteTranslation(validate *validator.Validate, translator ut.Translator) error {
	return validate.RegisterTranslation("lte", translator, func(ut ut.Translator) error {
		// The {idx} values are interpolated using the arguments to the ut.T("lte", ...) call in the function below
		return ut.Add("lte", "{0} is set to {1}, but must be less or equal to {2}", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		iStr := fieldErrorToNumString(fe)
		t, _ := ut.T("lte", fe.Namespace(), iStr, fe.Param())

		return t
	})
}

// fieldErrorToNumString can be used by any validation field that constrains numeric types,
// specifically int, *int, or uint, currently. It will take the FieldError and return a string
// that represents the value the user tried to use.
func fieldErrorToNumString(fe validator.FieldError) string {
	// values that are constrained by "lte" or "gte" can include: int, *int, uint
	// fe.Value() is an interface{}, so we need to check each of these type options, one by one
	i, ok := fe.Value().(int)
	if !ok {
		iPtr, k := fe.Value().(*int)
		if !k {
			unsignedVal, k3 := fe.Value().(uint)
			if !k3 {
				// this will be directly seen by the user if their field fails validation.
				return fmt.Sprintf("could not determine value %v for field %s", fe.Value(), fe.Field())
			}

			i = int(unsignedVal)
		} else {
			if iPtr == nil {
				i = 0
			} else {
				i = *iPtr
			}
		}
	}

	return strconv.Itoa(i)
}

func registerRequiredTranslation(validate *validator.Validate, translator ut.Translator) error {
	return validate.RegisterTranslation("required", translator, func(ut ut.Translator) error {
		// The {idx} values are interpolated using the arguments to the ut.T("required", ...) call in the function below
		return ut.Add("required", "{0} is a required field, and must be set", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("required", fe.Namespace())

		return t
	})
}

func registerOneofciTranslation(validate *validator.Validate, translator ut.Translator) error {
	return validate.RegisterTranslation("oneofci", translator, func(ut ut.Translator) error {
		// The {idx} values are interpolated using the arguments to the ut.T("oneofci", ...) call in the function below
		return ut.Add("oneofci", "{0} is set to {1}, but must be one of the following: \"{2}\"", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		options := fe.Param()

		userOptionsString := strings.Join(strings.Split(options, " "), ", ")

		userChoiceStr, ok := fe.Value().(string)
		if !ok {
			return fmt.Sprintf("could not permit value \"%v\" for field %s, try one of \"%s\"", fe.Value(), fe.Field(), userOptionsString)
		}

		t, _ := ut.T("oneofci", fe.Namespace(), userChoiceStr, userOptionsString)

		return t
	})
}

func validateStructTags(s interface{}) error {
	var retErr *multierror.Error

	validate, translator, err := newGoValidator()
	if err != nil {
		return fmt.Errorf("could not validate struct tags: %w", err)
	}

	err = validate.Struct(s)

	if err != nil {
		// this check is only needed when the rare case in which we produce
		// an invalid value for validation such as interface with a nil value
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
