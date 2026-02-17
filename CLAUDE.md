# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Kubernetes operator for chaos engineering (Datadog). Injects systemic failures (network, CPU, disk, DNS, gRPC, container/node failure) into Kubernetes clusters at scale. Built with Kubebuilder v3 and controller-runtime.

## Build Commands

```bash
make docker-build-all          # Build all Docker images (manager, injector, handler)
make manifests                 # Generate CRDs and RBAC manifests
make generate                  # Generate Go code (deepcopy, etc.)
make generate-mocks            # Regenerate mocks (mockery v2.53.5)
make chaosli                   # Build CLI helper tool
make godeps                    # go mod tidy + vendor
make header                    # Check/fix license headers
```

## Testing

```bash
make test                                  # Run all unit tests (Ginkgo v2)
make test TEST_ARGS="injector"             # Filter tests by package name
make test TEST_ARGS="--until-it-fails"     # Detect flaky tests
make test GINKGO_PROCS=4                   # Control parallelism
make e2e-test                              # End-to-end tests (requires cluster)
```

Tests use **Ginkgo v2** (BDD) with **Gomega** matchers. Coverage output: `cover.profile`.

## Linting and Formatting

```bash
make lint                      # golangci-lint (v2.8.0)
make fmt                       # Format Go code
make vet                       # Go vet
make spellcheck                # Spell check markdown docs
```

## Local Development

```bash
make lima-all                  # Start local k3s cluster with controller
make lima-redeploy             # Rebuild and redeploy to local cluster
make lima-stop                 # Stop local cluster
make watch                     # Auto-rebuild on file changes
make debug                     # Prepare for IDE debugging
make run                       # Run controller locally
```

## Architecture

Three main components, each with its own Dockerfile in `bin/`:

- **Manager** (`main.go`, `controllers/`): Long-running controller pod. Watches Disruption CRDs, selects targets via label selectors, creates chaos pods, manages lifecycle with finalizers. Reconciliation flow: add finalizer → compute spec hash → select targets → create chaos pods → track injection status.
- **Injector** (`injector/`, `cli/injector/`): Runs as ephemeral chaos pods on target nodes. Performs actual disruption using Linux primitives (cgroups, tc, iptables, eBPF). One chaos pod per target per disruption kind.
- **Handler** (`webhook/`, `cli/handler/`): Admission webhook for pod initialization-time network disruptions.

### CRDs (api/v1beta1/)

- **Disruption**: Main resource defining what failure to inject and targeting criteria
- **DisruptionCron**: Scheduled/recurring disruptions
- **DisruptionRollout**: Progressive disruption rollout

### Key Packages

- `targetselector/` — Target selection logic (labels, count, filters, safety nets)
- `safemode/` — Safety mechanisms to prevent dangerous disruptions
- `eventnotifier/` — Notifications (Slack, Datadog, HTTP)
- `o11y/` — Observability (metrics, tracing, profiling for Datadog and Prometheus)
- `cloudservice/` — Cloud provider integrations
- `ebpf/` — eBPF programs for network disruption
- `grpc/disruptionlistener/` — gRPC service for disruption events
- `chart/` — Helm chart for deployment

### Code Generation

CRDs are defined in `api/v1beta1/` with kubebuilder markers. After modifying types, run `make manifests generate`. Mocks are generated with mockery into `mocks/`. Protobuf definitions live in `grpc/` and `dogfood/`.

## Requirements

- Kubernetes >= 1.16 (not 1.20.0-1.20.4)
- Go 1.25.6
- Docker with buildx (multi-arch: amd64, arm64)
