# Network disruption

The `network` field provides an automated way of adding disruptions to the outgoing network traffic:

* `drop` drops a percentage of the outgoing traffic to simulate packets loss
* `corrupt` corrupts a percentage of the outgoing traffic to simulate packets corruption
* `delay` adds the given delay (with +- jitter) to the outgoing traffic to simulate a slow network
* `bandwidthLimit` limits the outgoing traffic bandwidth to simulate a bandwidth struggle

All of them can be combined in the same disruption resource.

## Traffic flow: egress vs. ingress

The `flow` field allows you to either disrupt outgoing traffic (`egress`) or incoming traffic (`ingress`).

### How is it different?

If you're not sure which one you should use, here is a concrete example to let you know how does it work.

Let's say you have 3 pods:
* `server`: an nginx pod listening on 80
* `client1`: a pod hitting nginx on port 80
* `client2`: another pod hitting nginx on port 80

Now let's explore the different use cases.

<kbd> <img src="../docs/img/network_flow_cases.png" height=150 width=600 align="center" /> </kbd>

#### Case 1: I want to disrupt `client1` without impacting `client2`

In this case, you want to target the `client1` pod only and use the `egress` flow so you target packets going from the `client1` pod to the `server` pod.

#### Case 2: I want to disrupt all clients

In this case, you want to target the `server` pod and use the `ingress` flow so you target all incoming packets from both `client1` and `client2` pods.

#### A note on `ingress` flow implementation and UDP

*TL;DR: the `ingress` flow only works for TCP and on ports and protocol, not hosts*

<kbd> <img src="../docs/img/network_flow_ingress.png" height=80 width=300 align="right" /> </kbd>

The current implementation of the `ingress` flow is not a real filter on incoming packets but rather a filter on incoming packets answers (ie. outgoing packets). During a TCP communication, when the client sends a packet to the server, the server answers with an acknowledgement packet to confirm that it received the client's packet. By disrupting this acknowledgement packet, it simulates an ingress disruption. It means that `ingress` flow only works for TCP (or if the server uses UDP to send back an answer to the client).

## Implementation

To apply those disruptions, the `tc` utility is used and the behavior is different according to the use cases.

### No host is specified

If no host is specified in the network disruption, the disruptions will be applied to all the outgoing traffic on all interfaces for pods which pass the requirements of the label selector aside from those necessary for healthchecks or communications with Kubernetes. It means that the qdiscs will be created directly from each interface `root` parent and chained together.

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
