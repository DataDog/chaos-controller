// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package v1beta1

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/go-multierror"
)

// MemoryPressureSpec represents a memory pressure disruption
type MemoryPressureSpec struct {
	// Target memory utilization as a percentage (e.g., "76%")
	// +kubebuilder:validation:Required
	TargetPercent string `json:"targetPercent" chaos_validate:"required"`
	// Duration over which memory is gradually consumed (e.g., "10m")
	// If empty, memory is consumed immediately
	RampDuration DisruptionDuration `json:"rampDuration,omitempty"`
}

// Validate validates args for the given disruption
func (s *MemoryPressureSpec) Validate() (retErr error) {
	// Rule: targetPercent must be a valid percentage between 1 and 100
	pct, err := ParseTargetPercent(s.TargetPercent)
	if err != nil {
		retErr = multierror.Append(retErr, fmt.Errorf("invalid targetPercent %q: %w", s.TargetPercent, err))
	} else if pct < 1 || pct > 100 {
		retErr = multierror.Append(retErr, fmt.Errorf("targetPercent must be between 1 and 100, got %d", pct))
	}

	// Rule: rampDuration must be non-negative
	if s.RampDuration.Duration() < 0 {
		retErr = multierror.Append(retErr, fmt.Errorf("rampDuration must be non-negative, got %s", s.RampDuration))
	}

	return retErr
}

// GenerateArgs generates injection or cleanup pod arguments for the given spec
func (s *MemoryPressureSpec) GenerateArgs() []string {
	args := []string{
		"memory-pressure",
		"--target-percent", s.TargetPercent,
	}

	if s.RampDuration.Duration() > 0 {
		args = append(args, "--ramp-duration", s.RampDuration.Duration().String())
	}

	return args
}

func (s *MemoryPressureSpec) Explain() []string {
	pct, _ := ParseTargetPercent(s.TargetPercent)

	explanation := fmt.Sprintf("spec.memoryPressure will cause memory pressure on the target, by joining its cgroup and allocating memory to reach %d%% of the target's memory limit", pct)

	if s.RampDuration.Duration() > 0 {
		explanation += fmt.Sprintf(", ramping up over %s.", s.RampDuration.Duration())
	} else {
		explanation += " immediately."
	}

	return []string{"", explanation}
}

// ParseTargetPercent parses a percentage string like "76%" or "76" and returns the integer value
func ParseTargetPercent(s string) (int, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "%")

	return strconv.Atoi(s)
}
