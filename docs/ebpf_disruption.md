# eBPF disruptions

eBPF _Extended Berkeley Packet Filter_ is a framework that allows users to load and run custom programs within the kernel of the operating system.
That means it can extend or even modify the way the kernel behaves. It is useful for observability, security, chaos, etc...

## Knowledge requirement

The purpose of this document is not to teach how eBPF works and how to use it, but how to create a disruption based on eBPF within the chaos-controller.
To learn more about eBPF you can find the following resources to help you:
- [Basics](https://isovalent.com/data/liz-rice-what-is-ebpf.pdf)
- [First eBPF program](https://github.com/lizrice/ebpf-beginners)
- [eBPF portability](https://nakryiko.com/posts/bpf-portability-and-co-re/)
- [libbpf](https://github.com/libbpf/libbpf)

## Architecture

```bash
bin/injector # Injector Docker image with embedded eBPF binaries
└── Dockerfile # Multi-stage build: EBPF builder + Go builder + final image
ebpf # Contains eBPF disruptions source code
├── Makefile # Build eBPF disruptions (called during Docker build)
├── const-arm.go # Contains constants for arm64 architecture
├── const-x64.go # Contains constants for amd64 architecture
├── disk-failure # eBPF project - disk failure
│   ├── injection.bpf.c # eBPF program loaded in the kernel
│   ├── injection.bpf.h # eBPF program header
│   └── main.go # eBPF program loaded in the user space
└── includes # Generic C header for eBPF program
    ├── aarch64
    │   └── vmlinux.h # Header to extend eBPF C program portability
    ├── amd64
    │   └── vmlinux.h # Header to extend eBPF C program portability
    └── bpf_common.h # Common eBPF header with all necessary includes to facilitate the creation of an eBPF program
```

**Note:** eBPF binaries are now built automatically during the injector Docker image build process. The injector Dockerfile uses a multi-stage build that:
1. Compiles eBPF programs in an eBPF builder stage (ubuntu:24.04 with clang, libbpf, llvm)
2. Compiles the Go injector binary in a Go builder stage
3. Assembles the final image with both eBPF and Go binaries at `/usr/local/bin/`

## How to create an eBPF program?

> **Step 1** - Copy an existing eBPF disruption

```bash
# Step 1 - clone the project and move to the root project
git clone git@github.com:DataDog/chaos-controller.git
cd chaos-controller
# Step 2 - Create the my-failure project
cp -r ebpf/disk-failure ebpf/my-failure
```

Modify the parameter of `bpf.NewModuleFromFile` method from `ebpf/my-failure/main.go`:

**Before**

```go
...
bpfModule, err := bpf.NewModuleFromFile("/usr/local/bin/bpf-disk-failure.bpf.o")
...
```

**After**

```go
...
bpfModule, err := bpf.NewModuleFromFile("/usr/local/bin/bpf-my-failure.bpf.o")
...
```

> **Step 2** - Build the injector Docker image

The eBPF programs are now built automatically as part of the injector Docker image build:

```bash
# Build the injector image (includes automatic eBPF compilation)
make docker-build-only-injector
```

This single command:
- Installs eBPF build dependencies (clang, libbpf, llvm)
- Compiles all eBPF programs in the `ebpf/` directory
- Builds the Go injector binary
- Creates the final Docker image with everything included

No separate eBPF build step is needed anymore!

> **Step 3** - Test the eBPF program

To test your eBPF program:

**Option A: Test within Docker (Recommended)**

```bash
# Build the injector image (includes automatic eBPF compilation)
make docker-build-only-injector
```

**Option B: Manual build (Linux only, for local development/testing)**

Note: This option requires a Linux system with kernel BTF support.

```bash
# Step 1 - Navigate to ebpf directory
cd ebpf

# Step 2 - Build eBPF programs locally (Linux only)
make all
```

> **Last step** - Create the injector

Create the `my_failure` injector which calls the `bpf-my-failure` program located in `/usr/local/bin/`.
You can refer to the `injector/disk_failure.go` injector.

> Before

```go
const EBPFDiskFailureCmd = "bpf-disk-failure"
...
execCmd := exec.Command(EBPFDiskFailureCmd, commandPath...)
```

> After

```go
const EBPFMyFailureCmd = "bpf-my-failure"
...
execCmd := exec.Command(EBPFMyFailureCmd, commandPath...)
```

When the `my_failure` injector is created you will be able to create a disruption with `kubectl`.
Example for `my_failure`:

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: my-failure
  namespace: chaos-demo
spec:
  level: pod
  selector:
    service: demo-curl
  count: 1
  myFailure:
    lorem: ipsum
```

## How to maintain an eBPF program?

An eBPF program can be a challenge to maintain across different Linux kernel versions. I highly recommend reading this article [BPF portability and CO-RE](https://nakryiko.com/posts/bpf-portability-and-co-re/) to understand how to maintain an eBPF program with great portability. It explains what is BTF, the purpose of the `vmlinux.h` header and `libbpf`.

`libbpf` has an important rule for the portability of eBPF programs. It provides a limited set of "stable interfaces" that eBPF programs can rely on to be stable between kernels. In reality, underlying structures and mechanisms do change, but these BPF-provided stable interfaces abstract such details from user programs.

The `vmlinux.h` avoids a lot of includes related to the kernels in your eBPF program with a single include. It is used to define the various data structures, functions and macros needed by the kernel to run on different architectures.
The `vmlinux.h` is generated via the `bpftool`. To generate it you need a Linux kernel with BTF installed. Then, you can generate the header file with the following command line:
```bash
bpftool btf dump file /sys/kernel/btf/vmlinux format c > vmlinux.h
```

### When to generate the `vmlinux.h`?

Most of the time it is not needed to generate the `vmlinux.h` file because it just defines data structures.
If you create an eBPF program and the `vmlinux.h` does not provide a struct or a function you need, then in this case you have to rebuild the `vmlinux.h`.
Otherwise, there is no need to rebuild it each time you build an eBPF program. The main role of portability is handled by the `libbpf`.
