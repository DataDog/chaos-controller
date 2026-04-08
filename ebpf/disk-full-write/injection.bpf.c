// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

// Intercepts write syscalls (write, pwrite64) and returns a configurable error
// code (default ENOSPC) with configurable probability for a target process and
// its children. Used by the disk-full disruption for eBPF-based write failure
// injection alongside real volume fill.

// +build ignore
#include "injection.bpf.h"

const volatile pid_t target_pid = 0;
const volatile pid_t exclude_pid;
const volatile pid_t exit_code = ENOSPC;
const volatile int probability = 100;

unsigned int hits = 0;
unsigned int disruptedHits = 0;

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

SEC("kprobe/sys_write")
int injection_disk_full_write(struct pt_regs *ctx)
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

    // No path filtering for write syscalls — when a disk is full, ALL writes
    // to the filesystem fail with ENOSPC, regardless of the target file.

    if (probability != 100) {
        if (hits != 0) {
            unsigned long long scaled_disruptedHits = disruptedHits * 100;
            unsigned long long scaled_hits = hits;

            if ((scaled_disruptedHits / scaled_hits) > probability) {
                hits++;
                return 0;
            }
        }

        hits++;
        disruptedHits++;
    }

    data.ppid = ppid;
    data.pid = pid;
    data.tid = tid;
    data.id = gid;

    // Get command name
    bpf_get_current_comm(&data.comm, sizeof(data.comm));

    // Add the event to the ring buffer
    bpf_perf_event_output(ctx, &events, BPF_F_CURRENT_CPU, &data, 100);

    // Override return of write syscall with error code (default -ENOSPC)
    bpf_override_return(ctx, -exit_code);

    return 0;
}
