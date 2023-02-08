// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

//go:build arm64
// +build arm64

package ebpf

const SYS_OPENAT = "__arm64_sys_openat"
const DISK_FAILURE_OBJ_NAME = "bpf-disk-failure-arm64.bpf.o"
