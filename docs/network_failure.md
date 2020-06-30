# Network failure ([example](../config/samples/network_failure.yaml))

The `networkFailure` field provides an automated way of dropping connections and corruption packets between a pod and a service. Please note that for the connection drop failure, the connection is only dropped when outgoing from the pod you targeted. For dropping connections, it means you can prevent the targeted pod from querying an API but not from being queried. However, if the call to query to targeted pod is using TCP, the SYN-ACK answer to establish the connection will never be sent and the result will be quite the same. The packet corruption also leads to some interesting test cases where some random portion of the packet will be corrupted.

For information regarding tc, which we use to apply these disruption, please take a look at the [network](network.md) docs.

## Kernel modules

The injector needs some kernel modules to be enabled to be able to run:

* `sch_netem` for the tc network emulator module
* `sch_prio` for the tc prio qdisc creation
