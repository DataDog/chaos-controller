# Network Bandwidth Limitation

The `networkLimitation` field allows you to set an artificial limit on the amount of input/output network bandwidth available to a running container, and see how it performs in a constrained (but not totally unavailable) network environment.

The injector will use `tc` to create a new `qdisc` that has a more constrained bandwidth limit than the default one, as if running this command (limits are configurable):

```
tc qdisc add dev eth0 root tbf rate 0.5mbit burst 5kb latency 0ms
```

On cleaning, it removes all the injected rules by clearing the dedicated chain and by removing it.

## Kernel modules

The injector needs some kernel modules to be enabled to be able to run:

* `sch_tbf` for the tc network rate limiting
* `sch_netem` for the tc network emulator module
* `sch_prio` for the tc prio qdisc creation

## See Also

* [tc rate limiting usage](https://linux.die.net/man/8/tc-tbf)
