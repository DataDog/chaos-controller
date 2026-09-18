// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

// +build ignore
#include "injection.bpf.h"

const volatile unsigned int target_pid_ns_inum = 0;
const volatile pid_t exclude_pid;
const volatile char filter_path[61];
const volatile pid_t exit_code = ENOENT;
const volatile int probability = 100;

unsigned int hits = 0;
unsigned int disruptedHits = 0;

struct data_t {
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

// openat() dirfd value meaning "relative to the current working directory".
#define AT_FDCWD -100

#ifndef offsetof
#define offsetof(TYPE, MEMBER) __builtin_offsetof(TYPE, MEMBER)
#endif
// container_of recovers the enclosing struct mount from its embedded vfsmount.
#ifndef container_of
#define container_of(ptr, type, member) \
    ((type *)((void *)(ptr) - offsetof(type, member)))
#endif

// Maximum number of path components walked when resolving the cwd, and the
// per-component name cap. filter_path is at most 60 chars. The depth is kept
// small on purpose: container working directories are shallow and a larger value
// makes the verifier explore too many states (the cost is super-linear in depth).
// CWD_CHAIN_SIZE must be a power of two >= CWD_MAX_DEPTH: the chain array is
// indexed with (idx & (CWD_CHAIN_SIZE-1)) to give the verifier a mask-based
// bounds proof. Keeping CWD_MAX_DEPTH=10 (loop bound) separate from
// CWD_CHAIN_SIZE=16 (array/mask bound) avoids both index corruption and the
// verifier E2BIG that results from increasing the loop iteration count.
#define CWD_MAX_DEPTH  10
#define CWD_CHAIN_SIZE 16
#define CWD_NAME_BUF   64

// Resolved-path buffer. PATH_MASK keeps the running write offset provably in
// bounds (offset & PATH_MASK) so the verifier does not have to track it
// precisely; PATH_BUF leaves room for one full CWD_NAME_BUF write at the highest
// masked offset. The offset wraps past PATH_MASK, but the filter is at most 60
// chars so only the first bytes of the path are ever compared.
#define PATH_BUF  320
#define PATH_MASK 255

// Scratch space for cwd resolution. Kept in a per-CPU array map instead of on
// the stack: the dentry chain and the path buffer exceed the 512 byte BPF stack
// limit.
#if defined(__TARGET_ARCH_arm64) || defined(__TARGET_ARCH_x86)
struct cwd_scratch {
    struct dentry *chain[CWD_CHAIN_SIZE];
    char filter[62];
    char path[PATH_BUF];
};

struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(max_entries, 1);
    __type(key, u32);
    __type(value, struct cwd_scratch);
} cwd_scratch_map SEC(".maps");
#endif

// abs_path_matches_filter applies the prefix filter to an absolute path.
#if defined(__TARGET_ARCH_arm64) || defined(__TARGET_ARCH_x86)
static __always_inline int abs_path_matches_filter(const char *p)
{
    for (int i = 0; i < 61; i++) {
        char fc = filter_path[i];
        if (fc == '\0')
            break;
        if (p[i] != fc)
            return 0;
    }
    return 1;
}

