// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector

import "syscall"

// devToKernel converts a userspace st_dev (glibc encode_dev encoding) to the
// kernel dev_t value (MKDEV: major<<20 | minor) stored in super_block.s_dev.
// The BPF filter_dir_dev variables are compared against s_dev, so the injector
// must pass the kernel-encoded value, not the raw st_dev.
func devToKernel(st *syscall.Stat_t) uint32 {
	d := st.Dev
	major := uint32((d&0x00000000000fff00)>>8) | uint32((d&0xfffff00000000000)>>32)
	minor := uint32((d&0x00000000000000ff)>>0) | uint32((d&0x00000ffffff00000)>>12)

	return (major << 20) | minor
}
