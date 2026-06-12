// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

//go:build arm64
// +build arm64

package ebpf

// SysOpenat is the kprobe target for disk-failure disruption.
// __arm64_sys_openat is tagged ALLOW_ERROR_INJECTION, which is required for
// bpf_override_return to override the syscall return value.
const SysOpenat = "__arm64_sys_openat"
