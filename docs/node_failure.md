# Node failure ([example](../config/samples/node_failure.yaml))

The `nodeFailure` field triggers a kernel panic on the node. Because the node will reboot suddenly, the pods running on it (the injection pod included) wont' see their status updated for a while. This is why the injection pod can appear as `Running` or `Unknown` while it has currently finished the injection. The kernel panic is triggered by using `sysrq`. The injector uses the following mounts to use it:

* `/proc/sys/kernel/sysrq` > `/mnt/sysrq`
* `/proc/sysrq-trigger` > `/mnt/sysrq-trigger`
