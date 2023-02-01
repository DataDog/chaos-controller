// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

// +build ignore
#include "injection.bpf.h"
#if defined(__x86_64__) || defined(__TARGET_ARCH_x86)
#include "amd64/vmlinux.h"
#endif
#if defined(__aarch64__) || defined(__TARGET_ARCH_arm64)
#include "aarch64/vmlinux.h"
#endif
#if SC_PLATFORM == SC_PLATFORM_LINUX
#include <errno.h>
#endif
#include <bpf/bpf_helpers.h>
#if defined(__TARGET_ARCH_arm64) || defined(__TARGET_ARCH_x86)
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_core_read.h>
#endif

const volatile pid_t target_pid = 0;
const volatile pid_t exclude_pid;
const volatile char filter_path[61];

struct data_t {
    u32 ppid;
    u32 pid;
    u32 tid; 
    u32 id;
    char comm[100];
};

struct {
    __uint(type, BPF_MAP_TYPE_PERF_EVENT_ARRAY);
    __uint(max_entries, 1024);
    __type(key, int);
    __type(value, u32);
} events SEC(".maps");

SEC("kprobe/sys_openat")
int injection_bpftrace(struct pt_regs *ctx)
{
    struct data_t data = {};

    // Get data of the current process
    u32 ppid = 0;
    u32 pid = bpf_get_current_pid_tgid();
    if (pid == exclude_pid) {
        return 0;
    }
    u32 tid = bpf_get_current_pid_tgid() >> 32;
    u32 gid = bpf_get_current_uid_gid();

    if (pid != 1) {
        // Get parent pid
        struct task_struct *task;
        struct task_struct *real_parent;
        task = (struct task_struct *)bpf_get_current_task();
        bpf_probe_read(&real_parent, sizeof(real_parent), &task->real_parent);
        bpf_probe_read(&ppid, sizeof(ppid), &real_parent->tgid);

        // Allow only children and parent process.
        if (target_pid != 0 && ppid != target_pid && pid != target_pid) {
          return 0;
        }
    }

    if (ppid == exclude_pid || tid == exclude_pid) {
        return 0;
    }

// Exclude this part of code if the following variables are not defined.
// It allows the go program to compile without error.
#if defined(__TARGET_ARCH_arm64) || defined(__TARGET_ARCH_x86)
    // Allow only file with the desired prefix.
    struct pt_regs *real_regs = (struct pt_regs *)PT_REGS_PARM1(ctx);
    char *path = (char *)PT_REGS_PARM2_CORE(real_regs);
    char cmp_path_name[62];
    bpf_probe_read(&cmp_path_name, sizeof(cmp_path_name), path);
    char cmp_expected_path[62];
    bpf_probe_read(cmp_expected_path, sizeof(cmp_expected_path), filter_path);
    int filter_len = (int) (sizeof(filter_path) / sizeof(filter_path[0])) - 1;
   
    if (filter_len > 62) {
        return 0;
    }

    for (int i = 0; i < filter_len; ++i) {
      if (cmp_expected_path[i] == NULL)
        break;
      if (cmp_path_name[i] != cmp_expected_path[i])
        return 0;
    }
#endif

    data.ppid = ppid;
    data.pid = pid;
    data.tid = tid;
    data.id = gid;

    // Get command name
    bpf_get_current_comm(&data.comm, sizeof(data.comm));

    // Uncomment for debuging
    //bpf_printk("COMM: %s, Pid: %i, Tid: %i\n", &data.comm, data.pid, data.tid);
    //bpf_printk("COMM: %s, Parent Id: %i, Path: %s.\n", &data.comm, data.ppid, path);
    //bpf_printk("COMM:%s, Start injection", &data.comm);

    // Add the event to the ring buffer
    bpf_perf_event_output(ctx, &events, BPF_F_CURRENT_CPU, &data, 100);

    // Override return of process with an -ENOENT error.
    bpf_override_return(ctx, -ENOENT);

    return 0;
}