// rel_path_matches_filter resolves a relative open against the process current
// working directory and applies the prefix filter to the resulting absolute
// path. The cwd is reconstructed by walking the dentry chain up to the global
// root, crossing mount points (jumping to the mountpoint dentry in the parent
// mount) like d_path does. The components are then streamed root->leaf, followed
// by "/" + relpath, and compared char by char against filter_path. Path
// components like "." and ".." are not normalised.
static __always_inline int rel_path_matches_filter(struct dentry *start_dentry, struct vfsmount *start_mnt, const char *relpath)
{
    u32 zero = 0;
    struct cwd_scratch *scratch = bpf_map_lookup_elem(&cwd_scratch_map, &zero);
    if (scratch == NULL)
        return 0;

    // Copy the filter out of .rodata into a runtime buffer so the verifier treats
    // its bytes as unknown scalars (see match_filter_char): this keeps the nested
    // cwd loops prunable instead of forking a state per known filter byte.
    bpf_probe_read_kernel(scratch->filter, sizeof(scratch->filter), (const void *)filter_path);

    int filter_len = 0;
    for (int i = 0; i < 61; i++) {
        if (scratch->filter[i] == '\0')
            break;
        filter_len = i + 1;
    }
    if (filter_len == 0)
        return 1; // empty filter matches everything

    // Collect the path component dentries leaf->root, crossing mount boundaries.
    struct dentry *dentry = start_dentry;
    struct vfsmount *vfsmnt = start_mnt;
    struct mount *mnt = container_of(vfsmnt, struct mount, mnt);
    int n = 0;
    int depth_truncated = 0;
    // Track whether dentry walk reached the filesystem root. If the loop
    // exhausts its iteration budget without reaching the root (e.g. more than
    // six nested mount crossings), reached_root stays 0 and we treat it as a
    // truncation: building a path from only the collected suffix would produce
    // a root-relative prefix that may falsely match the filter.
    int reached_root = 0;

    // A few extra iterations beyond CWD_MAX_DEPTH absorb mount crossings (which do
    // not add a component). Kept tight: more iterations explode verifier state.
    for (int i = 0; i < CWD_MAX_DEPTH + 6; i++) {
        struct dentry *mnt_root = BPF_CORE_READ(vfsmnt, mnt_root);
        struct dentry *parent = BPF_CORE_READ(dentry, d_parent);

        if (dentry == mnt_root) {
            struct mount *mnt_parent = BPF_CORE_READ(mnt, mnt_parent);
            if (mnt != mnt_parent) {
                // Cross into the parent mount at this mount's mountpoint.
                dentry = BPF_CORE_READ(mnt, mnt_mountpoint);
                mnt = mnt_parent;
                vfsmnt = &mnt->mnt;
                continue;
            }
            reached_root = 1;
            break; // reached the global root
        }
        if (dentry == parent) {
            reached_root = 1;
            break; // root without a mount crossing
        }

        if (n >= CWD_MAX_DEPTH) {
            // Path is deeper than CWD_MAX_DEPTH. The chain only holds the
            // deepest components; building a path from it would produce a
            // root-relative prefix that may match the filter even though
            // the real resolved path does not. Fail closed.
            depth_truncated = 1;
            break;
        }
        scratch->chain[n] = dentry;
        n++;
        dentry = parent;
    }

    if (depth_truncated || !reached_root)
        return 0; // cannot safely resolve; do not disrupt

    // Build the absolute path "<cwd>/<relpath>" into scratch->path. Each component
    // name is copied in a single bpf_probe_read_kernel_str call (not char by char):
    // the running offset is only ever used masked (pos & PATH_MASK), so the verifier
    // does not track it precisely and the loop does not explode into a state per
    // possible string length (which processed >1M insns and was rejected E2BIG).
    int pos = 0;

    // cwd components root->leaf (chain[0] is the leaf, chain[n-1] the topmost);
    // each contributes "/<name>".
    for (int k = 0; k < CWD_MAX_DEPTH; k++) {
        int idx = n - 1 - k;
        if (idx < 0)
            break;
        // Stop once we have more than 61 bytes — the prefix check only reads
        // scratch->path[0..60] (filter_path is at most 60 chars). Continuing
        // past this point risks pos wrapping past PATH_MASK(255) and overwriting
        // the bytes the comparison reads, producing false matches or misses.
        if (pos >= 62)
            break;
        idx &= (CWD_CHAIN_SIZE - 1); // mask to CWD_CHAIN_SIZE (power of 2) for verifier bounds proof

        scratch->path[pos & PATH_MASK] = '/';
        pos++;

        // Load the kernel dentry pointer from the map value with a plain access,
        // then apply CO-RE only to the kernel struct (cwd_scratch is not in vmlinux
        // BTF, so wrapping the whole chain in BPF_CORE_READ breaks relocation).
        struct dentry *comp = scratch->chain[idx];
        const char *dname = (const char *)BPF_CORE_READ(comp, d_name.name);
        int len = bpf_probe_read_kernel_str(&scratch->path[pos & PATH_MASK], CWD_NAME_BUF, dname);
        if (len > 1)
            pos += len - 1; // advance over the name, dropping the trailing NUL
    }

    // Append "/" + relpath — only if we still need more bytes for the comparison.
    if (pos < 62) {
        scratch->path[pos & PATH_MASK] = '/';
        pos++;
        int rlen = bpf_probe_read_kernel_str(&scratch->path[pos & PATH_MASK], 62, relpath);
        if (rlen > 1)
            pos += rlen - 1;
    }

    // Prefix match: the path matches when the whole filter is a prefix of it.
    for (int i = 0; i < 61; i++) {
        char fc = scratch->filter[i];
        if (fc == '\0')
            return 1; // entire filter matched: disrupt
        if (i >= pos)
            return 0; // path shorter than the filter
        if (scratch->path[i] != fc)
            return 0;
    }
    return 1;
}
#endif

