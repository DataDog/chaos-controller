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

// Debug counters read periodically by the Go loader to diagnose path filter
// behaviour without relying on tracefs (which is often blocked by node policy).
// 0: abs path matched; 1: abs path missed; 2: rel path, no inode filter;
// 3: rel path, inode matched (disrupted); 4: rel path, inode missed;
// 5: rel path, dirfd inode == filter_dir_inode but basename mismatch;
// 6: rel path, fdtable lookup returned null fd (silent drop).
// 7: cgroup filter returned 1 (in cgroup);
// 8: cgroup filter returned 0 (not in cgroup — PID fallback applied);
// 9: cgroup filter returned error (negative — PID fallback applied).
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 12);
    __type(key, u32);
    __type(value, u64);
} debug_counters SEC(".maps");

#define DBG_ABS_HIT        0
#define DBG_ABS_MISS       1
#define DBG_REL_NO_FILTER  2
#define DBG_REL_HIT        3
#define DBG_REL_MISS       4
#define DBG_REL_INO_MATCH  5
#define DBG_REL_NULL_FD    6
#define DBG_CGROUP_HIT     7
#define DBG_CGROUP_MISS    8
#define DBG_CGROUP_ERR     9

static __always_inline void dbg_inc(u32 idx)
{
    u64 *val = bpf_map_lookup_elem(&debug_counters, &idx);
    if (val) __sync_fetch_and_add(val, 1);
}

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

// AT_FDCWD sentinel: openat resolves relative paths against CWD only when
// this value is passed as dirfd.
#ifndef AT_FDCWD
#define AT_FDCWD -100
#endif

// check_basename_prefix returns 1 if rel_buf starts with the basename of filter_path.
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

// check_relative_path returns 1 if a relative openat should be disrupted.
// Handles both AT_FDCWD (match against CWD inode) and explicit dirfd (match
// against the inode of the directory the fd points to). Using inodes works
// inside containers because Kubernetes volumes are bind-mounted: the host inode
// and the in-container inode are identical.
static int check_relative_path(int dirfd, const char *rel_path)
{
    if (filter_dir_inode == 0 && filter_dir_inode2 == 0) {
        dbg_inc(DBG_REL_NO_FILTER);
        return 0;
    }

    u64 ino = 0;
    u32 dev = 0;

    if (dirfd == AT_FDCWD) {
        struct task_struct *task = (struct task_struct *)bpf_get_current_task();
        struct fs_struct *fs_ptr;
        bpf_probe_read_kernel(&fs_ptr, sizeof(fs_ptr), &task->fs);
        struct path pwd;
        bpf_probe_read_kernel(&pwd, sizeof(pwd), &fs_ptr->pwd);
        struct inode *inode_ptr;
        bpf_probe_read_kernel(&inode_ptr, sizeof(inode_ptr), &pwd.dentry->d_inode);
        bpf_probe_read_kernel(&ino, sizeof(ino), &inode_ptr->i_ino);
        struct super_block *sb_ptr = NULL;
        bpf_probe_read_kernel(&sb_ptr, sizeof(sb_ptr), &inode_ptr->i_sb);
        bpf_probe_read_kernel(&dev, sizeof(dev), &sb_ptr->s_dev);
    } else if (dirfd >= 0) {
        // Look up the inode of the directory referenced by the explicit dirfd.
        u32 ufd = (u32)dirfd;
        if (ufd >= 1024) return 0;

        struct task_struct *task = (struct task_struct *)bpf_get_current_task();
        struct files_struct *files_ptr;
        bpf_probe_read_kernel(&files_ptr, sizeof(files_ptr), &task->files);
        if (!files_ptr) return 0;
        struct fdtable *fdt_ptr;
        bpf_probe_read_kernel(&fdt_ptr, sizeof(fdt_ptr), &files_ptr->fdt);
        if (!fdt_ptr) return 0;
        struct file **fd_arr;
        bpf_probe_read_kernel(&fd_arr, sizeof(fd_arr), &fdt_ptr->fd);
        if (!fd_arr) return 0;
        struct file *f = NULL;
        bpf_probe_read_kernel(&f, sizeof(f), (void *)((__u64)fd_arr + (__u64)ufd * sizeof(struct file *)));
        if (!f) {
            dbg_inc(DBG_REL_NULL_FD);
            return 0;
        }
        struct inode *inode_ptr;
        bpf_probe_read_kernel(&inode_ptr, sizeof(inode_ptr), &f->f_inode);
        if (!inode_ptr) return 0;
        bpf_probe_read_kernel(&ino, sizeof(ino), &inode_ptr->i_ino);
        struct super_block *sb_ptr = NULL;
        bpf_probe_read_kernel(&sb_ptr, sizeof(sb_ptr), &inode_ptr->i_sb);
        if (!sb_ptr) return 0;
        bpf_probe_read_kernel(&dev, sizeof(dev), &sb_ptr->s_dev);
    } else {
        return 0;
    }

    char rel_buf[62] = {};
    bpf_probe_read_user(rel_buf, sizeof(rel_buf) - 1, rel_path);

    // Track when our dirfd's inode matches the filter inode (regardless of basename)
    // to distinguish "wrong directory" from "right directory but wrong filename".
    if ((filter_dir_inode != 0 && ino == filter_dir_inode) ||
        (filter_dir_inode2 != 0 && ino == filter_dir_inode2))
        dbg_inc(DBG_REL_INO_MATCH);

    // Check 1: dir == parent of filter_path AND rel_path starts with its basename.
    if (filter_dir_inode != 0 && ino == filter_dir_inode &&
        (filter_dir_dev == 0 || dev == filter_dir_dev) &&
        check_basename_prefix(rel_buf)) {
        dbg_inc(DBG_REL_HIT);
        return 1;
    }

    // Check 2: dir == filter_path itself (directory target) AND rel_path doesn't escape.
    if (filter_dir_inode2 != 0 && ino == filter_dir_inode2 &&
        (filter_dir_dev2 == 0 || dev == filter_dir_dev2) &&
        !(rel_buf[0] == '.' && rel_buf[1] == '.')) {
        dbg_inc(DBG_REL_HIT);
        return 1;
    }

    dbg_inc(DBG_REL_MISS);
    return 0;
}

