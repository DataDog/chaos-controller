// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

// +build ignore
#include "injection.bpf.h"

const volatile pid_t target_pid = 0;
const volatile pid_t exclude_pid;
const volatile int use_cgroup_filter = 0;
const volatile char filter_path[61];
// Inode of filter_path's parent directory. When non-zero, enables filtering of
// relative openat calls by comparing the process CWD inode against this value.
// Works correctly inside containers because Kubernetes volumes are bind-mounted:
// the host inode and the in-container inode are identical.
const volatile u64 filter_dir_inode = 0;
// Device ID paired with filter_dir_inode. Inodes are only unique within a device,
// so checking both prevents false matches on bind-mounted or multi-filesystem targets.
const volatile u32 filter_dir_dev = 0;
// Second inode/device pair: set when filter_path is itself a directory. When the
// CWD matches this inode/device, any relative open (except ".." escapes) is
// in-scope. This handles "cd /mnt/data && cat file" alongside the parent+basename
// case covered by filter_dir_inode (i.e. "cwd=/mnt && cat data/file").
const volatile u64 filter_dir_inode2 = 0;
const volatile u32 filter_dir_dev2 = 0;

// Populated from userspace with the container's cgroupv2 directory fd.
// bpf_current_task_under_cgroup() matches the process itself AND any sub-cgroup
// (e.g. containerd exec-<id> sub-cgroups created by kubectl exec).
struct {
    __uint(type, BPF_MAP_TYPE_CGROUP_ARRAY);
    __uint(max_entries, 1);
    __type(key, u32);
    __type(value, u32);
} target_cgroup SEC(".maps");

const volatile pid_t exit_code = ENOENT;
const volatile int probability = 100;

unsigned int hits = 0;
unsigned int disruptedHits = 0;

struct data_t {
    u32 ppid;
    // tid is the kernel thread ID (what userspace calls TID via gettid()).
    // tgid is the kernel thread-group ID (what userspace calls PID via getpid()).
    u32 tid;
    u32 tgid;
    u32 id;
    char comm[100];
};

struct {
    __uint(type, BPF_MAP_TYPE_PERF_EVENT_ARRAY);
    __uint(max_entries, 1024);
    __type(key, int);
    __type(value, u32);
} events SEC(".maps");


// do_filter_by_process returns 1 if the current process should be excluded (filtered out),
// 0 if it should be disrupted.
// tgid is the kernel TGID (== userspace PID from getpid()); ppid is the parent's TGID.
static __always_inline int do_filter_by_process(u32 tgid, u32 ppid)
{
    if (use_cgroup_filter) {
        return bpf_current_task_under_cgroup(&target_cgroup, 0) != 1 ? 1 : 0;
    } else if (target_pid != 0) {
        return (ppid != target_pid && tgid != target_pid) ? 1 : 0;
    }
    return 0;
}

// do_probability_check returns 1 if the event should be skipped due to probability sampling.
static __always_inline int do_probability_check()
{
    if (probability == 100) return 0;
    if (hits != 0) {
        unsigned long long scaled = disruptedHits * 100;
        if ((scaled / hits) > probability) {
            hits++;
            return 1;
        }
    }
    hits++;
    disruptedHits++;
    return 0;
}

#if defined(__TARGET_ARCH_arm64)
SEC("kprobe/__arm64_sys_openat")
#else
SEC("kprobe/__x64_sys_openat")
#endif
int injection_disk_failure(struct pt_regs *ctx)
{
    struct data_t data = {};

    u32 ppid = 0;
    // bpf_get_current_pid_tgid() returns (tgid << 32 | tid).
    // Lower 32 bits = kernel TID (thread ID); upper 32 bits = kernel TGID (process ID).
    u64 pid_tgid = bpf_get_current_pid_tgid();
    u32 tid = (u32)pid_tgid;            // kernel TID  == userspace TID (gettid)
    u32 tgid = (u32)(pid_tgid >> 32);  // kernel TGID == userspace PID (getpid)
    // Exclude the bpf-disk-failure binary itself (and all its threads) to prevent
    // the injector from disrupting its own file operations.
    if (tgid == exclude_pid) {
        return 0;
    }
    u32 gid = bpf_get_current_uid_gid();

    if (tgid != 1) {
        // Get parent pid (needed for cgroupv1 PID filter and exclude_pid check below)
        struct task_struct *task;
        struct task_struct *real_parent;
        task = (struct task_struct *)bpf_get_current_task();
        bpf_probe_read(&real_parent, sizeof(real_parent), &task->real_parent);
        bpf_probe_read(&ppid, sizeof(ppid), &real_parent->tgid);
    }

    if (do_filter_by_process(tgid, ppid)) return 0;

    if (ppid == exclude_pid || tgid == exclude_pid) {
        return 0;
    }

    // TODO: path filtering temporarily removed to verify that bpf_override_return
    // works on the target ARM64 kernels before re-adding argument parsing.

    if (do_probability_check()) return 0;

    data.ppid = ppid;
    data.tid = tid;
    data.tgid = tgid;
    data.id = gid;

    // Get command name
    bpf_get_current_comm(&data.comm, sizeof(data.comm));

    // Add the event to the ring buffer
    bpf_perf_event_output(ctx, &events, BPF_F_CURRENT_CPU, &data, 100);

    printt("disk-failure: disrupted tgid=%d rc=-%d\n", tgid, (int)exit_code);

    bpf_override_return(ctx, -exit_code);
    return 0;
}
