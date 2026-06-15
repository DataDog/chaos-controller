// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

//go:build arm64
// +build arm64

package ebpf

// SysOpenat is the kprobe target on ARM64. Kprobes on __arm64_sys_openat use
// the traditional int3 mechanism (not [FTRACE]), so bpf_override_return works.
const SysOpenat = "__arm64_sys_openat"

// UseKprobe tells main.go to attach via AttachKprobe instead of AttachGeneric.
const UseKprobe = true

