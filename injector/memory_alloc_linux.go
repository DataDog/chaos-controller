// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

//go:build linux

package injector

import "syscall"

// mmapAnonymous allocates anonymous memory pages using mmap with MAP_POPULATE
// to ensure physical pages are allocated immediately
func mmapAnonymous(size int) ([]byte, error) {
	return syscall.Mmap(-1, 0, size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_ANONYMOUS|syscall.MAP_PRIVATE|syscall.MAP_POPULATE)
}

// munmapMemory releases memory previously allocated with mmapAnonymous
func munmapMemory(data []byte) error {
	return syscall.Munmap(data)
}
