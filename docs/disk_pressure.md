# Disk pressure

The `diskPressure` field offers a way to apply pressure on a specific mount path.

## Throttling

Unlike the CPU pressure, this kind of disruption is not done by stressing the disk but by throttling its capacities. A throttle can be applied on read or write operations, or both.

The throttling is done by using the [blkio cgroup controller](https://www.kernel.org/doc/Documentation/cgroup-v1/blkio-controller.txt), and more specifically by the `blkio.throttle.read_bps_device` and `blkio.throttle.write_bps_device` files.

To apply the throttle, the injector will:

* search for the device related to the given path
  * it is done by using the `df` command which will print out the device path
* retrieve the device major identifier
  * it is done by using the `ls` command on the device path (eg. `/dev/sda1`) which will print out the major and minor identifiers of the device
* write the throttle using the major identifier of the device

### Notes

The throttle will be applied to the whole device (for the pod only) and not only to the partition handling the path.

When running a disk disruption on a pod with multiple containers, if a specified path to disrupt does not exist on any of the existing containers, those containers will be skipped and not disrupted.

### Known issues

TL;DR: the limit will only applies on direct read and write operations (using the `O_DIRECT` flag) because of cgroups v1. We can't use cgroups v2 for now because [containerd doesn't support it yet](https://github.com/opencontainers/runc/issues/2315).

Most of the time, when writing a file to the disk, data are first written to kernel page cache (in memory) and then flushed to the disk. Because controllers are totally independent in cgroups v1, the limit will never be applied on page flush. So what does it mean? Most of the applications won't be throttled because they don't use direct read or write operations. We are currently working on a way to improve the behavior of the throttle until we can switch to cgroups v2.

More information can be found on [this blog post](https://medium.com/some-tldrs/tldr-using-cgroups-to-limit-i-o-by-andr%C3%A9-carvalho-421bb1d855e) about this limitation.

## Manual cleanup instructions

:information_source: All those commands must be executed on the infected host (except for `kubectl`).

---

:warning: If the disruption is injected at the pod level, you must find the related cgroups path **for each container**.

* Identify the container IDs of your pod

```
kubectl get -ojson pod demo-curl-547bb9c686-57484 | jq '.status.containerStatuses[].containerID'
"containerd://cb33d4ce77f7396851196043a56e625f38429720cd5d3153cb061feae6038460"
"containerd://629c7da02cbcf77c6b7131a59f5be50579d9e374433a444210b6547186dd5f0d"
```

* Identify cgroups path

```
# crictl inspect cb33d4ce77f7396851196043a56e625f38429720cd5d3153cb061feae6038460 | grep cgroupsPath
        "cgroupsPath": "/kubepods/burstable/poda37541dc-4905-4a7f-98c0-7d13f58df0eb/cb33d4ce77f7396851196043a56e625f38429720cd5d3153cb061feae6038460",
```

---

* Identify blkio device to reset for read and write (depending on the applied disk pressure)

```
# cat /sys/fs/cgroup/blkio/kubepods/burstable/poda37541dc-4905-4a7f-98c0-7d13f58df0eb/cb33d4ce77f7396851196043a56e625f38429720cd5d3153cb061feae6038460/blkio.throttle.read_bps_device
8:0 1073741824
# cat /sys/fs/cgroup/blkio/kubepods/burstable/poda37541dc-4905-4a7f-98c0-7d13f58df0eb/cb33d4ce77f7396851196043a56e625f38429720cd5d3153cb061feae6038460/blkio.throttle.write_bps_device
8:0 1073741824
```

*Throttle files are located directly in `/sys/fs/cgroup/blkio` if the disruption is applied at the node level.*

* Reset throttle values for the found device

```
# echo "8:0 0" > /sys/fs/cgroup/blkio/kubepods/burstable/poda37541dc-4905-4a7f-98c0-7d13f58df0eb/cb33d4ce77f7396851196043a56e625f38429720cd5d3153cb061feae6038460/blkio.throttle.read_bps_device
# echo "8:0 0" > /sys/fs/cgroup/blkio/kubepods/burstable/poda37541dc-4905-4a7f-98c0-7d13f58df0eb/cb33d4ce77f7396851196043a56e625f38429720cd5d3153cb061feae6038460/blkio.throttle.write_bps_device
```

* Ensure that the values are reset

```
# cat /sys/fs/cgroup/blkio/kubepods/burstable/poda37541dc-4905-4a7f-98c0-7d13f58df0eb/cb33d4ce77f7396851196043a56e625f38429720cd5d3153cb061feae6038460/blkio.throttle.read_bps_device
# cat /sys/fs/cgroup/blkio/kubepods/burstable/poda37541dc-4905-4a7f-98c0-7d13f58df0eb/cb33d4ce77f7396851196043a56e625f38429720cd5d3153cb061feae6038460/blkio.throttle.write_bps_device
```
