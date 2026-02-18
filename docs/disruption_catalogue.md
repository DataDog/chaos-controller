# Chaos Controller — Disruption Catalogue

> Reference documentation for all fault injection capabilities provided by [Datadog Chaos Controller](https://github.com/DataDog/chaos-controller).
> This catalogue is designed for experiment planning and can be consumed by humans and AI agents alike.

---

## Quick Reference

| Disruption                                | What It Does                                       | Level     | Reversible | Can Combine    |
| ----------------------------------------- | -------------------------------------------------- | --------- | ---------- | -------------- |
| [Network](#1-network-disruption)          | Packet loss, delay, corruption, bandwidth limiting | Pod, Node | Yes        | Yes            |
| [DNS](#2-dns-disruption)                  | Fake DNS responses, NXDOMAIN, drops, SERVFAIL      | Pod       | Yes        | Yes            |
| [gRPC](#3-grpc-disruption)                | Return gRPC errors or override responses           | Pod       | Yes        | Yes            |
| [CPU Pressure](#4-cpu-pressure)            | Consume CPU cycles in target cgroup                | Pod, Node | Yes        | Yes            |
| [Memory Pressure](#5-memory-pressure)      | Gradually consume memory in target cgroup          | Pod, Node | Yes        | Yes            |
| [Disk Pressure](#6-disk-pressure)          | Throttle read/write I/O throughput                 | Pod, Node | Yes        | Yes            |
| [Disk Failure](#7-disk-failure)            | Fail file open syscalls via eBPF                   | Pod, Node | Yes*       | Yes            |
| [Container Failure](#8-container-failure)  | Kill container processes (SIGTERM/SIGKILL)         | Pod       | No         | No (exclusive) |
| [Node Failure](#9-node-failure)            | Kernel panic or power-off a node                   | Node      | No         | No (exclusive) |
| [Pod Replacement](#10-pod-replacement)     | Cordon node, delete pod and optionally PVCs        | Pod       | No         | No (exclusive) |

\* Disk Failure injection is removed when the injector process exits.

**Combination rule:** Network, DNS, gRPC, CPU Pressure, Memory Pressure, Disk Pressure, and Disk Failure can all be applied together in a single Disruption resource. Container Failure, Node Failure, and Pod Replacement are mutually exclusive with every other disruption type.

---

## Common Structure

Every disruption is a Kubernetes custom resource of kind `Disruption` (API group `chaos.datadoghq.com/v1beta1`). The common envelope looks like this:

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: <disruption-name>
  namespace: <target-namespace>
spec:
  level: pod          # "pod" or "node"
  selector:           # label key-value pairs to match targets
    <label>: <value>
  count: 1            # integer or percentage (e.g. "50%")
  duration: 1h        # auto-cleanup after this duration
  # ... disruption-specific fields below
```

See [Targeting](#targeting) and [Advanced Features](#advanced-features) for the full set of cross-cutting options.

---

## 1. Network Disruption

Applies Linux traffic control (`tc`) rules inside the target's network namespace to manipulate packets. Supports scoping traffic by host, Kubernetes service, cloud provider IP range, and HTTP method/path.

### Fault Modes

| Field            | Type | Range   | Description                                  |
| ---------------- | ---- | ------- | -------------------------------------------- |
| `drop`           | int  | 0–100   | Percentage of packets to drop                |
| `delay`          | uint | 0–60000 | Latency added to packets (milliseconds)      |
| `delayJitter`    | uint | 0–100   | Jitter as percentage of delay (default: 10%) |
| `corrupt`        | int  | 0–100   | Percentage of packets to corrupt             |
| `duplicate`      | int  | 0–100   | Percentage of packets to duplicate           |
| `bandwidthLimit` | int  | ≥ 32    | Throughput cap in bytes/sec                  |

At least one fault mode is required. Multiple can be combined in a single disruption.

### Traffic Scoping

Traffic scoping fields are all optional and combinable. When omitted, the disruption applies to all traffic.

#### Hosts

Target by IP address, CIDR block, or hostname.

| Field         | Type   | Description                                             |
| ------------- | ------ | ------------------------------------------------------- |
| `host`        | string | IP, CIDR, or hostname                                   |
| `port`        | int    | Port number (0–65535)                                   |
| `protocol`    | string | `tcp`, `udp`, or omit for both                          |
| `flow`        | string | `egress` (default), `ingress`                           |
| `connState`   | string | `new`, `est`, or omit for all                           |
| `dnsResolver` | string | `pod`, `node`, `pod-fallback-node`, `node-fallback-pod` |
| `percentage`  | int    | 1–100, percentage of resolved IPs to disrupt            |

#### Allowed Hosts

Exclude specific hosts from disruption. Same fields as `hosts`.

#### Kubernetes Services

Target ClusterIP services in the same cluster.

| Field        | Type   | Description                                       |
| ------------ | ------ | ------------------------------------------------- |
| `name`       | string | Service name (required)                           |
| `namespace`  | string | Service namespace (required)                      |
| `ports`      | list   | Optional port filter (by `name` and/or `port`)    |
| `percentage` | int    | 1–100, percentage of service endpoints to disrupt |

#### Cloud Provider Services

Target IP ranges of cloud provider services.

```yaml
cloud:
  aws:
    - service: "<service-name>"     # e.g. "S3", "DYNAMODB"
  gcp:
    - service: "<service-name>"     # e.g. "Google"
  datadog:
    - service: "<service-name>"     # e.g. "synthetics"
```

Each entry supports optional `protocol`, `flow`, and `connState` fields.

#### HTTP Filters

Filter by HTTP method and/or request path (uses eBPF).

| Field     | Type | Constraints                                                                  |
| --------- | ---- | ---------------------------------------------------------------------------- |
| `methods` | list | Up to 9 values: GET, POST, PUT, DELETE, PATCH, HEAD, CONNECT, OPTIONS, TRACE |
| `paths`   | list | Up to 20 paths, each ≤ 90 chars, must start with `/`                         |

### Constraints and Limitations

- Maximum 2048 tc filters per disruption. Complex host/service/cloud configurations can exhaust this limit.
- SSH (port 22), node IP, default gateway, cloud metadata (169.254.169.254), and ARP are automatically excluded. Disable with `disableDefaultAllowedHosts: true`.
- Ingress flow only works for TCP and requires at least a port or host.
- `bandwidthLimit` must be ≥ 32 bytes/sec when set.
- All packets experience a small additional latency due to `prio` qdisc processing even if no delay is configured.
- DNS resolution for hostname-based hosts is repeated on an interval, which can add latency to filter setup.
- Percentage-based host selection uses consistent hashing (stable across pod restarts but not configurable).
- Network interfaces must have a tx queue length > 0 (the injector temporarily sets it to 1000 if needed).
- HTTP filtering requires eBPF and adds an extra layer to the tc tree, increasing processing overhead.
- When using `services`, only ClusterIP-type services are supported. Services from other clusters cannot be resolved.

### Examples

**Drop 100% of outgoing packets:**

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: network-drop
  namespace: chaos-demo
spec:
  level: pod
  selector:
    service: demo-curl
  count: 1
  network:
    drop: 100
```

**Add 1 second latency with 5% jitter:**

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: network-delay
  namespace: chaos-demo
spec:
  level: pod
  selector:
    service: demo-curl
  count: 1
  network:
    delay: 1000
    delayJitter: 5
```

**Limit bandwidth to 1 KB/s:**

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: network-bandwidth-limitation
  namespace: chaos-demo
spec:
  level: pod
  selector:
    service: demo-curl
  count: 1
  network:
    bandwidthLimit: 1024
```

**Corrupt 100% of outgoing packets:**

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: network-corrupt
  namespace: chaos-demo
spec:
  level: pod
  selector:
    service: demo-curl
  count: 1
  network:
    corrupt: 100
```

**Drop all ingress traffic on port 80:**

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: network-ingress
  namespace: chaos-demo
spec:
  level: pod
  selector:
    service: demo-nginx
  count: 1
  network:
    drop: 100
    hosts:
      - port: 80
        flow: ingress
```

**Drop packets to a Kubernetes service:**

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: network-filter-service
  namespace: chaos-demo
spec:
  level: pod
  selector:
    service: demo-curl
  count: 1
  network:
    drop: 100
    services:
      - name: demo
        namespace: chaos-demo
        ports:
          - name: regular-port
            port: 8080
          - port: 8081
```

**Drop packets to cloud provider services (AWS S3, GCP, Datadog):**

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: network-cloud
  namespace: chaos-demo
spec:
  level: pod
  selector:
    service: demo-curl
  count: 100%
  network:
    drop: 100
    cloud:
      aws:
        - service: "S3"
          protocol: tcp
          flow: ingress
          connState: new
      gcp:
        - service: "Google"
      datadog:
        - service: "synthetics"
```

**Drop only HTTP GET requests to `/`:**

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: network-http-filter
  namespace: chaos-demo
spec:
  level: pod
  selector:
    service: demo-curl
  count: 1
  network:
    drop: 100
    http:
      methods:
        - GET
      paths:
        - /
```

**Disrupt 50% of resolved IPs for a hostname:**

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: network-dns-resolver-control
  namespace: chaos-demo
spec:
  level: pod
  selector:
    service: demo-curl
  count: 100%
  network:
    drop: 100
    hosts:
      - host: www.example.com
        port: 443
        protocol: tcp
        percentage: 50
```

---

## 2. DNS Disruption

Redirects DNS traffic via iptables to a custom DNS responder that serves fake records, errors, or drops queries. Non-disrupted hostnames are forwarded transparently to upstream DNS.

### Configuration

| Field      | Type   | Default    | Description                      |
| ---------- | ------ | ---------- | -------------------------------- |
| `port`     | int    | 53         | DNS port to intercept (1–65535)  |
| `protocol` | string | `both`     | `udp`, `tcp`, or `both`          |
| `records`  | list   | (required) | At least one DNS record override |

#### Record Configuration

| Field          | Type   | Description                                        |
| -------------- | ------ | -------------------------------------------------- |
| `hostname`     | string | Domain to intercept (subdomain matching supported) |
| `record.type`  | string | `A`, `AAAA`, `CNAME`, `MX`, `TXT`, or `SRV`        |
| `record.value` | string | See value formats below                            |
| `record.ttl`   | uint32 | TTL in seconds (default: 0)                        |

#### Value Formats by Record Type

| Type     | Value Format                                            | Example                                       |
| -------- | ------------------------------------------------------- | --------------------------------------------- |
| A / AAAA | Comma-separated IPs (round-robin)                       | `"192.168.1.10,192.168.1.11"`                 |
| A / AAAA | Failure mode keyword                                    | `NXDOMAIN`, `DROP`, `SERVFAIL`, `RANDOM`      |
| CNAME    | Target hostname                                         | `"www.google.com"`                            |
| MX       | `"priority hostname"` pairs, comma-separated            | `"10 mail1.example.com,20 mail2.example.com"` |
| TXT      | Arbitrary text                                          | `"verification-token-12345"`                  |
| SRV      | `"priority weight port target"` tuples, comma-separated | `"10 60 8080 server1.example.com"`            |

#### Failure Modes

| Keyword    | Effect                                   | Typical Use Case                         |
| ---------- | ---------------------------------------- | ---------------------------------------- |
| `NXDOMAIN` | Immediate "domain not found"             | Test missing-dependency handling         |
| `SERVFAIL` | DNS server failure response              | Test DNS infrastructure failures         |
| `DROP`     | Silent query drop (causes timeout)       | Test timeout/retry behavior              |
| `RANDOM`   | Returns random IPs (RFC 5737 TEST-NET-1) | Test connection to unreachable addresses |

### Constraints and Limitations

- Pod-level only (not compatible with `level: node`).
- No duplicate `hostname` + `type` combinations in the same disruption.
- At least one record is required.
- The DNS responder is global per injector pod — it cannot filter DNS queries by process. All DNS traffic from the pod is intercepted.
- Uses non-standard ports (5353/5354) internally to avoid conflicts with existing DNS services.
- The `RedirectTo` iptables rule must be set before `Intercept` internally; this is handled automatically but means partial injection states are possible during setup.
- Subdomain matching means disrupting `example.com` also matches `api.example.com`. This can cause wider impact than intended.
- The custom DNS responder must be able to reach upstream DNS servers to forward non-disrupted queries. Network disruptions applied simultaneously may interfere.

### Examples

**Return SERVFAIL and random IPs for different hosts:**

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: dns-failure-modes
  namespace: chaos-demo
spec:
  level: pod
  selector:
    service: demo-curl
  count: 1
  duration: 15m
  dns:
    protocol: both
    records:
      - hostname: www.example.com
        record:
          type: A
          value: SERVFAIL
      - hostname: demo.chaos-demo.svc.cluster.local
        record:
          type: A
          value: RANDOM
          ttl: 300
```

**Redirect a hostname to specific IPs (round-robin):**

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: dns-mixed-records
  namespace: chaos-demo
spec:
  level: pod
  selector:
    service: demo-curl
  count: 1
  duration: 15m
  dns:
    port: 53
    protocol: both
    records:
      - hostname: www.example.com
        record:
          type: A
          value: "192.168.1.10,192.168.1.11,192.168.1.12"
          ttl: 60
```

---

## 3. gRPC Disruption

Sends disruption configuration over gRPC to a chaos interceptor running inside the target application. The interceptor then returns specified error codes or overrides responses for matching endpoints.

**Prerequisite:** The target application must integrate the chaos interceptor library and register it with its gRPC server. See [gRPC disruption instructions](grpc_disruption/instructions.md).

### Configuration

| Field       | Type | Description                          |
| ----------- | ---- | ------------------------------------ |
| `port`      | int  | gRPC server port (1–65535, required) |
| `endpoints` | list | Endpoint alterations (required)      |

#### Endpoint Alteration

| Field          | Type   | Description                                                    |
| -------------- | ------ | -------------------------------------------------------------- |
| `endpoint`     | string | gRPC endpoint path, e.g. `/package.Service/Method` (required)  |
| `error`        | string | gRPC error code to return (mutually exclusive with `override`) |
| `override`     | string | Response payload to return (mutually exclusive with `error`)   |
| `queryPercent` | int    | 0–100, percentage of queries to affect                         |

#### Available gRPC Error Codes

`OK`, `CANCELED`, `UNKNOWN`, `INVALID_ARGUMENT`, `DEADLINE_EXCEEDED`, `NOT_FOUND`, `ALREADY_EXISTS`, `PERMISSION_DENIED`, `RESOURCE_EXHAUSTED`, `FAILED_PRECONDITION`, `ABORTED`, `OUT_OF_RANGE`, `UNIMPLEMENTED`, `INTERNAL`, `UNAVAILABLE`, `DATA_LOSS`, `UNAUTHENTICATED`

### Constraints and Limitations

- Exactly one of `error` or `override` per alteration.
- Sum of `queryPercent` for the same endpoint must not exceed 100%.
- Each alteration must have at least 1% chance of occurring.
- Pod-level only.
- **Requires application changes:** The target gRPC server must integrate the `DisruptionListener` chaos interceptor library. Without it, injection returns `codes.Unimplemented` and silently fails.
- Uses an insecure gRPC connection to the target (no TLS).
- 5-second connection timeout — targets that are slow to respond may fail injection.
- No streaming support. Only unary gRPC calls can be disrupted.
- The `override` field currently only supports returning `emptypb.Empty` (`"{}"`).
- Not re-injectable: cannot be reapplied to the same target without a full cleanup cycle.

### Example

**Return errors and overrides on gRPC endpoints:**

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: grpc
  namespace: chaos-demo
spec:
  level: pod
  selector:
    service: chaos-dogfood-server
  count: 100%
  grpc:
    port: 50050
    endpoints:
      - endpoint: /chaosdogfood.ChaosDogfood/getCatalog
        error: NOT_FOUND
        queryPercent: 25
      - endpoint: /chaosdogfood.ChaosDogfood/getCatalog
        error: ALREADY_EXISTS
        queryPercent: 50
      - endpoint: /chaosdogfood.ChaosDogfood/getCatalog
        override: "{}"
      - endpoint: /chaosdogfood.ChaosDogfood/order
        override: "{}"
        queryPercent: 50
```

---

## 4. CPU Pressure

Joins the target container's cgroup and burns CPU using pinned goroutines (one per core). Each goroutine runs a tight busy-loop with an on/off duty cycle proportional to the requested pressure.

### Configuration

| Field   | Type          | Default | Description                                                               |
| ------- | ------------- | ------- | ------------------------------------------------------------------------- |
| `count` | int or string | `100%`  | Number of cores (integer) or percentage of allocated cores (e.g. `"50%"`) |

When `count` is an integer, the injector stresses all allocated cores at `(count / allocated_cores) * 100%` intensity. When it is a percentage, that percentage is used directly.

### Constraints and Limitations

- Not compatible with `onInit` mode.
- The injector process joins the target's cgroup and sets highest priority (nice -20), which means it competes with the target application for CPU. This consumes the injector pod's own CPU quota as well.
- Cannot inject if stress goroutines are already running on the target (prevents double-injection; requires cleanup first).
- Goroutines are pinned to specific CPU cores via `sched_setaffinity`. If the target's cpuset changes during injection, the stress distribution may become uneven.
- Maximum CPU limit is 8192 cores.
- At node level, the pressure affects all workloads on the node, not just the selected target.

### Examples

**Stress 100% of all allocated cores:**

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: cpu-pressure
  namespace: chaos-demo
spec:
  duration: 5m
  selector:
    service: demo-curl
  count: 1
  cpuPressure: {}
```

**Stress 1 core's worth of CPU:**

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: cpu-pressure-count
  namespace: chaos-demo
spec:
  level: pod
  selector:
    service: demo-curl
  count: 1
  cpuPressure:
    count: 1
```

**Node-level CPU pressure:**

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: cpu-level-node
  namespace: chaos-demo
spec:
  level: node
  selector:
    node.kubernetes.io/instance-type: k3s
  count: 1
  cpuPressure: {}
```

---

## 5. Memory Pressure

Joins the target container's cgroup and gradually allocates anonymous memory pages using `mmap(MAP_ANONYMOUS|MAP_POPULATE)` until the target memory utilization percentage is reached. This simulates gradual memory buildup that can trigger auto-shrink, auto-scaling, and resource reclamation logic — unlike Container Failure which only simulates an OOM kill.

### Configuration

| Field           | Type   | Default | Description                                                                    |
| --------------- | ------ | ------- | ------------------------------------------------------------------------------ |
| `targetPercent` | string | —       | Target memory utilization as a percentage of cgroup limit (e.g. `"76%"`, required) |
| `rampDuration`  | string | `0`     | Duration over which memory is gradually consumed (e.g. `"10m"`)                |

When `rampDuration` is omitted or `0`, memory is consumed immediately in a single allocation. When set, memory is allocated in incremental steps over the ramp period (1-second intervals).

### How It Works

1. The injector spawns a child process (`memory-stress`) inside the target's cgroup.
2. The child reads the cgroup memory limit (`memory.limit_in_bytes` on v1, `memory.max` on v2) and current usage.
3. It calculates the target bytes: `(limit * targetPercent / 100) - currentUsage`.
4. Memory is allocated using `mmap` with `MAP_POPULATE` to ensure physical pages are committed immediately (no lazy allocation).
5. On cleanup, all allocations are released via `munmap`.

### Constraints and Limitations

- Not compatible with `onInit` mode, Container Failure, Node Failure, or Pod Replacement.
- Cannot target specific containers (applies to the pod's memory cgroup). Specifying `containers` is not allowed.
- `targetPercent` must be between 1 and 100.
- Requires the target cgroup to have a memory limit set. If the limit is `max` (unlimited), injection will fail.
- The allocated memory appears as RSS of the injector process. If the target container has memory requests/limits, the injected memory counts against those limits and may trigger an OOM kill by the kernel.
- If current memory usage already exceeds the target percentage, no additional memory is allocated.
- Memory allocation uses `MAP_POPULATE` which forces immediate page fault resolution. On systems with memory pressure, this may cause the kernel OOM killer to activate sooner than expected.
- At node level, the pressure affects the node's memory, impacting all workloads.

### Examples

**Consume 76% of memory with a 10-minute ramp:**

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

**Immediately consume 50% of memory:**

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
  memoryPressure:
    targetPercent: "50%"
```

**Combine with CPU pressure:**

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: resource-pressure
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

---

## 6. Disk Pressure

Throttles I/O throughput on the block device backing a given path using cgroup blkio (v1) or io (v2) controllers.

### Configuration

| Field                         | Type   | Description                                 |
| ----------------------------- | ------ | ------------------------------------------- |
| `path`                        | string | Mount point inside the container (required) |
| `throttling.readBytesPerSec`  | int    | Read throughput limit in bytes/sec          |
| `throttling.writeBytesPerSec` | int    | Write throughput limit in bytes/sec         |

At least one of `readBytesPerSec` or `writeBytesPerSec` is required.

### Constraints and Limitations

- Affects the entire block device backing the path, not just the specified path. Other mount points on the same device are also throttled.
- Cannot target specific containers (applies to all containers in the pod).
- Not compatible with `onInit` mode.
- **Cgroups v1 limitation:** Primarily affects direct I/O (O_DIRECT flag). Buffered I/O goes through the page cache and may not be throttled as expected. Cgroups v2 (`io.max`) provides more reliable throttling.
- The minor device number is always set to 0, meaning the entire disk is targeted regardless of partitioning.
- Path must exist and be mounted in the container's filesystem.

### Examples

**Throttle reads to 1 KB/s:**

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: disk-pressure-read
  namespace: chaos-demo
spec:
  level: pod
  selector:
    service: demo-curl
  count: 1
  diskPressure:
    path: /mnt/data
    throttling:
      readBytesPerSec: 1024
```

**Throttle writes to 1 KB/s:**

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: disk-pressure-write
  namespace: chaos-demo
spec:
  level: pod
  selector:
    service: demo-curl
  count: 1
  diskPressure:
    path: /mnt/data
    throttling:
      writeBytesPerSec: 1024
```

---

## 7. Disk Failure

Uses eBPF to intercept `openat` syscalls and return error codes, simulating file system failures.

### Configuration

| Field             | Type            | Default    | Description                                 |
| ----------------- | --------------- | ---------- | ------------------------------------------- |
| `paths`           | list of strings | (required) | File paths to fail (max 62 chars each)      |
| `probability`     | string          | `"100%"`   | Probability of failure per syscall (1–100%) |
| `openat.exitCode` | string          | `ENOENT`   | errno to return                             |

#### Available Exit Codes

`EACCES`, `EDQUOT`, `EEXIST`, `EFAULT`, `EFBIG`, `EINTR`, `EISDIR`, `ELOOP`, `EMFILE`, `ENAMETOOLONG`, `ENFILE`, `ENODEV`, `ENOENT`, `ENOMEM`, `ENOSPC`, `ENOTDIR`, `ENXIO`, `EOVERFLOW`, `EPERM`, `EROFS`, `ETXTBSY`, `EWOULDBLOCK`

### Constraints and Limitations

- Requires kernel with `CONFIG_BPF_KPROBE_OVERRIDE` enabled.
- Path strings limited to 62 characters (eBPF map value size limitation).
- Root path `/` is blocked by safemode unless `unsafeMode.allowRootDiskFailure: true`.
- Not compatible with `onInit` mode.
- Only intercepts the `openat` syscall. Other file-related syscalls (`open`, `openat2`, `read`, `write`, `stat`) are **not** intercepted. Applications using these syscalls directly will not experience failures.
- No explicit cleanup mechanism — the eBPF program is removed when the injector process exits. If the injector crashes, the probe may persist until the kernel cleans it up.
- Probability is evaluated per individual syscall invocation, so the actual failure rate may differ from the configured percentage depending on application access patterns.
- Can affect system stability if applied broadly with high probability, as system processes may also trigger the intercepted paths.

### Example

**Fail file opens on two paths with ENOENT:**

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: disk-failure
  namespace: chaos-demo
spec:
  level: pod
  selector:
    service: demo-curl
  count: 1
  diskFailure:
    paths:
      - /mnt/data/disk-read
      - /mnt/data/disk-write
    openat:
      exitCode: ENOENT
    probability: 100%
```

---

## 8. Container Failure

Sends a termination signal to container processes.

### Configuration

| Field    | Type | Default | Description                                                |
| -------- | ---- | ------- | ---------------------------------------------------------- |
| `forced` | bool | `false` | `false` = SIGTERM (graceful), `true` = SIGKILL (immediate) |

### Constraints and Limitations

- Pod-level only.
- **Exclusive:** cannot be combined with any other disruption type.
- Containers are continuously restarted for the disruption duration (per pod restart policy). If the restart policy is `Never`, the container stays dead.
- SIGKILL (`forced: true`) bypasses all application cleanup handlers, graceful shutdown hooks, and signal traps. Data corruption is possible if the application has in-flight writes.
- SIGTERM (`forced: false`) may be ignored by applications that do not handle it, resulting in no visible effect.
- Not re-injectable: once the container is killed, the disruption cannot be reapplied to the same target without recreation.

### Example

**Gracefully terminate a specific container:**

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: container-failure-graceful
  namespace: chaos-demo
spec:
  selector:
    service: demo-curl
  containers:
    - curl
  count: 1
  containerFailure:
    forced: false
```

---

## 9. Node Failure

Triggers a kernel panic or power-off by writing to `/proc/sysrq-trigger`. This is **irreversible** — the node becomes unavailable.

### Configuration

| Field      | Type | Default | Description                                          |
| ---------- | ---- | ------- | ---------------------------------------------------- |
| `shutdown` | bool | `false` | `false` = kernel panic, `true` = immediate power-off |

### Constraints and Limitations

- Requires `level: node` or targets nodes via pods.
- **Exclusive:** cannot be combined with any other disruption type.
- Affects ALL pods on the node — there is no way to scope the impact.
- 31-second delay before trigger to allow log collection. This is not configurable.
- **Irreversible and destructive.** The node becomes completely unavailable. Recovery depends on the cloud provider or infrastructure (auto-scaling groups, node auto-repair, etc.).
- Kernel panic (`shutdown: false`) causes an immediate crash — no graceful shutdown of any workload.
- Power-off (`shutdown: true`) may not work on all hardware or virtualization platforms. The node stays down and is not automatically restarted.
- Requires sysrq to be enabled in the kernel (`/proc/sys/kernel/sysrq`). The injector enables it automatically, but security policies may block this.
- Not re-injectable: the target is destroyed by the disruption.

### Examples

**Kernel panic on a node:**

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: node-failure
  namespace: chaos-demo
spec:
  level: node
  selector:
    node.kubernetes.io/instance-type: k3s
  count: 1
  nodeFailure:
    shutdown: false
```

**Power-off a node (no automatic restart):**

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: node-failure-shutdown
  namespace: chaos-demo
spec:
  level: node
  selector:
    node.kubernetes.io/instance-type: k3s
  count: 1
  nodeFailure:
    shutdown: true
```

---

## 10. Pod Replacement

Simulates complete pod rescheduling: cordons the node, optionally deletes PVCs, deletes the pod, then uncordons.

### Configuration

| Field                | Type | Default | Description                             |
| -------------------- | ---- | ------- | --------------------------------------- |
| `deleteStorage`      | bool | `true`  | Delete PVCs associated with the pod     |
| `forceDelete`        | bool | `false` | Force deletion with grace period 0      |
| `gracePeriodSeconds` | int  | —       | Custom deletion grace period in seconds |

### Constraints and Limitations

- Pod-level only.
- **Exclusive:** cannot be combined with any other disruption type.
- **PVC deletion is permanent and cannot be undone.** All data on the deleted volumes is lost. Use `deleteStorage: false` if data must be preserved.
- Pod recreation depends on the owning controller (Deployment, StatefulSet, etc.). Standalone pods without a controller will not be recreated.
- Node cordoning prevents all new pod scheduling on the node during the disruption, which can affect unrelated workloads that need to reschedule.
- The node is only uncordoned if it was cordoned by this specific injection. If the node was already cordoned, the injector will not uncordon it on cleanup.
- Force deletion (`forceDelete: true`) sets grace period to 0, which can leave resources in inconsistent state (e.g., mounted volumes not properly detached).
- Not re-injectable: the target pod is destroyed by the disruption.
- Requires cluster-level permissions (pods, nodes, PVCs) which may not be available in all RBAC configurations.

### Example

**Replace a pod, delete its storage:**

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: pod-replacement
  namespace: chaos-demo
spec:
  level: pod
  selector:
    service: demo-storage
  count: 1
  maxRuns: 1
  podReplacement:
    deleteStorage: true
    forceDelete: false
    gracePeriodSeconds: 30
```

---

## Targeting

All disruption types share these targeting options.

### Label Selectors

Simple key-value label matching:

```yaml
spec:
  selector:
    service: demo-curl
    team: backend
```

### Advanced Selectors

Label selectors with operators (`Exists`, `DoesNotExist`, `In`, `NotIn`):

```yaml
spec:
  advancedSelector:
    - key: app
      operator: Exists
    - key: env
      operator: In
      values:
        - production
        - staging
```

### Annotation Filters

```yaml
spec:
  filter:
    annotations:
      aws-zone: us-east-1b
```

### Container Targeting

Target specific containers within a pod (default: all containers):

```yaml
spec:
  containers:
    - my-app
    - sidecar
```

### Count

Fixed number or percentage of matching targets (percentage is rounded up, min 1%):

```yaml
spec:
  count: 3       # exactly 3 targets
  # or
  count: 50%     # half of matching targets
```

### Level

- `pod` — Targets pods. Disruption affects only the targeted pod.
- `node` — Targets nodes. Selectors match node labels. Disruption affects everything on the node.

```yaml
spec:
  level: node
  selector:
    node.kubernetes.io/instance-type: k3s
```

### Static Targeting

Lock targets at creation time. No dynamic re-targeting as pods come and go:

```yaml
spec:
  staticTargeting: true
```

### Allow Disrupted Targets

Permit targeting pods that are already under a different active disruption:

```yaml
spec:
  allowDisruptedTargets: true
```

---

## Advanced Features

### Duration

Auto-cleanup after a time period (default: 1 hour, configurable in controller):

```yaml
spec:
  duration: 30m    # supports: 45s, 15m30s, 4h
```

### Triggers (Delayed Injection)

Delay chaos pod creation and/or injection start:

```yaml
spec:
  triggers:
    createPods:
      offset: 1m                              # delay from disruption creation
      # notBefore: "2024-01-15T10:00:00Z"     # or an absolute time (RFC 3339)
    inject:
      offset: 2m                              # delay from pod creation
```

Duration countdown starts after injection begins, not after disruption creation.

### Pulse Mode

Alternate between active and dormant states. Available for Network, DNS, gRPC, CPU Pressure, Memory Pressure, Disk Pressure. Not available for Container Failure, Node Failure, Pod Replacement.

```yaml
spec:
  duration: 10m
  pulse:
    initialDelay: 1m       # optional, sleep before first active period
    activeDuration: 60s    # must be > 500ms
    dormantDuration: 20s   # must be > 500ms
  network:
    drop: 100
```

### Dry-Run Mode

Create chaos pods and select targets but do not actually inject any disruption. Useful for validating targeting before a real experiment:

```yaml
spec:
  dryRun: true
  nodeFailure:
    shutdown: false
```

### On-Init Mode

Apply disruption during pod initialization (before the main container starts). Only works with Network and DNS disruptions at pod level. Requires the handler to be enabled and pods to carry the label `chaos.datadoghq.com/disrupt-on-init: "true"`:

```yaml
spec:
  onInit: true
  network:
    drop: 100
```

### Max Runs

Limit the number of times a disruption executes (useful for Pod Replacement):

```yaml
spec:
  maxRuns: 1
```

---

## Safety Mechanisms

Safety nets are **enabled by default** and prevent accidental large-blast-radius experiments.

### Default Protections

| Safety Net          | What It Prevents                                    | Override Field                         |
| ------------------- | --------------------------------------------------- | -------------------------------------- |
| Namespace threshold | Disrupting > 80% of pods in a namespace             | `unsafeMode.disableCountTooLarge`      |
| Cluster threshold   | Disrupting > 66% of nodes in the cluster            | `unsafeMode.disableCountTooLarge`      |
| No host/port filter | Network disruption without a host or port specified | `unsafeMode.disableNeitherHostNorPort` |
| Root disk failure   | Disk failure on path `/`                            | `unsafeMode.allowRootDiskFailure`      |
| Controller node     | Node-level disruption on the controller's own node  | (always protected)                     |

### Disabling Safety Nets

```yaml
spec:
  unsafeMode:
    disableAll: true            # disable everything
    # or selectively:
    disableCountTooLarge: true
    disableNeitherHostNorPort: true
    allowRootDiskFailure: true
    config:
      countTooLarge:
        namespaceThreshold: 95  # raise namespace threshold to 95%
        clusterThreshold: 90    # raise cluster threshold to 90%
```

### Environment Annotation

Prevent accidental cross-environment disruptions by requiring a matching environment annotation:

```yaml
metadata:
  annotations:
    chaos.datadoghq.com/environment: "staging"
```

The annotation value must match the chaos controller's configured environment.

---

## Scheduling with DisruptionCron

Run disruptions on a recurring schedule using cron syntax.

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: DisruptionCron
metadata:
  name: network-drop
  namespace: chaos-demo
spec:
  schedule: "*/2 * * * *"         # every 2 minutes
  paused: false                    # set to true to pause without deleting
  delayedStartTolerance: 200s     # optional tolerance for schedule drift
  targetResource:
    kind: deployment               # "deployment" or "statefulset"
    name: demo-curl
  disruptionTemplate:
    level: pod
    count: 1
    duration: 10s
    network:
      drop: 100
```

The `disruptionTemplate` accepts the same fields as a regular `Disruption.spec`.

---

## Progressive Rollout with DisruptionRollout

Gradually apply disruptions to targets of a deployment.

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: DisruptionRollout
metadata:
  name: network-drop
  namespace: chaos-demo
spec:
  targetResource:
    kind: deployment
    name: demo-curl
  disruptionTemplate:
    level: pod
    count: 1
    duration: 10s
    network:
      drop: 100
```

---

## Reporting

Attach reporting configuration to receive Slack notifications about disruption lifecycle events:

```yaml
spec:
  reporting:
    slackChannel: team-slack-channel      # channel name or Slack channel ID
    purpose: |
      *full network drop*: _aims to validate retry capabilities of demo-curl_.
      Contact #team-test for more information.
    minNotificationType: Info             # Info, Success, Warning, Error
```

Supported notification backends (configured at the controller level): Slack, Datadog Events, HTTP webhooks, console logging.
