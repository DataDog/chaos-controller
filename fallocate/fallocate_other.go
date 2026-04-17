// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

// Fallocate fallback for platforms that don't support fallocate or F_PREALLOCATE.
// Based on https://github.com/detailyang/go-fallocate (MIT License).

//go:build !linux && !darwin

package fallocate

import "os"

// Fallocate allocates disk space by writing zeros. This is the fallback
// implementation for platforms without native fallocate support.
func Fallocate(file *os.File, offset int64, length int64) error {
	return fallocateWrite(file, offset, length)
}
