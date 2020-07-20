# Network latency ([example](../config/samples/network_latency.yaml))

The `networkLatency` field provides a way to add latency on pods. It works with the `tc` command to apply the given delay which replaces the related network interface root qdisc by a custom one.

For information regarding tc, which we use to apply these disruption, please take a look at the [network](network.md) docs.

## Kernel modules

The injector needs some kernel modules to be enabled to be able to run:

* `sch_netem` for the tc network emulator module
* `sch_prio` for the tc prio qdisc creation
