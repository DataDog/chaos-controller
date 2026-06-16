// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package ebpf

import "runtime"

// SysOpenat is the kprobe target for disk-failure disruption.
// bpf_override_return requires ALLOW_ERROR_INJECTION on this function.
func SysOpenat() string {
	if runtime.GOARCH == "arm64" {
		return "__arm64_sys_openat"
	}
	return "__x64_sys_openat"
}
