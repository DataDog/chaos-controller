# Network disruption ([example](../config/samples/network_disruption.yaml))

The `network` field provides an automated way of adding disruptions to the outgoing network traffic:

* `drop` drops a percentage of the outgoing traffic to simulate packets loss
* `corrupt` corrupts a percentage of the outgoing traffic to simulate packets corruption
* `delay` adds the given delay to the outgoing traffic to simulate a slow network
* `bandwidthLimit` limits the outgoing traffic bandwidth to simulate a bandwidth struggle

## Implementation

To apply those disruptions, the `tc` utility is used and the behavior is different according to the use cases.

### No host is specified

If no host is specified in the network disruption, the disruptions will be applied to all the outgoing traffic on all interfaces. It means that the qdiscs will be created directly from each interface `root` parent and chained together.

```
eth0
+------------+            +----------+
|parent: root|            |parent: 1 |
|handle: 1   +----------->+handle: 2 |
|qdisc: netem|            |qdisc: tbf|
+------------+            +----------+

eth1
+------------+            +----------+
|parent: root|            |parent: 1 |
|handle: 1   +----------->+handle: 2 |
|qdisc: netem|            |qdisc: tbf|
+------------+            +----------+
```

It is the most common and simplest usage of `tc`.

### One or multiple hosts are specified

If at least one host is specified in the network disruption, the injector has to deal with `tc` filters to avoid to apply the disruptions on all the traffic. In this case, a `prio` qdisc will be created from the root parent of the interface used to send traffic to the specified host(s). Other qdiscs will be chained to the `prio` qdisc. Finally, a filter will redirect the traffic going to the specified host(s) through the right `prio` qdisc band.

#### Trick of using a `prio` qdisc

The `prio` qdisc is used to define some QoS on the outgoing traffic. By default, a `prio` qdisc has 3 bands (each band being managed by a class) with a priority map spreading the traffic across those 3 bands depending on its criticality. More information about this on the official [tc-prio documentation](https://linux.die.net/man/8/tc-prio).

Because the `prio` qdisc bands are handled by classes, we can chain qdiscs to those classes. It means that we can apply delay on a specific band for instance. What we want to do is to avoid applying network disruptions to all the outgoing traffic, even for a small time, because it can lead to unexpected behavior.

To deal with this, we create a 4 bands `prio` qdisc with the default priority map (using only 3 bands). The latest band, the 4th one, is never used until we explicitly ask `tc` to use it. We can start to chain qdiscs on this band and, once everything is chained, create a filter that will send the traffic going to the specified host(s) through this band specifically.

```
eth0
+------------+            +------------+            +----------+
|parent: root|            |parent: 1:4 |            |parent: 2 |
|handle: 1   +----------->+handle: 2   +----------->+handle: 3 |
|qdisc: prio |            |qdisc: netem|            |qdisc: tbf|
+------------+            +------------+            +----------+
```

Please note the parent set to `1:4` meaning it is chained to the `prio` qdisc 4th band (once again, each band being managed by a class).

## Kernel modules

The injector needs some kernel modules to be enabled to be able to run:

* `sch_netem` for the `tc` network emulator module used to apply packets loss, packets corruption and delay
* `sch_tbf` for the `tc` bandwidth limitation used to apply bandwidth limitation
* `sch_prio` for the `tc` `prio` qdisc creation used to apply disruptions to some part of the traffic only

## More documentation about `tc`

* [tc](https://linux.die.net/man/8/tc)
* [tc-prio](https://linux.die.net/man/8/tc-prio)
* [tc-tbf](https://linux.die.net/man/8/tc-tbf)
* [tc-netem](https://man7.org/linux/man-pages/man8/tc-netem.8.html)