// do_filter_by_process returns 1 if the current process should be excluded (filtered out),
// 0 if it should be disrupted.
// tgid is the kernel TGID (== userspace PID from getpid()); ppid is the parent's TGID.
static __always_inline int do_filter_by_process(u32 tgid, u32 ppid)
{
    if (use_cgroup_filter) {
        int in_cgroup = bpf_current_task_under_cgroup(&target_cgroup, 0);
        if (in_cgroup == 1) {
            dbg_inc(DBG_CGROUP_HIT);
            return 0;  // in cgroup → disrupt
        }
        // cgroup filter returned 0 (not in cgroup) or negative (error).
        // Fall back to PID filter so we still catch processes that are direct
        // children of the container init (e.g. dd run from container's PID 1
        // shell) when bpf_current_task_under_cgroup fails on this kernel.
        if (in_cgroup < 0) {
            dbg_inc(DBG_CGROUP_ERR);
        } else {
            dbg_inc(DBG_CGROUP_MISS);
        }
        if (target_pid != 0 && (ppid == target_pid || tgid == target_pid)) {
            return 0;  // PID fallback matched → disrupt
        }
        return 1;  // exclude
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
SEC("fmod_ret/__arm64_sys_openat")
#else
SEC("fmod_ret/__x64_sys_openat")
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

    // Read openat arguments from inner pt_regs. Both __arm64_sys_openat and
    // __x64_sys_openat wrap syscall args in a (const struct pt_regs *) passed
    // as their only argument, so PARM1(ctx) is the inner regs pointer.
    struct pt_regs *inner_regs = (struct pt_regs *)(unsigned long)PT_REGS_PARM1_CORE(ctx);
    int dirfd = (int)(long)PT_REGS_PARM1_CORE(inner_regs);
    const char *path = (const char *)PT_REGS_PARM2_CORE(inner_regs);

    char cmp_path_name[62];
    bpf_probe_read_user(cmp_path_name, sizeof(cmp_path_name), path);

    if (cmp_path_name[0] == '/') {
        char cmp_expected_path[62];
        bpf_probe_read(cmp_expected_path, sizeof(cmp_expected_path), (const void *)filter_path);
        int filter_len = (int)(sizeof(filter_path) / sizeof(filter_path[0])) - 1;
        if (filter_len > 62) return 0;
        int abs_match = 1;
        for (int i = 0; i < filter_len; ++i) {
            if (cmp_expected_path[i] == '\0') break;
            if (cmp_path_name[i] != cmp_expected_path[i]) { abs_match = 0; break; }
        }
        if (!abs_match) {
            dbg_inc(DBG_ABS_MISS);
            return 0;
        }
        dbg_inc(DBG_ABS_HIT);
    } else {
        if (!check_relative_path(dirfd, path)) return 0;
    }

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

    return -(int)exit_code;
}
