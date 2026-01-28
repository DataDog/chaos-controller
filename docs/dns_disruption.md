# DNS Disruption

The `dns` provides a flexible way to manipulate DNS query responses at the pod level. It intercepts DNS traffic and returns custom DNS records or simulates various failure modes.

**Supported DNS Record Types:**
* **A** - IPv4 addresses with round-robin support
* **AAAA** - IPv6 addresses with round-robin support
* **CNAME** - Canonical name (hostname aliases)
* **MX** - Mail exchange records with priority
* **TXT** - Text records (for SPF, DKIM, etc.)
* **SRV** - Service discovery records with priority, weight, and port

**Supported Failure Modes:**
* **NXDOMAIN** - Returns "domain does not exist" error (immediate failure)
* **DROP** - Silently drops DNS queries (causes timeout)
* **SERVFAIL** - Returns "server failure" error (temporary DNS server issue)
* **RANDOM** - Returns random invalid IP addresses (connection failures)

**Additional Features:**
* Round-robin IP rotation for A/AAAA records with multiple addresses
* Dynamic upstream DNS resolution (reads target pod's `/etc/resolv.conf`)
* Multiple upstream DNS servers with automatic failover
* Configurable DNS port (default: 53)
* Configurable protocol support (UDP, TCP, or both)
* Per-record TTL configuration

:information_source: **Important**: DNS disruption can only be applied at the **pod level** (not node level). The same hostname can have multiple record types configured (e.g., both A and AAAA for dual-stack support).

## Specification

The disruption has the following fields under `dns`:

* **records** (required): List of DNS records to fake or disrupt
  * **hostname** (required): Domain name to intercept (e.g., `api.example.com`)
  * **record** (required): DNS record configuration
    * **type** (required): DNS record type (A, AAAA, CNAME, MX, TXT, SRV)
    * **value** (required): Record value or special failure mode
    * **ttl** (optional): Time-to-live in seconds (default: 0)
* **port** (optional): DNS port to intercept (default: 53)
  * Note: The DNS responder internally uses separate ports for protocol handling:
    - When protocol is "udp" or "tcp": responder uses port 5353
    - When protocol is "both": UDP responder uses port 5353, TCP responder uses port 5354
  * These internal ports are managed automatically and don't need to be configured
* **protocol** (optional): Protocol to disrupt - `udp`, `tcp`, or `both` (default: both)

## Examples

:information_source: **Note**: Many examples in the YAML files (`examples/network_dns_disruption_*.yaml`) are commented out to keep the default examples simple and focused. Uncomment the record types you want to test, or use them as templates for your own disruptions. All commented features are fully implemented and production-ready.

### Basic A Record Disruption

Redirect DNS queries for a specific hostname to a custom IP address:

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: dns-basic-a-record
  namespace: chaos-demo
spec:
  level: pod
  selector:
    service: demo-curl
  count: 1
  duration: 15m
  dns:
    records:
      - hostname: api.example.com
        record:
          type: A
          value: "192.168.1.100"
```

### NXDOMAIN Failure Mode

Simulate a "domain does not exist" error for immediate failure testing:

```yaml
dns:
  records:
    - hostname: api.example.com
      record:
        type: A
        value: NXDOMAIN  # Returns "domain not found" error
```

### DROP (Timeout) Behavior

Silently drop DNS queries to simulate timeout scenarios:

```yaml
dns:
  records:
    - hostname: api.example.com
      record:
        type: A
        value: DROP  # Silently drops queries, causes timeout
```

### SERVFAIL Error

Simulate DNS server failure errors:

```yaml
dns:
  records:
    - hostname: api.example.com
      record:
        type: A
        value: SERVFAIL  # Returns "server failure" error
```

### RANDOM Invalid IP

Return random invalid IP addresses to test connection error handling:

```yaml
dns:
  records:
    - hostname: api.example.com
      record:
        type: A
        value: RANDOM  # Returns random invalid IP address
```

**Note**: RANDOM mode generates random IP addresses specifically in the RFC 5737 TEST-NET-1 range (192.0.2.0/24). This is a reserved range designated for documentation and examples, ensuring the IPs are guaranteed to be non-routable and won't accidentally reach real hosts.

### Round-Robin IP Addresses

Configure multiple IP addresses for load balancing simulation:

```yaml
dns:
  records:
    - hostname: api.example.com
      record:
        type: A
        value: "192.168.1.10,192.168.1.11,192.168.1.12"  # Round-robin
        ttl: 60
```

### Dual-Stack Support (IPv4 and IPv6)

Configure both A and AAAA records for the same hostname to support dual-stack networking:

```yaml
dns:
  records:
    # IPv4 addresses (A record)
    - hostname: api.example.com
      record:
        type: A
        value: "192.168.1.100,192.168.1.101"
        ttl: 60
    # IPv6 addresses (AAAA record) for the same hostname
    - hostname: api.example.com
      record:
        type: AAAA
        value: "2001:db8::1,2001:db8::2"
        ttl: 60
```

This allows applications that support both IPv4 and IPv6 to resolve the hostname using either protocol.

### Multiple Record Types

Configure different DNS record types for various hostnames:

```yaml
dns:
  records:
    # A record for API endpoint
    - hostname: api.example.com
      record:
        type: A
        value: "192.168.1.100"
    # MX records for mail server
    - hostname: example.com
      record:
        type: MX
        value: "10 mail1.example.com,20 mail2.example.com"
        ttl: 3600
    # SRV record for service discovery
    - hostname: _sip._tcp.example.com
      record:
        type: SRV
        value: "10 60 5060 sipserver.example.com"
        ttl: 600
    # CNAME record for aliasing
    - hostname: www.example.com
      record:
        type: CNAME
        value: "cdn.example.com"
        ttl: 300
```

### Custom Port and Protocol

Configure DNS disruption for non-standard DNS ports or specific protocols:

```yaml
dns:
  records:
    - hostname: custom-dns.example.com
      record:
        type: A
        value: "10.0.0.1"
  port: 5353        # Custom DNS port
  protocol: udp     # Only UDP (options: udp, tcp, both)
```

### Complete Example

A comprehensive example demonstrating multiple failure modes:

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: dns-comprehensive-test
  namespace: chaos-demo
spec:
  level: pod
  selector:
    service: demo-app
  count: 1
  duration: 30m
  dns:
    records:
      # Normal operation: redirect to test environment
      - hostname: prod-api.example.com
        record:
          type: A
          value: "10.0.1.100"
          ttl: 60

      # Test NXDOMAIN handling
      - hostname: nonexistent.example.com
        record:
          type: A
          value: NXDOMAIN

      # Test timeout handling
      - hostname: slow-service.example.com
        record:
          type: A
          value: DROP

      # Test connection error handling
      - hostname: broken-service.example.com
        record:
          type: A
          value: RANDOM
```

## Record Type Details

### A Records (IPv4)
IPv4 address records. Supports comma-separated list for round-robin behavior.

**Format**: `"192.168.1.1"` or `"192.168.1.1,192.168.1.2,192.168.1.3"`

**Special values**: NXDOMAIN, DROP, SERVFAIL, RANDOM

**Example**:
```yaml
record:
  type: A
  value: "192.168.1.10,192.168.1.11"  # Round-robin between two IPs
  ttl: 60
```

### AAAA Records (IPv6)
IPv6 address records. Supports comma-separated list for round-robin behavior.

**Format**: `"2001:db8::1"` or `"2001:db8::1,2001:db8::2"`

**Special values**: NXDOMAIN, DROP, SERVFAIL, RANDOM

**Example**:
```yaml
record:
  type: AAAA
  value: "2001:db8::1,2001:db8::2"
  ttl: 300
```

### CNAME Records (Canonical Name)
Hostname aliases that redirect to another hostname.

**Format**: `"target.example.com"`

**Example**:
```yaml
record:
  type: CNAME
  value: "cdn.example.com"
  ttl: 300
```

### MX Records (Mail Exchange)
Mail server records with priority.

**Format**: `"priority hostname,priority hostname"` (e.g., `"10 mail1.example.com,20 mail2.example.com"`)

**Example**:
```yaml
record:
  type: MX
  value: "10 mail1.example.com,20 mail2.example.com"
  ttl: 3600
```

### TXT Records (Text)
Text records for arbitrary data (SPF, DKIM, domain verification, etc.).

**Format**: Any string value

**Example**:
```yaml
record:
  type: TXT
  value: "v=spf1 include:_spf.example.com ~all"
  ttl: 300
```

### SRV Records (Service Discovery)
Service location records with priority, weight, port, and target.

**Format**: `"priority weight port target"` (e.g., `"10 60 5060 sipserver.example.com"`)

**Example**:
```yaml
record:
  type: SRV
  value: "10 60 5060 sipserver.example.com,20 40 5060 sipserver2.example.com"
  ttl: 600
```

## Failure Modes

| Failure Mode | DNS Response                      | Client Behavior                                      | Use Case                                                                     |
|--------------|-----------------------------------|------------------------------------------------------|------------------------------------------------------------------------------|
| **NXDOMAIN** | Returns RCODE 3 (Name Error)      | Immediate "domain not found" error                   | Test immediate error handling, client retry logic after DNS errors           |
| **DROP**     | No response (silent drop)         | Connection timeout after resolver timeout period     | Test timeout handling, connection timeout configuration, exponential backoff |
| **SERVFAIL** | Returns RCODE 2 (Server Failure)  | "DNS server failure" error, may retry                | Test DNS server failure handling, retry logic for temporary DNS issues       |
| **RANDOM**   | Returns random/invalid IP address | Connection refused or timeout to unreachable address | Test connection error handling, fallback mechanisms, circuit breakers        |

### When to Use Each Failure Mode

**NXDOMAIN**: Use when testing:
* Immediate error detection
* Retry logic after permanent DNS failures
* Fallback to alternative services
* Error message clarity to users

**DROP**: Use when testing:
* Timeout configuration (are timeouts set correctly?)
* Connection pool behavior under slow DNS
* Exponential backoff and jitter
* User experience during DNS server unavailability

**SERVFAIL**: Use when testing:
* Retry logic for transient DNS failures
* Fallback to alternative DNS servers
* Degraded mode operation
* Monitoring and alerting on DNS errors

**RANDOM**: Use when testing:
* Connection error handling
* Connection pool cleanup
* Circuit breaker patterns
* Fallback to alternative endpoints

## Use Cases

### Test DNS Failure Handling
Verify your application handles DNS errors gracefully:
```yaml
# Simulate DNS unavailability
- hostname: api.example.com
  record:
    type: A
    value: NXDOMAIN
```

### Test Retry Logic
Validate exponential backoff and retry mechanisms:
```yaml
# Simulate temporary DNS failure
- hostname: api.example.com
  record:
    type: A
    value: SERVFAIL
```

### Test Timeout Configuration
Ensure timeout settings are appropriate:
```yaml
# Simulate DNS query timeout
- hostname: api.example.com
  record:
    type: A
    value: DROP
```

### Test Load Balancing
Verify round-robin DNS behavior:
```yaml
# Multiple IPs for round-robin
- hostname: api.example.com
  record:
    type: A
    value: "192.168.1.10,192.168.1.11,192.168.1.12"
    ttl: 60
```

### Test Service Discovery
Validate SRV record handling for service discovery:
```yaml
# Service discovery with SRV records
- hostname: _http._tcp.example.com
  record:
    type: SRV
    value: "10 60 8080 server1.example.com,20 40 8080 server2.example.com"
```

### Test DNS Caching
Verify TTL and caching behavior:
```yaml
# Test DNS caching with TTL
- hostname: cached.example.com
  record:
    type: A
    value: "192.168.1.100"
    ttl: 300  # Should be cached for 5 minutes
```

### Test Connection Pooling
Ensure connection errors are handled correctly:
```yaml
# Simulate unreachable endpoint
- hostname: api.example.com
  record:
    type: A
    value: RANDOM  # Returns invalid IP
```

## DNS Disruption vs Network Disruption

Understanding when to use DNS disruption versus network disruption:

| Feature               | DNS Disruption                                          | Network Disruption                                   |
|-----------------------|---------------------------------------------------------|------------------------------------------------------|
| **What it disrupts**  | DNS queries and responses                               | Network packets (TCP/UDP/ICMP)                       |
| **Use for**           | Testing DNS failure handling, service discovery issues  | Testing packet loss, latency, bandwidth limits       |
| **Granularity**       | Per-hostname                                            | Per-host, per-service, per-port                      |
| **Failure modes**     | NXDOMAIN, SERVFAIL, DROP, RANDOM, record spoofing       | Packet drop, corruption, delay, bandwidth limit      |
| **Level support**     | Pod only                                                | Pod and Node                                         |
| **Best for**          | DNS-specific resilience testing                         | Network-level resilience testing                     |
| **Example use cases** | Test DNS caching, service discovery, DNS error handling | Test network partitions, latency spikes, packet loss |

### When to Use DNS Disruption

Use **DNS disruption** when testing:
* Application behavior when DNS returns errors (NXDOMAIN, SERVFAIL)
* DNS caching and TTL handling
* Service discovery with SRV records
* Hostname resolution in microservices
* DNS-based load balancing and round-robin
* Fallback mechanisms when DNS fails
* Connection retry logic after DNS errors

### When to Use Network Disruption

Use **Network disruption** when testing:
* Timeout behavior for established connections
* Retry logic for connection failures (not DNS failures)
* Network partition scenarios
* Latency sensitivity and performance
* Bandwidth limitations
* Packet loss and corruption handling

### Combined Scenarios

For comprehensive testing, you may want to use both:
1. **DNS disruption** to simulate DNS server issues
2. **Network disruption** to simulate network connectivity issues after successful DNS resolution

## How It Works

```
┌──────────────────────────────────────────────────────────────────────────┐
│ Kubernetes Node                                                          │
│                                                                          │
│  ┌──────────────────────────────────────────────────────────────────┐    │
│  │ Target Pod Network Namespace                                     │    │
│  │                                                                  │    │
│  │  ┌──────────────────┐                                            │    │
│  │  │ Application      │                                            │    │
│  │  │                  │                                            │    │
│  │  │ DNS Query:       │                                            │    │
│  │  │ api.example.com  │                                            │    │
│  │  └────────┬─────────┘                                            │    │
│  │           │ Port 53                                              │    │
│  │           ↓                                                      │    │
│  │  ┌──────────────────┐                                            │    │
│  │  │ IPTables DNAT    │                                            │    │
│  │  │ (CHAOS-DNS)      │                                            │    │
│  │  │ 53 → chaos-pod-ip:5353                                        │    │
│  │  └────────┬─────────┘                                            │    │
│  └───────────┼──────────────────────────────────────────────────────┘    │
│              │ Redirected to Chaos Pod                                   │
│              ↓                                                           │
│  ┌──────────────────────────────────────────────────────────────────┐    │
│  │ Chaos Injector Pod Network Namespace                             │    │
│  │                                                                  │    │
│  │  ┌──────────────────┐        ┌────────────────────┐              │    │
│  │  │ DNS Responder    │        │ Configured Records:│              │    │
│  │  │ (github.com/     │───────▶│ - api.example.com  │              │    │
│  │  │  miekg/dns)      │        │   → NXDOMAIN       │              │    │
│  │  │ Port 5353/5354   │        └────────────────────┘              │    │
│  │  └────────┬─────────┘                                            │    │
│  │           │                                                      │    │
│  │           │ Match? → Return custom response                      │    │
│  │           │ No match? ↓                                          │    │
│  └───────────┼──────────────────────────────────────────────────────┘    │
│              │                                                           │
│              ↓ Forward to upstream                                       │
│  ┌──────────────────────────────────────────────────┐                    │
│  │ Upstream DNS (from /etc/resolv.conf)             │                    │
│  │ - 10.96.0.10:53 (CoreDNS)                        │                    │
│  │ - 8.8.8.8:53 (fallback)                          │                    │
│  └──────────────────────────────────────────────────┘                    │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│ Chaos Injector Pod Setup Process                            │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ 1. Read /proc/<pid>/root/etc/resolv.conf             │   │
│  │ 2. Start DNS responder on port 5353/5354             │   │
│  │    (runs in chaos pod namespace)                     │   │
│  │ 3. Enter target network namespace (nsenter)          │   │
│  │ 4. Setup IPTables DNAT rules                         │   │
│  │    (53 → chaos-pod-ip:5353/5354)                     │   │
│  │ 5. Exit target network namespace                     │   │
│  │ 6. Monitor disruption lifecycle                      │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

The DNS disruption works by redirecting DNS traffic from the target pod to a DNS responder in the chaos injector pod:

1. **DNS Discovery**: The chaos injector reads the target pod's `/etc/resolv.conf` to discover upstream DNS servers (e.g., cluster DNS)
2. **DNS Responder**: A lightweight DNS responder starts in the **chaos injector pod's namespace** (not in the target pod)
3. **Traffic Redirection**: IPTables rules in the target pod's network namespace redirect DNS traffic from port 53 to the chaos pod's IP and responder port (5353/5354)
4. **Query Interception**: The responder intercepts queries for configured hostnames and returns custom responses
5. **Upstream Forwarding**: Non-matching queries are forwarded to upstream DNS servers with automatic failover

The DNS responder uses the `github.com/miekg/dns` library and is implemented as a lightweight goroutine-based server. It runs in the chaos injector pod's namespace and receives redirected traffic via IPTables DNAT rules configured in the target pod's namespace.

## Performance Considerations

### Overhead and Resource Usage

**Minimal Overhead**: DNS disruption adds negligible performance overhead to target pods:
* **DNS Responder**: Lightweight goroutine-based server (~5-10MB memory footprint)
* **IPTables Rules**: Negligible CPU overhead for DNAT rules
* **DNS Traffic Only**: Only intercepts DNS queries (port 53), application traffic is unaffected
* **No Additional Hops**: Queries forwarded to upstream DNS have the same latency as normal operation

### Scaling

* **Multiple Pods**: DNS disruption can be applied to multiple pods simultaneously
* **Independent**: Each disruption operates independently in its own network namespace
* **No Interference**: Multiple disruptions do not interfere with each other

### Network Impact

* **DNS Only**: Only affects DNS resolution (port 53)
* **Application Traffic**: HTTP, gRPC, database connections, etc. are not affected
* **Transparent**: Applications see DNS responses as if they came from normal DNS servers

### Upstream DNS

* **Automatic Discovery**: Reads target pod's `/etc/resolv.conf` to find upstream DNS servers
* **No Latency**: Non-matching queries are forwarded directly to upstream DNS
* **Automatic Failover**: When multiple upstream DNS servers are configured in `/etc/resolv.conf`, the DNS responder automatically tries each server in order until one succeeds. If a primary DNS server fails, fallback servers are used transparently.
* **Cluster DNS**: Works seamlessly with Kubernetes CoreDNS and supports multiple nameservers
* **Error Handling**: Returns SERVFAIL only if all upstream servers fail

### Round-Robin Performance

* **Thread-Safe**: IP rotation uses mutex for thread-safe operation
* **No Locks on Query**: Lock is only held during IP selection (microseconds)
* **No Degradation**: Round-robin has no measurable performance impact

## Important Notes and Limitations

:warning: **Pod-level only**: DNS disruption can only be applied at `level: pod` (not node level). This is because it operates within the target pod's network namespace.

:information_source: **Multiple Record Types per Hostname**: The same hostname can have multiple record types configured (e.g., both A and AAAA records for dual-stack IPv4/IPv6 support). However, the same hostname+type combination can only appear once. For example:
- ✅ Valid: `www.example.com` with type `A` and `www.example.com` with type `AAAA`
- ❌ Invalid: `www.example.com` with type `A` configured twice

:information_source: **Subdomain Matching**: DNS disruption supports subdomain matching. When you configure a hostname like `example.com`, it will also match subdomains like `api.example.com`, `sub.api.example.com`, etc. Exact wildcard patterns (e.g., `*.example.com`) are not required - subdomain matching is automatic.

**Example**: Configuring `hostname: example.com` will intercept queries for:
- `example.com` (exact match)
- `www.example.com` (subdomain match)
- `api.example.com` (subdomain match)
- `sub.api.example.com` (nested subdomain match)

:warning: **Case-insensitive**: Hostname matching is case-insensitive. `api.Example.COM` and `api.example.com` are treated as the same hostname.

:information_source: **Thread-Safe Round-Robin**: The round-robin IP rotation for A/AAAA records uses mutex locking for thread-safe operation across concurrent DNS queries.

:information_source: **RFC 5737 Compliance**: RANDOM mode uses the TEST-NET-1 IP range (192.0.2.0/24) as specified in RFC 5737, ensuring generated IPs are documentation-safe and non-routable.

:warning: **Protocol support**: Both UDP and TCP DNS queries are supported. Most DNS clients use UDP by default, falling back to TCP for large responses.

:information_source: **Upstream DNS**: The chaos injector automatically reads the target pod's `/etc/resolv.conf` to discover upstream DNS servers (typically the cluster DNS like CoreDNS).

:information_source: **Failover**: If multiple upstream DNS servers are configured in `resolv.conf`, the DNS responder will try each server in order until one succeeds.

:information_source: **TTL**: Default TTL is 0 (no caching). You can customize the TTL per record to test DNS caching behavior.

:information_source: **Safe Mode**: DNS disruption respects safe mode settings. Certain high-risk configurations may be blocked by safe mode.

## Manual cleanup instructions

:information_source: All those commands must be executed on the infected host (except for `kubectl`).

---

:warning: you must enter the pod network namespace first.

* Identify the container IDs of your pod

```
kubectl get -ojson pod demo-curl-547bb9c686-57484 | jq '.status.containerStatuses[].containerID'
"containerd://cb33d4ce77f7396851196043a56e625f38429720cd5d3153cb061feae6038460"
"containerd://629c7da02cbcf77c6b7131a59f5be50579d9e374433a444210b6547186dd5f0d"
```

* Find one of the container PID

```
# crictl inspect cb33d4ce77f7396851196043a56e625f38429720cd5d3153cb061feae6038460 | grep pid
    "pid": 5607,
            "pid": 1
            "type": "pid"
```

* Enter the network namespace

```
# nsenter --net=/proc/5607/ns/net
```

---

* Remove iptables rules jumping to the `CHAOS-DNS` chain

```
# iptables-save | grep -- '-j CHAOS-DNS'
-A PREROUTING -p tcp -j CHAOS-DNS
-A OUTPUT -p tcp -j CHAOS-DNS
# iptables -t nat -D PREROUTING -p tcp -j CHAOS-DNS
# iptables -t nat -D OUTPUT -p tcp -j CHAOS-DNS
```

* Remove iptables `CHAOS-DNS` chain

```
# iptables -t nat -F CHAOS-DNS
# iptables -t nat -X CHAOS-DNS
```

---

## Links and References

### Implementation Files
* [DNS Disruption API Spec](../api/v1beta1/dns_disruption.go) - Go API specification
* [DNS Responder](../network/dns_responder.go) - DNS responder implementation
* [DNS Disruption Injector](../injector/dns_disruption.go) - Injector logic

### Example Files
* [Failure Modes](../examples/network_dns_disruption_failures.yaml)
* [Mixed Record Types](../examples/network_dns_disruption_mixed.yaml)

### Related Documentation
* [Network Disruption](network_disruption.md) - For packet-level network disruptions
* [Targeting](targeting.md) - For pod selection and targeting details
* [Features Guide](features.md) - For duration, count, and other common disruption fields
* [Safeguards](safemode.md) - For safe mode configuration
* [Changes Handling](changes_handling.md) - For handling environment changes during disruptions

### External Resources
* [miekg/dns library](https://github.com/miekg/dns) - Go DNS library used by the DNS responder
* [DNS RFC 1035](https://tools.ietf.org/html/rfc1035) - DNS specification
* [DNS Record Types](https://en.wikipedia.org/wiki/List_of_DNS_record_types) - Complete list of DNS record types
