// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

//go:build !linux

package injector

import "syscall"

// devToKernel is a no-op stub on non-Linux platforms where the eBPF injector
// does not run.
func devToKernel(st *syscall.Stat_t) uint32 {
	return uint32(st.Dev)
}
