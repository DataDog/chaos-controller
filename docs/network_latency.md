# Network latency ([example](../config/samples/network_latency.yaml))

The `networkLatency` field provides a way to add latency on pods. It works with the `tc` command to apply the given delay which replaces the related network interface root qdisc by a custom one.

## With no hosts specified

If no hosts are specified, the injector creates a `netem` qdisc in place of the interface root qdisc and applies the delay on all the outgoing traffic.

## With one or multiple hosts specified

The injector first resolves the given hosts to get a set of IP to filter on.

Once done, it creates a `prio` qdisc in place of the interface root qdisc. This qdisc has the specificity of having 4 bands (instead of 3 by default) but keeps the default `prio map`, which means that the traffic will be sent over the 3 first bands only.

It then adds a child class to the `prio` qdisc to add the given delay to the 4th band.

It finally filters the traffic to the given hosts to redirect it through the 4th band where the delay will be applied. By doing this, the delay only applies to the traffic going to the given hosts and not to all the outgoing traffic.

Please note that to create a `prio` qdisc on a virtual interface (such as a docker interface), this interface must have a set queue length (`qlen`), otherwise the traffic will be dropped. Container interfaces don't have any `qlen` set by default. The injector sets up the interface queue length if it's not already the case and clears it right after the delay has been injected.

## Kernel modules

The injector needs some kernel modules to be enabled to be able to run:

* `sch_netem` for the tc network emulator module
* `sch_prio` for the tc prio qdisc creation

## More information

* [tc qdisc prio](https://linux.die.net/man/8/tc-prio)
* [tc qdisc netem](http://man7.org/linux/man-pages/man8/tc-netem.8.html)
* [tc filter](http://man7.org/linux/man-pages/man8/tc-u32.8.html)
* [applying delay to a single IP](https://serverfault.com/questions/389290/using-tc-to-delay-packets-to-only-a-single-ip-address)
* [creating a prio qdisc on a virtual interface](https://github.com/moby/moby/issues/33162#issuecomment-306424194)
