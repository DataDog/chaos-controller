# Spec: Docker Integration Tests for Network Disruption Injector

## Objective

Enable developers to validate `networkDisruptionInjector` behavior against real
Linux kernel primitives (tc, iptables, netlink) inside Docker, without a
Kubernetes cluster. Supplements existing mocked unit tests and full E2E.

**Target user**: developer iterating on network disruption logic locally.
**Success**: `make test-integration` runs in < 2 minutes and asserts both
structural (tc rules exist) and behavioral (HTTP traffic is affected) outcomes.

---

## Architecture

The test binary must run on Linux because `netns.Set()` is a Linux syscall.
`make test-integration` cross-compiles for Linux, builds a test Docker image,
and runs it privileged with the host `/proc` and Docker socket mounted.

```
make test-integration
  │
  ├── GOOS=linux go test -c -tags=integration -o bin/integration.test ./injector/
  ├── docker build -f Dockerfile.integration -t chaos-integration-test:local .
  └── docker run --rm --privileged \
        -v /proc:/mnt/proc \
        -v /var/run/docker.sock:/var/run/docker.sock \
        -e CHAOS_INJECTOR_MOUNT_PROC=/mnt/proc/ \
        chaos-integration-test:local

       Inside the container, the Ginkgo suite:
       ┌──────────────────────────────────────────────┐
       │  testcontainers-go                           │
       │    creates → target container (nginx:alpine) │
       │    creates → sender container (alpine+wget)  │
       │                                              │
       │  container.New("docker://<target-id>")       │
       │    → docker inspect → target host PID        │
       │                                              │
       │  netns.NewManager(log, pid)                  │
       │    → opens /mnt/proc/<pid>/ns/net            │
       │                                              │
       │  NewNetworkDisruptionInjector(spec, config)  │
       │    config.TrafficController = real tc        │
       │    config.IPTables          = real iptables  │
       │    config.NetlinkAdapter    = real netlink   │
       │    config.K8sClient         = fake k8s       │
       │    config.BPFDisruptCmdRunner = mock         │
       │                                              │
       │  injector.Inject()                           │
       │    → enters target netns                     │
       │    → applies tc/iptables rules               │
       │                                              │
       │  assert structural: docker exec tc qdisc show│
       │  assert behavioral: sender HTTP → target     │
       │                                              │
       │  injector.Clean()                            │
       │    → removes rules, verify clean state       │
       └──────────────────────────────────────────────┘
```

---

## Commands

```makefile
# Run integration tests (requires Docker Desktop)
make test-integration

# Filter to specific test(s)
make test-integration TEST_ARGS="-test.run TestLatency"

# Compile test binary only (no run)
GOOS=linux GOARCH=$(go env GOARCH) go test -c -tags=integration \
    -o bin/integration.test ./injector/
```

Makefile target:

```makefile
.PHONY: test-integration
test-integration:
	GOOS=linux GOARCH=$(shell go env GOARCH) \
	  go test -c -tags=integration -o bin/integration.test ./injector/
	docker build -f Dockerfile.integration -t chaos-integration-test:local .
	docker run --rm --privileged \
	  -v /proc:/mnt/proc \
	  -v /var/run/docker.sock:/var/run/docker.sock \
	  -e CHAOS_INJECTOR_MOUNT_PROC=/mnt/proc/ \
	  -e DOCKER_HOST=unix:///var/run/docker.sock \
	  chaos-integration-test:local $(TEST_ARGS)
```

---

## Project Structure

```
chaos-controller/
├── Dockerfile.integration               # new — test runner image
├── SPEC.md                              # this file
├── bin/
│   └── integration.test                # generated, add to .gitignore
└── injector/
    ├── injector_suite_test.go           # existing unit suite (unchanged)
    ├── network_disruption_test.go       # existing unit tests (unchanged)
    ├── integration_suite_test.go        # new — Ginkgo suite for integration
    └── network_disruption_integration_test.go  # new — integration scenarios
```

### `Dockerfile.integration`

```dockerfile
FROM debian:12-slim
RUN apt-get update -qq && apt-get install -y -qq iproute2 iptables wget 2>/dev/null
COPY bin/integration.test /integration.test
ENTRYPOINT ["/integration.test", "-test.v"]
```

---

## Code Style

- **Test framework**: Ginkgo v2 + Gomega (consistent with existing `injector/` tests)
- **Container management**: testcontainers-go v0.42+
- **Build tag**: `//go:build integration` on all new files
- **Package**: `injector_test` (consistent with existing external test package)
- **K8s client**: `fake.NewSimpleClientset()` with pre-seeded Services (consistent
  with existing unit tests — see `network_disruption_test.go` `k8sClient` setup)
- **Mocked only**: `BPFDisruptCmdRunner` (CmdRunner), `BPFConfigInformer` — same
  mocks as unit tests; BPF path is out of scope for this integration layer
- **Real drivers**: `network.NewTrafficController`, `network.NewIPTables`,
  `network.NewNetlinkAdapter`, `netns.NewManager`, `container.New`
- No `dryRun` mode — real kernel calls are the point
- Container cleanup: always call `defer container.Terminate(ctx)` in `BeforeEach`

---

## Test Scenarios

### Suite setup (`integration_suite_test.go`)

```
BeforeSuite:
  - verify CHAOS_INJECTOR_MOUNT_PROC is set (fail fast if not in privileged container)
  - verify /var/run/docker.sock is accessible
  - initialize shared logger, metrics sink (noop)

AfterSuite:
  - no-op (testcontainers-go Ryuk handles container cleanup)
```

