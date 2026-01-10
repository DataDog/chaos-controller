// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package v1beta1

import (
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"
)

// ClockSkewSpec represents a clock/time skew disruption
// This disruption manipulates the perceived system time for targeted containers,
// useful for testing time-sensitive behavior like certificate expiration,
// distributed locks, time-based tokens, and more.
type ClockSkewSpec struct {
	// Offset specifies the time shift to apply
	// Positive values advance time into the future (e.g., "+24h" to simulate tomorrow)
	// Negative values move time into the past (e.g., "-1h" to simulate an hour ago)
	// Format follows Go's time.Duration format: "300ms", "1.5h", "2h45m", etc.
	// Examples:
	//   "+24h" - advance time by 24 hours (test cert expiry in 1 day)
	//   "+8760h" - advance time by 1 year (365 days)
	//   "-30m" - go back 30 minutes
	// +kubebuilder:validation:Required
	Offset DisruptionDuration `json:"offset" chaos_validate:"required"`
}

// Validate validates args for the given disruption
func (s *ClockSkewSpec) Validate() (retErr error) {
	if s.Offset == "" {
		retErr = multierror.Append(retErr, fmt.Errorf("clockSkew.offset must be specified"))
		return multierror.Prefix(retErr, "ClockSkew:")
	}

	duration := s.Offset.Duration()
	if duration == 0 {
		retErr = multierror.Append(retErr, fmt.Errorf("clockSkew.offset must not be zero; use a positive value to advance time or a negative value to go back in time"))
	}

	// Sanity check: warn about very large time skews (more than 10 years)
	maxSkew := time.Duration(10 * 365 * 24 * time.Hour) // 10 years
	if duration > maxSkew || duration < -maxSkew {
		retErr = multierror.Append(retErr, fmt.Errorf("clockSkew.offset of %s seems unusually large (>10 years); please verify this is intentional", s.Offset))
	}

	return multierror.Prefix(retErr, "ClockSkew:")
}

// GenerateArgs generates injection pod arguments for the given spec
func (s *ClockSkewSpec) GenerateArgs() []string {
	args := []string{
		"clock-skew",
		"--offset",
		string(s.Offset),
	}

	return args
}

// Explain returns a human-readable explanation of this disruption
func (s *ClockSkewSpec) Explain() []string {
	explanation := []string{""}

	duration := s.Offset.Duration()
	var direction, example string

	if duration > 0 {
		direction = "advance time forward"
		example = "This can be used to test certificate expiration, token expiry, time-based leases, or scheduled tasks that should trigger in the future."
	} else {
		direction = "move time backward"
		example = "This can be used to test handling of clock drift, time synchronization issues, or replaying time-sensitive operations."
	}

	explanation = append(explanation,
		fmt.Sprintf("spec.clockSkew will %s by %s for all processes in the targeted containers.", direction, s.Offset),
		fmt.Sprintf("\tThe system clock will appear to be shifted by %s from the actual time.", s.Offset),
		"\tThis is achieved using LD_PRELOAD and libfaketime to intercept time-related system calls.",
		fmt.Sprintf("\t%s", example),
		"",
		"Important notes:",
		"\t- The actual system clock is not modified; only the perception of time for targeted processes",
		"\t- This affects gettimeofday(), clock_gettime(), time(), and related syscalls",
		"\t- New processes started in the container will inherit the skewed time",
		"\t- Network time synchronization (NTP) in the container will show the real time, not the skewed time",
	)

	if duration > 24*time.Hour {
		days := int(duration.Hours() / 24)
		explanation = append(explanation, fmt.Sprintf("\t- You're skewing time by approximately %d days - great for testing certificate expiration!", days))
	}

	return explanation
}
