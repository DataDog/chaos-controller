# Disk full (ENOSPC)

The `diskFull` field offers a way to genuinely fill a target pod volume, causing real ENOSPC errors on all subsequent write operations. Unlike disk pressure (which throttles I/O) or disk failure (which intercepts `openat` syscalls), this disruption makes the filesystem actually run out of space — visible to `df`, `statfs()`, Kubernetes eviction, and monitoring systems.

## How it works

The injector creates a **ballast file** (`.chaos-diskfull-{disruption-name}`) at the target path using the `fallocate(2)` syscall, which is instant (O(1), metadata-only allocation on ext4/xfs). On filesystems that don't support `fallocate`, it falls back to writing zeros.

When the disruption is cleaned up, the ballast file is removed and space is freed immediately.

## Spec fields

| Field       | Type   | Required | Description |
|-------------|--------|----------|-------------|
| `path`      | string | Yes      | Mount path inside the target pod to fill (e.g., `/data`, `/var/log`) |
| `capacity`  | string | One of   | Fill to this percentage of total volume capacity (e.g., `"95%"`) |
| `remaining` | string | One of   | Leave only this much free space on the volume (e.g., `"50Mi"`, `"1Gi"`) |
| `writeSyscall` | object | No    | Optional eBPF-based write syscall interception (see below) |

`capacity` and `remaining` are **mutually exclusive** — exactly one must be set.

### writeSyscall (optional)

When set, an eBPF program is launched alongside the volume fill to intercept `write` syscalls and return errors with configurable probability. This is useful for testing partial write failures or for environments where the volume fill alone isn't sufficient.

| Field         | Type   | Default   | Description |
|---------------|--------|-----------|-------------|
| `exitCode`    | string | `ENOSPC`  | errno to return: `ENOSPC`, `EDQUOT`, `EIO`, `EROFS`, `EFBIG`, `EPERM`, `EACCES` |
| `probability` | string | `"100%"`  | Percentage of write syscalls to fail (1-100%) |

**Requirements:** The kernel must support eBPF with `CONFIG_BPF_KPROBE_OVERRIDE` enabled.

## Examples

### Fill to 95% capacity

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: disk-full-test
  namespace: my-app
spec:
  level: pod
  selector:
    app: my-service
  count: 1
  duration: 10m
  diskFull:
    path: "/data"
    capacity: "95%"
```

### Leave only 10Mi free

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: disk-full-remaining
  namespace: my-app
spec:
  level: pod
  selector:
    app: my-service
  count: 1
  duration: 5m
  diskFull:
    path: "/var/log"
    remaining: "10Mi"
```

### Volume fill + eBPF write interception

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: disk-full-with-ebpf
  namespace: my-app
spec:
  level: pod
  selector:
    app: my-service
  count: 1
  duration: 10m
  diskFull:
    path: "/data"
    capacity: "90%"
    writeSyscall:
      exitCode: ENOSPC
      probability: "50%"
```

### Fill to 100% (requires unsafeMode)

By default, the controller enforces a 1Mi minimum free space safety floor to prevent filesystem journal corruption. To fill completely:

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: disk-full-complete
  namespace: my-app
spec:
  level: pod
  selector:
    app: my-service
  count: 1
  duration: 5m
  unsafeMode:
    allowDiskFullNoFloor: true
  diskFull:
    path: "/data"
    capacity: "100%"
```

## Safety

### Minimum free space floor

A 1Mi safety floor is enforced by default. This prevents:
- Filesystem journal corruption on ext4
- Inability to perform cleanup operations
- Cascade failures from completely exhausted filesystems

Override with `unsafeMode.allowDiskFullNoFloor: true`.

### Ephemeral storage eviction

If the target pod has `resources.limits.ephemeral-storage` set and the target volume is ephemeral (`emptyDir`), filling the volume may cause the kubelet to evict the pod. This is **realistic behavior** — it's exactly what would happen if the application itself filled the disk.

The controller emits a warning Kubernetes Event when this condition is detected, but does not block the disruption.

### Level restriction

Disk full disruptions are **pod-level only**. Node-level disk fill is not supported because it can crash the kubelet and affect all pods on the node.

## Manual cleanup instructions

If the chaos pod crashes before cleanup and the finalizer fails:

1. Identify the ballast file on the target node:

```shell
find /var/lib/kubelet/pods/ -name ".chaos-diskfull-*" -type f
```

2. Remove it:

```shell
rm /path/to/.chaos-diskfull-<disruption-name>
```

Space is freed immediately upon file removal.

## Comparison with other disk disruptions

| Disruption | Mechanism | ENOSPC on writes? | Visible to `df`/monitoring? | Affects open FDs? |
|---|---|---|---|---|
| **Disk Pressure** | Cgroup blkio throttling | No (slows I/O only) | No | N/A |
| **Disk Failure** | eBPF on `openat` | Only on file open | No | No |
| **Disk Full** | Real space allocation | Yes (all syscalls) | Yes | Yes |