#if defined(__TARGET_ARCH_arm64)
SEC("fmod_ret/__arm64_sys_openat")
#else
SEC("fmod_ret/__x64_sys_openat")
#endif
// fmod_ret programs receive the accumulated return value from earlier programs in
// the chain via `ret`. Pass-through branches must return `ret` (not 0) so a prior
// program's -exit_code is not cleared when multiple disk-failure programs run
// concurrently (e.g. one per spec.Paths entry or per target container).
int BPF_PROG(injection_disk_failure, struct pt_regs *real_regs, long ret)
{
    struct data_t data = {};

    // Get data of the current process
    u32 pid = bpf_get_current_pid_tgid();
    if (pid == exclude_pid) {
        return ret;
    }
    u32 tid = bpf_get_current_pid_tgid() >> 32;
    u32 gid = bpf_get_current_uid_gid();

    if (target_pid_ns_inum != 0) {
        struct task_struct *task = (struct task_struct *)bpf_get_current_task();
        // Use the task's active PID namespace (thread_pid->numbers[level].ns)
        // rather than nsproxy->pid_ns_for_children. pid_ns_for_children is
        // updated by unshare(CLONE_NEWPID)/setns before the process forks, so
        // a host process preparing to launch a container child would be matched,
        // while a container process that called unshare would be missed.
        // thread_pid->numbers[level].ns is the namespace the task's PID is
        // actually registered in — the correct active namespace.
        struct pid *thread_pid = BPF_CORE_READ(task, thread_pid);
        unsigned int level = BPF_CORE_READ(thread_pid, level);
        struct pid_namespace *active_ns = NULL;
        bpf_probe_read_kernel(&active_ns, sizeof(active_ns),
                              &thread_pid->numbers[level].ns);
        unsigned int ns_inum = 0;
        if (active_ns)
            ns_inum = BPF_CORE_READ(active_ns, ns.inum);
        if (ns_inum != target_pid_ns_inum)
            return ret;
    }

    if (tid == exclude_pid) {
        return ret;
    }

// Exclude this part of code if the following variables are not defined.
// It allows the go program to compile without error.
#if defined(__TARGET_ARCH_arm64) || defined(__TARGET_ARCH_x86)
    int dirfd = (int)PT_REGS_PARM1_CORE(real_regs);
    char *path = (char *)PT_REGS_PARM2_CORE(real_regs);
    char cmp_path_name[62];
    bpf_probe_read(&cmp_path_name, sizeof(cmp_path_name), path);

    if (cmp_path_name[0] == '/') {
        // Absolute path: apply the prefix filter directly.
        if (!abs_path_matches_filter(cmp_path_name))
            return ret;
    } else {
        // When the filter is "/" it matches everything — any relative open
        // (regardless of dirfd) resolves under root. Check this first so that
        // dirfd-relative opens are not passed through before reaching the root
        // shortcut. For non-root filters, only AT_FDCWD-relative opens can be
        // resolved to an absolute path in BPF.
        int filter_is_root = (filter_path[0] == '/' && filter_path[1] == '\0');

        if (!filter_is_root) {
            if (dirfd != AT_FDCWD)
                return ret;

            // If the path was truncated at 62 bytes (no NUL in the buffer), a
            // ".." component may appear past the cutoff. Reject conservatively:
            // we cannot guarantee the visible prefix is free of ".." escapes.
            int relpath_truncated = 1;
            for (int ti = 0; ti < 62; ti++) {
                if (cmp_path_name[ti] == '\0') {
                    relpath_truncated = 0;
                    break;
                }
            }
            if (relpath_truncated)
                return ret;

            // Reject relative paths that contain ".." components. BPF cannot
            // normalize them: openat(AT_FDCWD, "../outside") from a cwd under
            // the filter prefix would produce "/filtered/../outside", which
            // passes the prefix check but resolves outside the filter. Pass
            // through conservatively rather than disrupting the wrong path.
            if (cmp_path_name[0] == '.' && cmp_path_name[1] == '.' &&
                (cmp_path_name[2] == '\0' || cmp_path_name[2] == '/'))
                return ret;
            for (int di = 1; di < 60; di++) {
                if (cmp_path_name[di] == '\0')
                    break;
                if (cmp_path_name[di - 1] == '/' && cmp_path_name[di] == '.' &&
                    cmp_path_name[di + 1] == '.' &&
                    (cmp_path_name[di + 2] == '\0' || cmp_path_name[di + 2] == '/'))
                    return ret;
            }

            struct task_struct *cwd_task = (struct task_struct *)bpf_get_current_task();
            struct dentry *cwd_dentry = BPF_CORE_READ(cwd_task, fs, pwd.dentry);
            struct vfsmount *cwd_mnt = BPF_CORE_READ(cwd_task, fs, pwd.mnt);
            if (cwd_dentry == NULL || cwd_mnt == NULL)
                return ret;

            if (!rel_path_matches_filter(cwd_dentry, cwd_mnt, cmp_path_name))
                return ret;
        }
        // filter_is_root: fall through to disrupt
    }
#endif

    if (probability != 100) {
        if (hits != 0) {
            unsigned long long scaled_disruptedHits = disruptedHits * 100;
            unsigned long long scaled_hits = hits;

            if ((scaled_disruptedHits / scaled_hits) > probability) {
                hits++;
                return ret;
            }
        }

        hits++;
        disruptedHits++;
    }

    data.pid = pid;
    data.tid = tid;
    data.id = gid;

    // Get command name
    bpf_get_current_comm(&data.comm, sizeof(data.comm));

    // Add the event to the ring buffer
    bpf_perf_event_output(ctx, &events, BPF_F_CURRENT_CPU, &data, 100);

    return -(int)exit_code;
}

