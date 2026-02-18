# Memory pressure

The `memoryPressure` field generates memory load on the targeted pod by gradually allocating anonymous memory pages until a target utilization percentage is reached.

## How it works

Containers achieve resource limitation (cpu, disk, memory) through cgroups. The memory cgroup controller tracks and limits memory usage of processes within a cgroup. Docker and containerd containers get their own memory cgroup with limits derived from Kubernetes resource requests and limits.

> :open_book: More information on how memory cgroups work: [cgroup v1](https://www.kernel.org/doc/Documentation/cgroup-v1/memory.txt) | [cgroup v2](https://docs.kernel.org/admin-guide/cgroup-v2.html#memory-interface-files)

The `/sys/fs/cgroup` directory of the host must be mounted in the injector pod at the `/mnt/cgroup` path for it to work.

### Injection process

When the injector pod starts:

- It creates a dedicated process for each container in the targeted pod (same architecture as CPU pressure):
  - The standard injector process (`/usr/local/bin/chaos-injector memory-pressure`) acts as an orchestrator
  - It spawns one child process (`/usr/local/bin/chaos-injector memory-stress`) per container
  - When a container restarts, the orchestrator can re-create the stress process
- Each `memory-stress` child process is responsible for stressing memory of a SPECIFIC container:
  1. It joins the target container's cgroup (`cgroup.procs`) so allocated memory counts against the container's memory limit
  2. It reads the memory limit from the cgroup:
     - cgroup v1: `memory.limit_in_bytes`
     - cgroup v2: `memory.max`
  3. It reads the current memory usage:
     - cgroup v1: `memory.usage_in_bytes`
     - cgroup v2: `memory.current`
  4. It calculates the target allocation: `(limit * targetPercent / 100) - currentUsage`
  5. If `rampDuration` is set, it divides the allocation into incremental steps (1-second intervals) spread over the ramp period
  6. It allocates memory using `mmap(MAP_ANONYMOUS | MAP_PRIVATE | MAP_POPULATE)`:
     - `MAP_ANONYMOUS`: allocates memory not backed by any file
     - `MAP_PRIVATE`: pages are not shared with other processes
     - `MAP_POPULATE`: forces the kernel to allocate physical pages immediately (no lazy allocation), ensuring the RSS increase is deterministic
  7. On cleanup, all allocations are released via `munmap()`

### Why mmap instead of malloc?

Using `mmap` with `MAP_POPULATE` provides several advantages over `malloc` or Go's built-in memory allocator:

- **Deterministic RSS**: `MAP_POPULATE` forces immediate physical page allocation, so the memory usage increase is visible immediately in cgroup accounting
- **No GC interference**: Go's garbage collector cannot reclaim `mmap`-allocated pages since they are outside the Go heap
- **Clean deallocation**: `munmap` immediately releases pages back to the kernel, unlike `free()` which may retain pages in the process's free list
- **No fragmentation**: each allocation is a contiguous virtual memory region

### Ramp duration

When `rampDuration` is specified, memory is consumed gradually rather than all at once. This is important because:

- It simulates realistic memory leak scenarios where memory usage increases over time
- It gives auto-scaling and resource reclamation systems time to detect and react
- It allows operators to observe the progression and abort if needed

The ramp divides the total allocation into equal-sized chunks allocated at 1-second intervals. For example, with `targetPercent: "80%"` and `rampDuration: 10m`, the injector allocates 1/600th of the target amount every second over 10 minutes.

## Spec fields

| Field           | Type   | Default | Description                                                                        |
| --------------- | ------ | ------- | ---------------------------------------------------------------------------------- |
| `targetPercent` | string | —       | Target memory utilization percentage of the cgroup limit (e.g. `"76%"`, required)  |
| `rampDuration`  | string | `0`     | Duration over which memory is gradually consumed (e.g. `"10m"`, Go duration format) |

### targetPercent

A string representing the target memory utilization as a percentage of the container's cgroup memory limit. Can be specified with or without a `%` suffix (e.g. `"76%"` or `"76"`). Must be between 1 and 100.

The injector calculates the actual bytes to allocate as: `(limit * targetPercent / 100) - currentUsage`. If current usage already exceeds the target, no additional memory is allocated.

### rampDuration

A Go duration string (e.g. `"10m"`, `"30s"`, `"1h5m"`) specifying how long the ramp-up period should last. If omitted or `0`, all memory is allocated immediately in a single step.

## Compatibility

| Feature             | Compatible |
| ------------------- | ---------- |
| `level: pod`        | Yes        |
| `level: node`       | Yes        |
| `containers` scoping | No — applies to all containers |
| `onInit`            | No         |
| `pulse`             | Yes        |
| `dryRun`            | Yes        |
| Container Failure   | No (exclusive) |
| Node Failure        | No (exclusive) |
| Pod Replacement     | No (exclusive) |
| Network             | Yes        |
| DNS                 | Yes        |
| gRPC                | Yes        |
| CPU Pressure        | Yes        |
| Disk Pressure       | Yes        |
| Disk Failure        | Yes        |

## Examples

### Gradual memory pressure (realistic memory leak)

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: memory-pressure-gradual
  namespace: chaos-demo
spec:
  level: pod
  selector:
    service: demo-app
  count: 1
  duration: 30m
  memoryPressure:
    targetPercent: "76%"
    rampDuration: 10m
```

### Immediate memory pressure (sudden allocation spike)

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: memory-pressure-immediate
  namespace: chaos-demo
spec:
  level: pod
  selector:
    service: demo-app
  count: 1
  duration: 15m
  memoryPressure:
    targetPercent: "50%"
```

### Combined with CPU pressure (resource exhaustion)

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: resource-exhaustion
  namespace: chaos-demo
spec:
  level: pod
  selector:
    service: demo-app
  count: 1
  duration: 15m
  cpuPressure:
    count: "50%"
  memoryPressure:
    targetPercent: "60%"
    rampDuration: 5m
```

### Pulsing memory pressure

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: memory-pressure-pulse
  namespace: chaos-demo
spec:
  level: pod
  selector:
    service: demo-app
  count: 1
  duration: 30m
  pulse:
    activeDuration: 2m
    dormantDuration: 1m
  memoryPressure:
    targetPercent: "80%"
    rampDuration: 30s
```

### Node-level memory pressure

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: memory-pressure-node
  namespace: chaos-demo
spec:
  level: node
  selector:
    node.kubernetes.io/instance-type: k3s
  count: 1
  duration: 10m
  memoryPressure:
    targetPercent: "70%"
    rampDuration: 5m
```

## Manually confirming memory pressure

In a memory disruption, the injector process is moved to the target memory cgroup but the injector container keeps its own PID namespace. Tools running inside the target container won't see the injector process, but the memory usage increase will be visible in cgroup accounting.

### Checking from the target container

```sh
# Check cgroup memory usage (v1)
cat /sys/fs/cgroup/memory/memory.usage_in_bytes

# Check cgroup memory usage (v2)
cat /sys/fs/cgroup/memory.current
```

### Checking from the node

```sh
# Find the target container's cgroup path
crictl inspect <container-id> | grep cgroupsPath

# Check memory usage (v1)
cat /sys/fs/cgroup/memory/<cgroup-path>/memory.usage_in_bytes

# Check memory limit (v1)
cat /sys/fs/cgroup/memory/<cgroup-path>/memory.limit_in_bytes

# Check memory usage (v2)
cat /sys/fs/cgroup/<cgroup-path>/memory.current

# Check memory limit (v2)
cat /sys/fs/cgroup/<cgroup-path>/memory.max
```

### Checking via Kubernetes metrics

```sh
kubectl top pod <pod-name> -n <namespace>
```

The memory usage reported should increase according to the configured `targetPercent` and `rampDuration`.

## Manual cleanup instructions

:information_source: All those commands must be executed on the infected host (except for `kubectl`).

Under normal circumstances, killing the injector process is sufficient to release all allocated memory — `munmap` is called during cleanup and, even if the process is killed abruptly, the kernel automatically reclaims all `mmap`-allocated pages when the process exits.

- Identify the injector process PIDs

```sh
# ps ax | grep chaos-injector | grep memory
1113879 ?        Ssl    0:00 /usr/local/bin/chaos-injector memory-pressure --target-percent 76 --ramp-duration 10m ...
1113902 ?        Sl     0:00 /usr/local/bin/chaos-injector memory-stress --target-percent=76 --ramp-duration=10m0s ...
```

- Kill the orchestrator process (it will also stop the child stress process)

```sh
# kill 1113879
```

_You can SIGKILL the injector process if it is stuck but a standard kill is recommended._

- Ensure the injector processes are gone

```sh
# ps ax | grep chaos-injector | grep memory
1119071 pts/0    S+     0:00 grep chaos-injector
```

- Verify that memory usage has returned to normal

```sh
# Check cgroup memory usage has decreased (v1)
cat /sys/fs/cgroup/memory/<cgroup-path>/memory.usage_in_bytes

# Check cgroup memory usage has decreased (v2)
cat /sys/fs/cgroup/<cgroup-path>/memory.current
```

Since the injector uses `mmap` for allocation, all memory is automatically returned to the kernel when the process exits. There is no persistent state to clean up (unlike disk pressure which modifies cgroup throttle files).
