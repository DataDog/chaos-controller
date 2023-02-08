// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

/* In Linux 5.4 asm_inline was introduced, but it's not supported by clang.
 * Redefine it to just asm to enable successful compilation.
 * see https://github.com/iovisor/bcc/commit/2d1497cde1cc9835f759a707b42dea83bee378b8 for more details
 */
#if defined(__x86_64__) || defined(__TARGET_ARCH_x86)
#include "./amd64/vmlinux.h"
#elif defined(__aarch64__) || defined(__TARGET_ARCH_arm64)
#include "./aarch64/vmlinux.h"
#endif
#include <errno.h>
#ifdef asm_inline
#undef asm_inline
#define asm_inline asm
#endif
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_core_read.h>

char LICENSE[] SEC("license") = "Dual BSD/GPL";