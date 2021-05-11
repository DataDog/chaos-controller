# Chaos Controller Usage Guide

## Dry-run mode

 This fakes the injection while still going through the process of selecting targets, creating chaos pods, and simulating the disruption as much as possible. Put a different way, all "read" operations (like knowing which network interface to disrupt) will be executed while all "write" operations won't be (like creating what's needed to drop packets). Checkout this [example](config/samples/dry_run.yaml).

## Level

A disruption can be applied either at the `pod` level or at the `node` level:

* When applied at the `pod` level, the controller will target pods and will affect only the targeted pods. Other pods running on the same node as those targeted may still be affected depending on the injected disruption.
* When applied at the `node` level, the controller will target nodes and will potentially affect everything running on the node (other processes).

### Example

Let's imagine a node with two pods running: `foo` and `bar` and a disruption dropping all outgoing network packets:

* Applying this disruption at the `pod` level and with a selector targeting the `foo` pod will result with the `foo` pod not being able to send any packets, but the `bar` pod will still be able to send packets, as well as other processes on the node.
* Applying this disruption at the `node` level and with a selector targeting the node itself, both `foo` and `bar` pods won't be able to send network packets anymore, as well as all the other processes running on the node.

## Targeting

The `Disruption` resource uses [label selectors](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) to target pods and nodes. The controller will retrieve all pods or nodes matching the given label selector and will randomly select a number (defined in the `count` field) of matching targets. It's possible to specify multiple label selectors, in which case the controller will select from targets that match all of them. Once applied, you can see the targeted pods/nodes by describing the `Disruption` resource.

**NOTE:** If you are targeting pods, the disruption must be created in the same namespace as the targeted pods.

### Targeting a specific pod

How can you target a specific pod by name, if it doesn't have a unique label selector you can use? The `Disruption` spec doesn't support field selectors at this time, so selecting by name isn't possible. However, you can use the `kubectl label pods` command, e.g., `kubectl label pods $podname unique-label-for-this-disruption=target-me` to dynamically add a unique label to the pod, which you can use as your label selector in the `Disruption` spec.

### Targeting a specific container within a pod

By default, a disruption affects all containers within the pod. You can restrict the scope of the disruption to a single container or to only some containers [like this](config/samples/containers_targeting.yaml).

## Examples

Please take a look at the different disruptions documentation linked in the table of content for more information about what they can do and how to use them.

Here is [a full example of the disruption resource](config/samples/complete.yaml) with comments. You can also have a look at the following use cases with examples of disruptions you can adapt and apply as you wish:

* [Node disruptions](docs/node_disruption.md)
  * [I want to randomly kill one of my node](config/samples/node_failure.yaml)
  * [I want to randomly kill one of my node and keep it down](config/samples/node_failure_shutdown.yaml)
* [Network disruptions](docs/network_disruption.md)
  * [I want to drop packets going out from my pods](config/samples/network_drop.yaml)
  * [I want to corrupt packets going out from my pods](config/samples/network_corrupt.yaml)
  * [I want to add network latency to packets going out from my pods](config/samples/network_delay.yaml)
  * [I want to restrict the outgoing bandwidth of my pods](config/samples/network_bandwidth_limitation.yaml)
  * [I want to disrupt packets going to a specific port or host](config/samples/network_filters.yaml)
* [CPU pressure](docs/cpu_pressure.md)
  * [I want to put CPU pressure against my pods](config/samples/cpu_pressure.yaml)
* [Disk pressure](docs/disk_pressure.md)
  * [I want to throttle my pods disk reads](config/samples/disk_pressure_read.yaml)
  * [I want to throttle my pods disk writes](config/samples/disk_pressure_write.yaml)
* [DNS resolution mocking](docs/dns_disruption.md)
  * [I want to fake my pods DNS resolutions](config/samples/dns.yaml)