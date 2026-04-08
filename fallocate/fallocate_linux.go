// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

// Fallocate implementation for Linux using the fallocate(2) syscall.
// Based on https://github.com/detailyang/go-fallocate (MIT License).
// Falls back to writing zeros if the filesystem does not support fallocate.

package fallocate

import (
	"errors"
	"os"
	"syscall"
)

// Fallocate allocates disk space for the given file without writing data.
// If the filesystem does not support fallocate (EOPNOTSUPP), it falls back
// to writing zeros.
func Fallocate(file *os.File, offset int64, length int64) error {
	if length == 0 {
		return nil
	}

	err := syscall.Fallocate(int(file.Fd()), 0, offset, length)
	if err == nil {
		return nil
	}

	// Fall back to writing zeros on unsupported filesystems (e.g., NFS, some FUSE)
	if errors.Is(err, syscall.EOPNOTSUPP) || errors.Is(err, syscall.ENOTSUP) {
		return fallocateWrite(file, offset, length)
	}

	return err
}
