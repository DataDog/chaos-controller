# Network disruption

The `network` field provides an automated way of adding disruptions to the outgoing network traffic:

* `drop` drops a percentage of the outgoing traffic to simulate packets loss
* `corrupt` corrupts a percentage of the outgoing traffic to simulate packets corruption
* `delay` adds the given delay (with +- jitter) to the outgoing traffic to simulate a slow network
* `bandwidthLimit` limits the outgoing traffic bandwidth to simulate a bandwidth struggle

All of them can be combined in the same disruption resource. To apply these disruptions, the `tc` utility is used and the behavior is different according to the use cases.

<p align="center"><kbd>
    <img src="../docs/img/network_prio/pfifo.png" height=200 width=650 />
</kbd></p>

By extending the default linux kernal functionality for prioritizing network traffic, the `chaos-controller` can disrupt only the packets matching criteria specified in the network disruption spec.

<p align="center">
    <kbd>
        <img src="../docs/img/network_prio/traditional_notation.png" height=250 width=400 />
    </kbd>
    <kbd>
        <img src="../docs/img/network_hosts/generic.png" height=250 width=400 />
    </kbd>
</p>

Even if you do not specify many fields, our default configurations can be effective for most use cases. However, some disruption scenarios require careful tuning of the specs to replicate. If you have specific disruption requirements such as what protocol to disrupt, flow direction, or target hosts and ports, check out the FAQ pages below to learn more about this tool!

## FAQs:

* [How do I decide my traffic flow? (Ingress vs Egress)](/docs/network_disruption_flow.md)
* [What should I specify in hosts?](/docs/network_disruption_hosts.md)
* [What are `prio` qdiscs and how does chaos-controller use them?](/docs/network_disruption_prio.md)

Still have questions? Reach out to contributors to explore out the right disruption for your team!

## Kernel modules

The injector needs some kernel modules to be enabled to be able to run:

* `sch_netem` for the `tc` network emulator module used to apply packets loss, packets corruption and delay
* `sch_tbf` for the `tc` bandwidth limitation used to apply bandwidth limitation
* `sch_prio` for the `tc` `prio` qdisc creation used to apply disruptions to some part of the traffic only