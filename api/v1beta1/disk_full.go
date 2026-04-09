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
	"k8s.io/apimachinery/pkg/api/resource"
)

// DiskFullSpec represents a disk full (ENOSPC) disruption that fills a target volume
type DiskFullSpec struct {
	// Path is the mount path inside the target pod to fill (e.g., "/data", "/var/log")
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Path string `json:"path" chaos_validate:"required"`
	// Capacity is the target fill percentage of total volume capacity (e.g., "95%").
	// Mutually exclusive with Remaining.
	// +kubebuilder:validation:Pattern=`^\d{1,3}%$`
	Capacity string `json:"capacity,omitempty"`
	// Remaining is the amount of free space to leave on the volume (e.g., "50Mi", "1Gi").
	// Mutually exclusive with Capacity.
	Remaining string `json:"remaining,omitempty"`
}

// Validate validates args for the given disruption
func (s *DiskFullSpec) Validate() (retErr error) {
	if strings.TrimSpace(s.Path) == "" {
		retErr = multierror.Append(retErr, fmt.Errorf("the path of the disk full disruption must not be empty"))
	}

	hasCapacity := s.Capacity != ""
	hasRemaining := s.Remaining != ""

	if hasCapacity && hasRemaining {
		retErr = multierror.Append(retErr, fmt.Errorf("capacity and remaining are mutually exclusive, only one can be set"))
	}

	if !hasCapacity && !hasRemaining {
		retErr = multierror.Append(retErr, fmt.Errorf("one of capacity or remaining must be set"))
	}

	if hasCapacity {
		if err := validateCapacity(s.Capacity); err != nil {
			retErr = multierror.Append(retErr, err)
		}
	}

	if hasRemaining {
		if err := validateRemaining(s.Remaining); err != nil {
			retErr = multierror.Append(retErr, err)
		}
	}

	return retErr
}

func validateCapacity(capacity string) error {
	if !strings.HasSuffix(capacity, "%") {
		return fmt.Errorf("capacity must be a percentage suffixed with %%, got %q", capacity)
	}

	valueStr := strings.TrimSuffix(capacity, "%")

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return fmt.Errorf("capacity percentage must be an integer, got %q: %w", valueStr, err)
	}

	if value < 1 || value > 100 {
		return fmt.Errorf("capacity percentage must be between 1 and 100, got %d", value)
	}

	return nil
}

func validateRemaining(remaining string) error {
	qty, err := resource.ParseQuantity(remaining)
	if err != nil {
		return fmt.Errorf("remaining must be a valid Kubernetes resource quantity (e.g., 50Mi, 1Gi), got %q: %w", remaining, err)
	}

	if qty.Value() < 0 {
		return fmt.Errorf("remaining must not be negative, got %q", remaining)
	}

	return nil
}

// GenerateArgs generates injection or cleanup pod arguments for the given spec
func (s *DiskFullSpec) GenerateArgs() []string {
	args := []string{
		"disk-full",
		"--path",
		s.Path,
	}

	if s.Capacity != "" {
		args = append(args, "--capacity", s.Capacity)
	}

	if s.Remaining != "" {
		args = append(args, "--remaining", s.Remaining)
	}

	return args
}

// Explain returns a human-readable description of the disruption
func (s *DiskFullSpec) Explain() []string {
	explanation := fmt.Sprintf("spec.diskFull will fill the volume mounted at %s", s.Path)

	if s.Capacity != "" {
		explanation += fmt.Sprintf(" to %s of its total capacity", s.Capacity)
	}

	if s.Remaining != "" {
		explanation += fmt.Sprintf(", leaving only %s of free space", s.Remaining)
	}

	explanation += ", causing ENOSPC errors on subsequent write operations."

	return []string{"", explanation}
}
