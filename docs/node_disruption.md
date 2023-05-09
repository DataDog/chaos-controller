# Node failure

The `nodeFailure` field triggers a kernel panic on the node. Because the node will reboot suddenly, the pods running on it (the injection pod included) won't see their status updated for a while. This is why the injection pod can appear as `Running` or `Unknown` while it has currently finished the injection. The kernel panic is triggered by using `sysrq`. The injector uses the following mounts to use it:

* `/proc/sys/kernel/sysrq` > `/mnt/sysrq`
* `/proc/sysrq-trigger` > `/mnt/sysrq-trigger`

> :warning:️ Node behavior when using this disruption can differ depending on the cloud provider (node may or may not be replaced, restarted, cordoned, etc.).

## Targeting

For clarity purpose, it is mandatory to explicitly set the disruption's `level` field to either `pod` or `node`.

The `nodeFailure` disruption acts on a node. However, the `level` field needs to be adapted for the selector:

- if `level: node` is set, the selector will target nodes and impact those nodes directly.
- if `level: pod` is set, the selector will targets pods and impact the nodes that host them.
