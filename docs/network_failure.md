# Network failure

The `networkFailure` field provides an automated way of dropping the connection between a pod and a service. Please note that the connection is dropped when outgoing from the pod you targeted. It means you can prevent the targeted pod from querying an API but not from being queried. However, if the call to query to targeted pod is using TCP, the SYN-ACK answer to establish the connection will never be sent and the result will be quite the same.

The injector injects iptables rules in a dedicated iptables chain. The chain is created during the injection and has a unique name formed with the `CHAOS-` prefix and with a part of the `Disruption` Kubernetes resource UUID. All iptables injection are done in the `filter` table and during the `OUTPUT` step.

On cleaning, it removes all the injected rules by clearing the dedicated chain and by removing it.
