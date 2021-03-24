# Chaos Controller

**Latest Kubernetes version supported: 1.19.x ([more info](#kubernetes-120x-known-issues))**

The Chaos Controller was created to facilitate automation requirements in Datadog chaos experiments. It helps to deal with failures during chaos engineering events by abstracting them, especially when dealing with big deployments or complex network operations. It introduces a custom Kubernetes resource named `Disruption`.

## Table of Contents

* [Usage](#usage)
* [Examples](#examples)
* [Installation](#installation)
* [Design](docs/design.md)
* [Metrics](docs/metrics.md)
* [FAQ](docs/faq.md)
* [Contributing](#contributing)

## Usage

Disrupting your system by generating some failures is as simple as creating a Kubernetes resource, and removing those failures is as simple as deleting the resource!

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: node-failure
  namespace: chaos-engineering
spec:
  selector: # a label selector used to target some resources
    app: demo-curl
  count: 1 # the number of resources to target
  nodeFailure:
    shutdown: false # trigger a kernel panic on the target node
```

*Do not hesitate to apply disruptions with the dry-run mode enabled to safely try some disruptions!*

**Disruptions are built as short-living resources which should be manually created and removed once your experiments are done. They should not be part of any application deployment.**

### Dry-run mode

You can enable the dry-run mode on any disruption to fake the injection. The dry-run mode will still select targets, create chaos pods and simulate the disruption as much as possible. It means that all "read" operations (like knowing which network interface to disrupt) will be executed while all "write" operations won't be (like creating what's needed to drop packets).

[Please look at the following example for how to do.](config/samples/dry_run.yaml).

### Targeting

The `Disruption` resource uses [label selectors](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) to target pods and nodes. The controller will retrieve all pods or nodes matching the given label selector and will randomly select as many matching targets as defined in the `count` field.

Once applied, you can see the targeted pods/nodes by describing the `Disruption` resource.

#### Targeting a specific pod

How can you target a specific pod by name, if it doesn't have a unique label selector you can use? The `Disruption` spec doesn't support field selectors at this time, so selecting by name isn't possible. However, you can use the `kubectl label pods` command, e.g., `kubectl label pods $podname unique-label-for-this-disruption=target-me` to dynamically add a unique label to the pod, which you can use as your label selector in the `Disruption` spec.

### Level

A disruption can be applied either at the `pod` level or at the `node` level:

* When applied at the `pod` level, the controller will target pods and will affect only the targeted pods. Other pods running on the same node as those targeted may still be affected depending on the injected disruption.
* When applied at the `node` level, the controller will target nodes and will potentially affect everything running on the node (other processes).

#### Example

Let's imagine a node with two pods running: `foo` and `bar` and a disruption dropping all outgoing network packets:

* Applying this disruption at the `pod` level and with a selector targeting the `foo` pod will result with the `foo` pod not being able to send any packets, but the `bar` pod will still be able to send packets, as well as other processes on the node.
* Applying this disruption at the `node` level and with a selector targeting the node itself, both `foo` and `bar` pods won't be able to send network packets anymore, as well as all the other processes running on the node.

### Containers targeting

*It only applies to disruption applied at the `pod` level.*

A disruption affects all containers within the pod by default. You can restrict the scope of the disruption to a single container or to only some containers.

[Please look at the following example for how to do.](config/samples/containers_targeting.yaml).

### A quick note on immutability

The `Disruption` resource is immutable. Once applied, editing it will have no effect. If you need to change the disruption definition, you need to delete the existing resource and to re-create it.

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

## Installation

**Note: it only applies to people outside of Datadog.**

To deploy it on your cluster, you just have to run the `make install` command and it will create the CRD for the `Disruption` kind and apply the needed manifests to create the controller deployment.

You can uninstall it the same way, by using the `make uninstall` command.

### Injector pods extra annotations

The injector pods spec is generated by the controller itself. You can add custom annotations to it by providing the `--injector-annotations` flag to the controller. For instance:

```
--injector-annotations "my-annotation.my-workspace.io/foo=bar" --injector-annotations "my-annotation.my-workspace.io/bar=baz"
```

### Delete-only mode

This flag can be enabled specifically on the controller configuration itself (through the arguments of it's container). Once enabled, the controller in question will reject any incoming requests to create new injections for new disruptions. In this state, the controller will only accept requests to clean/remove disruptions. The controller must be restarted with the corresponding `--delete-only` argument in order to reach this state.

## Contributing

Please read the [contributing documentation](CONTRIBUTING.md) for more information.

## Kubernetes 1.20.x known issues

[The following issue](https://github.com/kubernetes/kubernetes/issues/97288) prevents the controller from running properly on Kubernetes 1.20.x. We don't plan to support this version. [The fix](https://github.com/kubernetes/kubernetes/pull/97980) should be released with Kubernetes 1.21.
