// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

// Fallocate implementation for Darwin using F_PREALLOCATE fcntl.
// Based on https://github.com/detailyang/go-fallocate (MIT License).

package fallocate

import (
	"os"
	"syscall"
	"unsafe"
)

// Fallocate pre-allocates disk space for the given file on macOS.
func Fallocate(file *os.File, offset int64, length int64) error {
	if length == 0 {
		return nil
	}

	fst := syscall.Fstore_t{
		Flags:      syscall.F_ALLOCATECONTIG,
		Posmode:    syscall.F_PREALLOCATE,
		Offset:     0,
		Length:     offset + length,
		Bytesalloc: 0,
	}

	// Try contiguous allocation first, fall back to non-contiguous
	// See: https://lists.apple.com/archives/darwin-dev/2007/Dec/msg00040.html
	_, _, err := syscall.Syscall(syscall.SYS_FCNTL, file.Fd(), syscall.F_PREALLOCATE, uintptr(unsafe.Pointer(&fst)))
	if err != syscall.Errno(0x0) {
		fst.Flags = syscall.F_ALLOCATEALL
		_, _, _ = syscall.Syscall(syscall.SYS_FCNTL, file.Fd(), syscall.F_PREALLOCATE, uintptr(unsafe.Pointer(&fst)))
	}

	return syscall.Ftruncate(int(file.Fd()), fst.Length)
}
