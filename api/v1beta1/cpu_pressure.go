// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package v1beta1

// CPUPressureSpec represents a cpu pressure disruption
type CPUPressureSpec struct {
}

// Validate validates args for the given disruption
func (s *CPUPressureSpec) Validate() []error {
	return []error{}
}

// GenerateArgs generates injection or cleanup pod arguments for the given spec
func (s *CPUPressureSpec) GenerateArgs() []string {
	args := []string{
		"cpu-pressure",
	}

	return args
}
