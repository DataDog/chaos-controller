// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package v1beta1

import (
	"github.com/hashicorp/go-multierror"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// CPUPressureSpec represents a cpu pressure disruption
type CPUPressureSpec struct {
	// Count represents the number of cores to target
	// either an integer form or a percentage form appended with a %
	// if empty, it will be considered to be 100%
	Count *intstr.IntOrString `json:"count,omitempty"`
}

// Validate validates args for the given disruption
func (s *CPUPressureSpec) Validate() (retErr error) {
	if s.Count == nil {
		return nil
	}

	// Rule: count must be valid
	if err := ValidateCount(s.Count); err != nil {
		retErr = multierror.Append(retErr, err)
	}

	return retErr
}

// GenerateArgs generates injection or cleanup pod arguments for the given spec
func (s *CPUPressureSpec) GenerateArgs() []string {
	args := []string{
		"cpu-pressure",
	}

	if s.Count != nil {
		args = append(args, "--count", s.Count.String())
	} else {
		// starting from here, we expect downstream consumer to benefit from a valid disruption
		// mostly the injector
		// hence we provide a non empty value
		args = append(args, "--count", "100%")
	}

	return args
}