### Scenario 1 — Latency injection (`network_disruption_integration_test.go`)

```
Describe "network latency disruption"
  BeforeEach:
    - create target container: nginx:alpine on isolated bridge network
    - create sender container: alpine with wget, on same bridge network
    - create NetworkDisruptionInjector with 200ms latency spec
    - measure baseline HTTP latency (must be < 50ms, else BeforeEach fails)

  It "applies netem delay rule to target netns"
    - call injector.Inject()
    - docker exec target: `tc qdisc show dev eth0`
    - assert output contains "netem" and "delay 200ms"

  It "causes measurable HTTP latency increase"
    - call injector.Inject()
    - sender: wget GET http://<target-ip>/ with timeout, record latency
    - assert measured latency > 150ms (75% of injected 200ms accounts for jitter)

  AfterEach:
    - call injector.Clean()
    - docker exec target: `tc qdisc show dev eth0`
    - assert "netem" not present
```

### Scenario 2 — Packet loss injection

```
Describe "network packet loss disruption"
  It "applies netem loss rule to target netns"
    - inject 50% packet loss
    - assert `tc qdisc show` contains "netem" and "loss 50%"

  It "causes measurable HTTP failure rate"
    - inject 50% loss
    - sender: 20 sequential wget requests with short timeout
    - assert failure rate > 20% (conservative: 50% loss on TCP means > 20% failures)

  AfterEach:
    - Clean() → assert no netem rule
```

### Scenario 3 — Clean state verification

```
Describe "clean up"
  It "removes all tc rules after Clean()"
    - Inject() then Clean()
    - assert tc qdisc show: no netem, no prio, default qdisc only

  It "HTTP traffic recovers after Clean()"
    - Inject() → measure failure/latency
    - Clean() → measure again
    - assert post-clean success rate == 100% and latency < 50ms
```

### Scenario 4 — Bandwidth limit (stretch, same spec)

```
Describe "bandwidth limit disruption"
  It "applies tbf/htb rule limiting to target netns"
    - inject 1mbit bandwidth limit
    - assert `tc qdisc show` contains tbf or htb rule
    - assert measured throughput < 2mbit (2x headroom for test stability)
```

---

## Testing Strategy

| Layer | What | How | Speed |
|-------|------|-----|-------|
| Unit (existing) | Logic, rule computation | All interfaces mocked | < 5s |
| Integration (this spec) | Real kernel + Docker | tc/iptables/netlink real, K8s fake | < 2min |
| E2E (existing) | Full Kubernetes flow | Real cluster required | > 10min |

**Integration test principles:**
- Each `It` block is independent: containers created in `BeforeEach`, cleaned in `AfterEach`
- Baseline assertions in `BeforeEach` prevent false positives (test infra problems
  show as setup failures, not test failures)
- Structural assertions run first, behavioral second — a structural failure skips behavioral
- Flakiness mitigation: behavioral assertions use conservative thresholds and
  `Eventually` with 10s timeout / 500ms polling rather than point-in-time checks
- Tests must pass on: macOS (Docker Desktop arm64/amd64) and Linux CI runners

---

## Boundaries

### Always
- Create containers on an isolated Docker bridge network (not host network)
- Terminate all containers in `AfterEach` (not just `AfterSuite`)
- Verify `CHAOS_INJECTOR_MOUNT_PROC` is set at suite start — fail with clear message if not
- Assert clean state after every `Clean()` call — never leave leaked tc rules

### Ask first
- Adding a new disruption kind to integration tests (risk of scope creep)
- Changing behavioral assertion thresholds (affects flakiness/false positive balance)
- Enabling BPF path (requires `bpfdisrupt` binary in test image and additional setup)

### Never
- Require a running Kubernetes cluster
- Modify the host network namespace (only modify target container netns)
- Run integration tests as part of `make test` (unit test target must stay fast)
- Use `--network=host` for test containers
- Skip `AfterEach` cleanup (no leaked containers)

---

## Open Questions (all resolved)

- [x] netns-enter from sibling container — **VALIDATED** (see `docs/ideas/docker-integration-tests.md`)
- [x] BPF syscall in Docker Desktop — **VALIDATED** (MAP_CREATE, PROG_LOAD both work on arm64)
- [x] cgroup v2 in Docker Desktop — **VALIDATED** (unified hierarchy, all controllers present)
- [x] testcontainers-go license — **VALIDATED** (MIT, all transitive deps MIT/Apache/BSD)
- [x] iptables binary in debian:12-slim — **VALIDATED**. `apt-get update && apt-get install
      iproute2 iptables` succeeds. Installs `iptables v1.8.9 (nf_tables)` and
      `tc iproute2-6.1.0`. The `go-iptables` library calls the `iptables` binary
      directly — nf_tables backend is a drop-in. If issues arise, add
      `update-alternatives --set iptables /usr/sbin/iptables-legacy` to Dockerfile.
- [x] Ryuk inside DinD (mounted socket) — **VALIDATED** by reading testcontainers-go
      source. When `InAContainer() == true` with a Unix socket, testcontainers-go
      uses the Docker bridge **gateway IP** (e.g. `172.17.0.1`) instead of
      `localhost` to connect to Ryuk's mapped port. Works without
      `TESTCONTAINERS_RYUK_DISABLED=true`. Safety valve:
      `TESTCONTAINERS_HOST_OVERRIDE=172.17.0.1` if auto-detection fails.
