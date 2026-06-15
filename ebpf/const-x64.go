// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

//go:build amd64
// +build amd64

package ebpf

// SysOpenat is unused on x86_64 (fmod_ret attachment uses the SEC annotation).
const SysOpenat = ""

// UseKprobe is false on x86_64: kprobes are [FTRACE]-based and bpf_override_return
// silently fails. Use fmod_ret + AttachGeneric instead.
const UseKprobe = false

