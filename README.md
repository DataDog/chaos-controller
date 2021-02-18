# Chaos controller

The controller was created to facilitate automation requirements in Datadog chaos workflows and pipelines. It helps to deal with failures during gamedays by abstracting them, especially when dealing with big deployments or complex network operations.

The `controller` is deployed as a `Deployment`. It watches for changes on the `Disruption` CRD, as well as their child resources.

## Table of Contents

* [Usage](#usage)
* [Controller Installation](#controller-installation)
* [Design](docs/design.md)
* [Metrics](docs/metrics.md)
* [FAQ](docs/faq.md)
* [Contributing](#contributing)

## Usage

The controller works with a custom Kubernetes resource named `Disruption` describing the wanted failures and the pods/nodes to target. By creating this resource in the namespace of the pods (no matter the namespace for nodes) you want to affect, it'll create pods to inject the needed failures. On `Disruption` resource delete, those failures will be cleaned up by those same pods.

*Do not hesitate to apply disruptions with the dry-run mode enabled to do your tests!*

### Dry-run mode

First of all, you can enable the dry-run mode on any disruption to fake the injection if you are not sure about what you're doing. The dry-run mode will still select targets, create chaos pods and simulate the disruption as much as possible. It means that all "read" operations (like knowing which network interface to disrupt) will be executed while all "write" operations won't be (like creating what's needed to drop packets).

It can be enabled by adding the `dryRun: true` field to the disruption spec. Please look at [the complete example](config/samples/complete.yaml) for more information.

### Level

A disruption can be applied either at the `pod` level or at the `node` level:

* When applied at the `pod` level, the controller will target pods and will affect only the targeted pods. Other pods running on the same node as those targeted should not be affected (there is a potential blast radius depending on the injected disruption of course).
* When applied at the `node` level, the controller will target nodes and will potentially affect everything running on the node (other containers and processes).

#### Example

Let's imagine a node with two pods running: `foo` and `bar` and a disruption dropping all outgoing network packets:

* Applying this disruption at the `pod` level and with a selector targeting the `foo` pod will result with the `foo` pod not being able to send any packets, but the `bar` pod will still be able to send packets, as well as other processes on the node.
* Applying this disruption at the `node` level and with a selector targeting the node itself, both `foo` and `bar` pods won't be able to send network packets anymore, as well as all the other processes running on the node.

### Targeting

The `Disruption` custom resource helps you to target the pods/nodes you want to be affected by the failures. This is done by a [label selector](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/). This selector will find all the pods/nodes matching the specified labels in the `Disruption` resource namespace and will affect either all of them or some of them *randomly* depending on the `count` value specified in the resource. For those who have pods with multiple containers and want to target specific containers, the `containers` array can be used to identify which containers (by name) to target within the pod. By default all containers are targeted. If any specified target container is not found in the container list for all targeted pod (e.x. a typo), the disruption will fail.

Depending on the [disruption level](#level), the selector will be applied to pods or nodes.

Once applied, you can see the targeted pods/nodes by describing the `Disruption` resource.

### Use cases

Please take a look at the different disruptions documentation linked in the table of content for more information about what they can do and how to use them.

Here is [a full example of the disruption resource](config/samples/complete.yaml) with comments. You can also have a look at the following use cases with examples of disruptions you can adapt and apply as you wish:

* [Node disruptions](docs/node_disruption.md)
  * [I want to randomly kill one of my node](config/samples/node_failure.yaml)
  * [I want to randomly kill one of my node and keep it down](config/samples/node_failure_shutdown.yaml)
* [Network disruptions](docs/network_disruption.md)
  * [I want to drop packets between my pods and a service](config/samples/network_disruption_drop.yaml)
  * [I want to corrupt packets between my pods and a service](config/samples/network_disruption_corrupt.yaml)
  * [I want to add network latency to packets between my pods and a service](config/samples/network_disruption_latency.yaml)
  * [I want to restrict the bandwidth between my pods and a service](config/samples/network_disruption_bandwidth.yaml)
* [CPU pressure](docs/cpu_pressure.md)
  * [I want to put CPU pressure against my pods](config/samples/cpu_pressure.yaml)
* [Disk pressure](docs/disk_pressure.md)
  * [I want to throttle my disk reads](config/samples/disk_pressure_read.yaml)
  * [I want to throttle my disk writes](config/samples/disk_pressure_write.yaml)

### Deploying a Disruption

If you want to get started and deploy a disruption to your service, it's important to first note that a disruption is an **ephemeral resource** -- it should be created and then deleted as soon as your test is done, and thus the YAML generally shouldn't be kept long-term (in a Helm chart for example).

To deploy a disruption, simply create a `disruption.yaml` file as done in the examples above. Then, `kubectl apply -f disruption.yaml` to create the resource in the same namespace as the targets you want to disrupt. You should be able to `kubectl get pods` and see the running disruption injector pod.

Then, when you're finished testing and want to remove the disruption, similarly run `kubectl delete -f disruption.yaml` to delete the disruption resource. The existing chaos pods should clean the disruption and exit.

### A quick note on immutability

The `Disruption` resource is immutable. Once applied, editing it will have no effect. If you need to change the disruption definition, you need to delete the existing resource and to re-create it.

## Controller Installation

**Note: it only applies to people outside of Datadog.**

To deploy it on your cluster, you just have to run the `make install` command and it will create the CRD for the `Disruption` kind and apply the needed manifests to create the controller deployment.

You can uninstall it the same way, by using the `make uninstall` command.

### Injector pods extra annotations

The injector pods spec is generated by the controller itself. You can add custom annotations to it by providing the `--injector-annotations` flag to the controller. For instance:

```
--injector-annotations "my-annotation.my-workspace.io/foo=bar" --injector-annotations "my-annotation.my-workspace.io/bar=baz"
```

## Contributing

Please read the [contributing documentation](CONTRIBUTING.md) for more information.
