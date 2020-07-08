# Disk pressure ([example](../config/samples/disk_pressure.yaml))

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
