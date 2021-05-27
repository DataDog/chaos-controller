// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package v1beta1

import (
	"fmt"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/util/intstr"
)

// GetIntOrPercentValueSafely has three return values. The first is the int value of intOrStr, and the second is
// if that int value is a percentage (true) or simply an integer (false).
func GetIntOrPercentValueSafely(intOrStr *intstr.IntOrString) (int, bool, error) {
	switch intOrStr.Type {
	case intstr.Int:
		return intOrStr.IntValue(), false, nil
	case intstr.String:
		s := intOrStr.StrVal

		if strings.HasSuffix(s, "%") {
			s = strings.TrimSuffix(intOrStr.StrVal, "%")

			v, err := strconv.Atoi(s)
			if err != nil {
				return 0, false, fmt.Errorf("invalid value %q: %v", intOrStr.StrVal, err)
			}

			return v, true, nil
		}

		return 0, false, fmt.Errorf("invalid type: string is not a percentage")
	}

	return 0, false, fmt.Errorf("invalid type: neither int nor percentage")
}

func ValidateCount(count *intstr.IntOrString) error {
	value, isPercent, err := GetIntOrPercentValueSafely(count)
	if err != nil {
		return err
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
