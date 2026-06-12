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

// AT_FDCWD sentinel value; openat resolves relative paths against the CWD only
// when this value is passed as dirfd.
#ifndef AT_FDCWD
#define AT_FDCWD -100
#endif

// check_basename_prefix returns 1 if rel_buf starts with the basename suffix of
// filter_path (the part after the last '/').
static __always_inline int check_basename_prefix(const char *rel_buf)
{
    int last_slash = 0;
    for (int i = 0; i < 60; i++) {
        if (filter_path[i] == '\0') break;
        if (filter_path[i] == '/') last_slash = i;
    }
    for (int i = 0; i < 60; i++) {
        int fi = last_slash + 1 + i;
        if (fi >= 61) break;
        if (filter_path[fi & 0x3f] == '\0') break;
        if (rel_buf[i] != filter_path[fi & 0x3f]) return 0;
    }
    return 1;
}

// check_relative_path returns 1 if a relative openat call should be disrupted.
// Two checks are attempted in order:
//   1. CWD inode == filter_dir_inode (parent of filter_path) AND rel_path starts
//      with the basename — handles "cwd=/parent && openat(AT_FDCWD, "dir/file")".
//   2. CWD inode == filter_dir_inode2 (filter_path itself, set when it is a dir)
//      AND rel_path does not start with ".." — handles "cwd=/dir && openat(AT_FDCWD, "file")".
// Only called when dirfd == AT_FDCWD; other dirfds are not supported.
// Using the CWD inode rather than walking the dentry chain works inside containers
// because Kubernetes volumes are bind-mounted: host inode == in-container inode.
static int check_relative_path(int dirfd, const char *rel_path)
{
    if (dirfd != AT_FDCWD) return 0;
    if (filter_dir_inode == 0 && filter_dir_inode2 == 0) return 0;

    // Read CWD inode and device via task->fs->pwd.dentry->d_inode.
    struct task_struct *task = (struct task_struct *)bpf_get_current_task();
    struct fs_struct *fs_ptr;
    bpf_probe_read_kernel(&fs_ptr, sizeof(fs_ptr), &task->fs);
    struct path pwd;
    bpf_probe_read_kernel(&pwd, sizeof(pwd), &fs_ptr->pwd);
    struct inode *inode_ptr;
    bpf_probe_read_kernel(&inode_ptr, sizeof(inode_ptr), &pwd.dentry->d_inode);
    u64 ino = 0;
    bpf_probe_read_kernel(&ino, sizeof(ino), &inode_ptr->i_ino);
    struct super_block *sb_ptr = NULL;
    bpf_probe_read_kernel(&sb_ptr, sizeof(sb_ptr), &inode_ptr->i_sb);
    u32 dev = 0;
    bpf_probe_read_kernel(&dev, sizeof(dev), &sb_ptr->s_dev);

    char rel_buf[62] = {};
    bpf_probe_read(rel_buf, sizeof(rel_buf) - 1, rel_path);

    // Check 1: parent inode + basename prefix.
    if (filter_dir_inode != 0 && ino == filter_dir_inode &&
        (filter_dir_dev == 0 || dev == filter_dir_dev) &&
        check_basename_prefix(rel_buf)) return 1;

    // Check 2: exact directory inode — match any file that does not escape via "..".
    if (filter_dir_inode2 != 0 && ino == filter_dir_inode2 &&
        (filter_dir_dev2 == 0 || dev == filter_dir_dev2) &&
        !(rel_buf[0] == '.' && rel_buf[1] == '.')) return 1;

    return 0;
}

// do_filter_by_process returns 1 if the current process should be excluded (filtered out),
// 0 if it should be disrupted.
static __always_inline int do_filter_by_process(u32 pid, u32 ppid)
{
    if (use_cgroup_filter) {
        return bpf_current_task_under_cgroup(&target_cgroup, 0) != 1 ? 1 : 0;
    } else if (target_pid != 0) {
        return (ppid != target_pid && pid != target_pid) ? 1 : 0;
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

SEC("kprobe/sys_openat")
int injection_disk_failure(struct pt_regs *ctx)
{
    struct data_t data = {};

    u32 ppid = 0;
    u32 pid = bpf_get_current_pid_tgid();
    if (pid == exclude_pid) {
        return 0;
    }
    u32 tid = bpf_get_current_pid_tgid() >> 32;
    u32 gid = bpf_get_current_uid_gid();

    if (pid != 1) {
        // Get parent pid (needed for cgroupv1 PID filter and exclude_pid check below)
        struct task_struct *task;
        struct task_struct *real_parent;
        task = (struct task_struct *)bpf_get_current_task();
        bpf_probe_read(&real_parent, sizeof(real_parent), &task->real_parent);
        bpf_probe_read(&ppid, sizeof(ppid), &real_parent->tgid);
    }

    if (do_filter_by_process(pid, ppid)) return 0;

    if (ppid == exclude_pid || tid == exclude_pid) {
        return 0;
    }

// Exclude this part of code if the following variables are not defined.
// It allows the go program to compile without error.
#if defined(__TARGET_ARCH_arm64) || defined(__TARGET_ARCH_x86)
    // __x64_sys_openat / __arm64_sys_openat wrap the inner syscall args in a
    // pt_regs struct passed as PARM1. Read dirfd and path from the inner regs.
    struct pt_regs *inner_regs = (struct pt_regs *)PT_REGS_PARM1(ctx);
    int dirfd = (int)(long)PT_REGS_PARM1_CORE(inner_regs);
    char *path = (char *)PT_REGS_PARM2_CORE(inner_regs);
    char cmp_path_name[62];
    bpf_probe_read(&cmp_path_name, sizeof(cmp_path_name), path);

    if (cmp_path_name[0] == '/') {
        // Absolute path: compare raw path argument against filter prefix directly.
        char cmp_expected_path[62];
        bpf_probe_read(cmp_expected_path, sizeof(cmp_expected_path), (const void *)filter_path);
        int filter_len = (int)(sizeof(filter_path) / sizeof(filter_path[0])) - 1;
        if (filter_len > 62) return 0;
        for (int i = 0; i < filter_len; ++i) {
            if (cmp_expected_path[i] == NULL) break;
            if (cmp_path_name[i] != cmp_expected_path[i]) return 0;
        }
    } else {
        // Relative path: compare CWD inode against the filter. dirfd is passed so
        // that opens with a non-AT_FDCWD dirfd are skipped (CWD is irrelevant there).
        if (!check_relative_path(dirfd, path)) return 0;
    }
#endif

    if (do_probability_check()) return 0;

    data.ppid = ppid;
    data.pid = pid;
    data.tid = tid;
    data.id = gid;

    // Get command name
    bpf_get_current_comm(&data.comm, sizeof(data.comm));

    // Add the event to the ring buffer
    bpf_perf_event_output(ctx, &events, BPF_F_CURRENT_CPU, &data, 100);

    // Override return of process with an -ENOENT error.
    bpf_override_return(ctx, -exit_code);

    return 0;
}
