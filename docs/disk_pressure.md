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

**Note: the throttle will be applied to the whole device (for the pod only) and not only to the partition handling the path.**

### Container Filtering

There may be the scenario where a user wants to run a disk disruption but only on specific containers which have specific volume's set up, leaving the other containers to be unaffected.
This is possible at no additional expense to the user as the disk disruption will filter out all containers which do not carry the volume's specified in the user's disk disruption configuration.

### Known issues

TL;DR: the limit will only applies on direct read and write operations (using the `O_DIRECT` flag) because of cgroups v1. We can't use cgroups v2 for now because [containerd doesn't support it yet](https://github.com/opencontainers/runc/issues/2315).

Most of the time, when writing a file to the disk, data are first written to kernel page cache (in memory) and then flushed to the disk. Because controllers are totally independent in cgroups v1, the limit will never be applied on page flush. So what does it mean? Most of the applications won't be throttled because they don't use direct read or write operations. We are currently working on a way to improve the behavior of the throttle until we can switch to cgroups v2.

More information can be found on [this blog post](https://medium.com/some-tldrs/tldr-using-cgroups-to-limit-i-o-by-andr%C3%A9-carvalho-421bb1d855e) about this limitation.
