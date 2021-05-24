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

func GetIntOrPercentValueSafely(intOrStr *intstr.IntOrString) (int, bool, error) {
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

func validateCount(count *intstr.IntOrString) error {
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
