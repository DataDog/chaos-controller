// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package v1beta1

import (
	"fmt"
	"strings"
)

// OpenatSyscallSpec syscall specs
type OpenatSyscallSpec struct {
	// Refer to this documentation: https://linux.die.net/man/2/open
	// +kubebuilder:validation:Enum=EACCES;EDQUOT;EEXIST;EFAULT;EFBIG;EINTR;EISDIR;ELOOP;EMFILE;ENAMETOOLONG;ENFILE;ENODEV;ENOENT;ENOMEM;ENOSPC;ENOTDIR;ENXIO;EOVERFLOW;EPERM;EROFS;ETXTBSY;EWOULDBLOCK
	// +ddmark:validation:Enum=EACCES;EDQUOT;EEXIST;EFAULT;EFBIG;EINTR;EISDIR;ELOOP;EMFILE;ENAMETOOLONG;ENFILE;ENODEV;ENOENT;ENOMEM;ENOSPC;ENOTDIR;ENXIO;EOVERFLOW;EPERM;EROFS;ETXTBSY;EWOULDBLOCK
	ExitCode string `json:"exitCode"`
}

// GetExitCodeInt return the integer value of a linux exit code.
func (oss *OpenatSyscallSpec) GetExitCodeInt() int {
	switch oss.ExitCode {
	case "EACCES":
		return 13
	case "EDQUOT":
		return 122
	case "EEXIST":
		return 17
	case "EFAULT":
		return 14
	case "EFBIG":
		return 27
	case "EINTR":
		return 4
	case "EISDIR":
		return 21
	case "ELOOP":
		return 40
	case "EMFILE":
		return 24
	case "ENAMETOOLONG":
		return 36
	case "ENFILE":
		return 23
	case "ENODEV":
		return 19
	case "ENOENT":
		return 2
	case "ENOMEM":
		return 12
	case "ENOSPC":
		return 28
	case "ENOTDIR":
		return 20
	case "ENXIO":
		return 6
	case "EOVERFLOW":
		return 75
	case "EPERM":
		return 1
	case "EROFS":
		return 30
	case "ETXTBSY":
		return 26
	case "EWOULDBLOCK":
		return 11
	default:
		return 0
	}
}

// DiskFailureSpec represents a disk failure disruption
type DiskFailureSpec struct {
	Path string `json:"path"`
	// +nullable
	OpenatSyscall *OpenatSyscallSpec `json:"openat,omitempty"`
}

const MaxDiskPathCharacters = 62

// Validate validates args for the given disruption
func (s *DiskFailureSpec) Validate() error {
	path := strings.TrimSpace(s.Path)

	if path == "" {
		return fmt.Errorf("the path of the disk failure disruption must not be empty")
	}

	if len(path) > MaxDiskPathCharacters {
		return fmt.Errorf("the path of the disk failure disruption must not exceed %d characters", MaxDiskPathCharacters)
	}

	return nil
}

// GenerateArgs generates injection or cleanup pod arguments for the given spec
func (s *DiskFailureSpec) GenerateArgs() (args []string) {
	args = append(args, "disk-failure")
	path := strings.TrimSpace(s.Path)
	args = append(args, "--path", path)

	if s.OpenatSyscall != nil {
		if s.OpenatSyscall.ExitCode != "" {
			args = append(args, "--exit-code", s.OpenatSyscall.ExitCode)
		}
	}

	return args
}
