# Examples

The repository contains [an examples folder](../examples/) with disruptions files that you can customize and use. It has [a full example of the disruption resource](../examples/complete.yaml) with comments.

You can also have a look at the following use cases with examples of disruptions you can adapt and apply as you wish:

- General options
  - [I want to simulate a flapping failure (injecting and cleaning continuously)](../examples/pulse.yaml)
  - [I want to notify (eg. Slack) on a specific disruption injection](../examples/reporting_network_drop.yaml)
  - [I want my disruption to expire automatically after some time](../examples/timed_disruption.yaml)
  - [I want the injection to start on all targets simultaneously](../examples/triggers.yaml)
- Targeting options
  - [I want to select my targets with label selector operators (advanced selector)](../examples/advanced_selector.yaml)
  - [I want to select my targets based on annotations in addition to the label selector](../examples/annotation_filter.yaml)
  - [I want to target one or some containers of my pod only, not all of them](../examples/containers_targeting.yaml)
  - [I want to disrupt network packets on pod initialization](../examples/on_init.yaml)
  - [I want to select a fixed set of targets (static targeting)](../examples/static_targeting.yaml)
- [Node disruptions](/docs/node_disruption.md)
  - [I want to randomly kill one of my node](../examples/node_failure.yaml)
  - [I want to randomly kill one of my node and keep it down](../examples/node_failure_shutdown.yaml)
- [Pod disruptions](/docs/container_disruption.md)
  - [I want to terminate all the containers of one of my pods gracefully](../examples/container_failure_all_graceful.yaml)
  - [I want to terminate all the containers of one of my pods non-gracefully](../examples/container_failure_all_forced.yaml)
  - [I want to terminate a container of one of my pods gracefully](../examples/container_failure_graceful.yaml)
  - [I want to terminate a container of one of my pods non-gracefully](../examples/container_failure_forced.yaml)
- [Network disruptions](/docs/network_disruption.md)
  - [I want to drop packets going out from my pods](../examples/network_drop.yaml)
  - [I want to corrupt packets going out from my pods](../examples/network_corrupt.yaml)
  - [I want to add network latency to packets going out from my pods](../examples/network_delay.yaml)
  - [I want to restrict the outgoing bandwidth of my pods](../examples/network_bandwidth_limitation.yaml)
  - [I want to disrupt packets going to a specific host, port or Kubernetes service](../examples/network_filters.yaml)
  - [I want to disrupt packets going to a specific cloud managed service](../examples/network_cloud.yaml)
- [CPU pressure](/docs/cpu_pressure.md)
  - [I want to put CPU pressure against my pods](../examples/cpu_pressure.yaml)
- [Disk pressure](/docs/disk_pressure.md)
  - [I want to throttle my pods disk reads](../examples/disk_pressure_read.yaml)
  - [I want to throttle my pods disk writes](../examples/disk_pressure_write.yaml)
- [DNS resolution mocking](/docs/dns_disruption.md)
  - [I want to fake my pods DNS resolutions](../examples/dns.yaml)
