# Disk failure

The `diskFailure` field offers a way to apply a disk failure on a specific process and path.

## How it works

The disruption run an eBPF disk failure program used to intercept system calls and inject errors. 
It is used to prevent certain processes from accessing certain files. 
It defines a target process and a filter path, and if the process is trying to 
open a file that matches the filter path, an -ENOENT error will be injected, 
preventing the process from opening the file.
The source code of the eBPF disk failure program is [here](../ebpf/disk-failure). 

### Notes

:warning: If you target a node with a `"/"` filter path, the disruption will catch all openat syscalls of the node.
If it's the case your last chance is to restart the node manually because you will not be able to connect remotely to the 
targeted node or do any command line.

### Known issues

Because an eBPF program has a limited memory and you cannot do dynamic loop, the filter path could not exceed `62` characters 

Be sure to have a kernel build with eBPF:

```shell
CONFIG_BPF=y
CONFIG_HAVE_EBPF_JIT=y
CONFIG_ARCH_WANT_DEFAULT_BPF_JIT=y
CONFIG_BPF_SYSCALL=y
CONFIG_BPF_JIT=y
CONFIG_BPF_JIT_ALWAYS_ON=y
CONFIG_BPF_JIT_DEFAULT_ON=y
CONFIG_BPF_UNPRIV_DEFAULT_OFF=y
CONFIG_BPF_LSM=y
CONFIG_CGROUP_BPF=y
CONFIG_IPV6_SEG6_BPF=y
CONFIG_NETFILTER_XT_MATCH_BPF=m
CONFIG_BPFILTER=y
CONFIG_BPFILTER_UMH=m
CONFIG_NET_CLS_BPF=m
CONFIG_NET_ACT_BPF=m
CONFIG_BPF_STREAM_PARSER=y
CONFIG_LWTUNNEL_BPF=y
CONFIG_BPF_EVENTS=y
CONFIG_BPF_KPROBE_OVERRIDE=y
CONFIG_TEST_BPF=m
```
