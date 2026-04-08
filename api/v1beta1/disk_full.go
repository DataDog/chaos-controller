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
	// WriteSyscall optionally enables eBPF-based write syscall interception to return errors
	// with configurable probability. This runs in addition to the volume fill.
	// +nullable
	WriteSyscall *WriteSyscallSpec `json:"writeSyscall,omitempty"`
}

// WriteSyscallSpec configures eBPF-based interception of write syscalls (write, pwrite64)
// to return a configurable error code with a given probability.
type WriteSyscallSpec struct {
	// ExitCode is the errno to return on intercepted write syscalls.
	// +kubebuilder:validation:Enum=ENOSPC;EDQUOT;EIO;EROFS;EFBIG;EPERM;EACCES
	// +kubebuilder:default=ENOSPC
	ExitCode string `json:"exitCode,omitempty" chaos_validate:"omitempty,oneofci=ENOSPC EDQUOT EIO EROFS EFBIG EPERM EACCES"`
	// Probability is the percentage of write syscalls to fail (e.g., "50%"). Default: "100%".
	Probability string `json:"probability,omitempty"`
}

// GetExitCodeInt returns the integer value of the configured errno.
func (s *WriteSyscallSpec) GetExitCodeInt() int {
	switch s.ExitCode {
	case "ENOSPC":
		return 28
	case "EDQUOT":
		return 122
	case "EIO":
		return 5
	case "EROFS":
		return 30
	case "EFBIG":
		return 27
	case "EPERM":
		return 1
	case "EACCES":
		return 13
	default:
		return 28 // ENOSPC
	}
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

	if s.WriteSyscall != nil {
		if err := validateWriteSyscallProbability(s.WriteSyscall.Probability); err != nil {
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

func validateWriteSyscallProbability(probability string) error {
	if probability == "" {
		return nil
	}

	if !strings.HasSuffix(probability, "%") {
		return fmt.Errorf("writeSyscall probability must be a percentage suffixed with %%, got %q", probability)
	}

	valueStr := strings.TrimSuffix(probability, "%")

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return fmt.Errorf("writeSyscall probability must be an integer, got %q: %w", valueStr, err)
	}

	if value < 1 || value > 100 {
		return fmt.Errorf("writeSyscall probability must be between 1 and 100, got %d", value)
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

	if s.WriteSyscall != nil {
		exitCode := s.WriteSyscall.ExitCode
		if exitCode == "" {
			exitCode = "ENOSPC"
		}

		args = append(args, "--write-exit-code", exitCode)

		probability := s.WriteSyscall.Probability
		if probability == "" {
			probability = "100%"
		}

		args = append(args, "--write-probability", probability)
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

	if s.WriteSyscall != nil {
		exitCode := s.WriteSyscall.ExitCode
		if exitCode == "" {
			exitCode = "ENOSPC"
		}

		probability := s.WriteSyscall.Probability
		if probability == "" {
			probability = "100%"
		}

		explanation += fmt.Sprintf(" Additionally, write syscalls will be intercepted via eBPF and return %s %s of the time.", exitCode, probability)
	}

	return []string{"", explanation}
}
