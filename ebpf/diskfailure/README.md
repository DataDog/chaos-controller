# bpf-diskfailure
This eBPF program catchs openat syscalls and replace its return by -ENOENT. The goal is to fake disk failures. This library is based on in Golang using [libbpfgo](http://github.com/aquasecurity/libbpfgo). 

## Install Go 

See [the Go documentation](https://golang.org/doc/install)

## Install packages

```sh
sudo apt-get update
sudo apt-get -y install libbpf-dev pahole make clang llvm libelf-dev
# Only for amd64
sudo apt-get -y install libc6-dev-i386
```

## Build and run

```sh
make all
sudo ./bpf-diskfailure-(arm64|x86) -p <your-pid>
```

Output:
* bpf-diskfailure.bpf.o - an object file for the eBPF program
* bpf-diskfailure - a Go executable

The Go executable reads in the object file at runtime. Take a look at the .o file with readelf if you want to see the sections defined in it.

## Docker

To avoid compatibility issues, you can use the `Dockerfile` provided in this repository.

Build it by your own:

```bash
# ARM
nerdctl build --build-arg ARCH=arm64 --platform linux/arm64 -t build-bpf-diskfailure:lunar-arm64 .
# AMD
nerdctl build --build-arg ARCH=amd64 --platform linux/amd64 -t build-bpf-diskfailure:lunar-amd64 .
```

And then run it from the project directory to compile the program:

```bash
# ARM
docker run --rm -v $(pwd)/:/app/:z build-bpf-diskfailure:lunar-arm64
# Output:
bpf-diskfailure-arm64 # Go binary
bpf-diskfailure-arm64.bpf.o # C binary
# AMD
docker run --rm -v $(pwd)/:/app/:z build-bpf-diskfailure:lunar-amd64
# Output:
bpf-diskfailure-x86 # Go binary
bpf-diskfailure-x86.bpf.o # C binary

```

## Notes 

I'm using Ubuntu 23.04, kernel 5.15, go 1.19

This approach installs the libbpf-dev package. Another alternative (which is what [Tracee](https://github.com/aquasecurity/tracee) does) is to install the [libbpf source](https://github.com/libbpf/libbpf) as a git submodule, build it from source and install it to the expected location (e.g. `/usr/lib/x86_64-linux-gnu/libbpf.a` on an Intel x86 processor).
