# Chaos Controller

Datadog runs regular chaos experiments to test the resilience of our distributed cloud applications hosted in Kubernetes. The Chaos Controller facilitates automation of these experiments by simulating common "disruptions" including but not limited to: poor network quality, exhaustion of computational resources, or unexpected node failures.

To get started, a chaos engineer simply needs to define a `yaml` file which contains all of the specifications needed by our custom Kubernetes Resource to run the preferred disruption `Kind`.

## Kubernetes 1.20.x known issues

**Latest Kubernetes version supported: 1.19.x ([more info](#kubernetes-120x-known-issues))**
[The following issue](https://github.com/kubernetes/kubernetes/issues/97288) prevents the controller from running properly on Kubernetes 1.20.x. We don't plan to support this version. [The fix](https://github.com/kubernetes/kubernetes/pull/97980) should be released with Kubernetes 1.21.

## Disclaimer

The **Chaos Controller** allows you to disrupt your Kubernetes infrastructure through various means including but not limited to: bringing down resources you have provisioned and preventing critical data from being transmitted between resources. The use of **Chaos Controller** on your production system is done at your own discretion and risk.

## Table of Contents

* [Getting Started](#getting-started)
* [Examples](#examples)
* [Installation](#installation)
* [Design](docs/design.md)
* [Metrics](docs/metrics.md)
* [FAQ](docs/faq.md)
* [Contributing](#contributing)

## Getting Started

Disruptions are built as short-living resources which should be manually created and removed once your experiments are done. They should not be part of any application deployment. Getting started is as simple as creating a Kubernetes resource using `kubectl apply -f <disruption_file.yaml>`, and clean up would be `kubectl delete -f <disruption_file>.yaml`. For your safety, we recommend you get started with the `dry-run` mode enabled.

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

Please note that the `Disruption` resource is **immutable**. Once applied, editing it will have no effect. If you need to change the disruption definition, you need to delete the existing resource and to re-create it.

Below is a list of common options to get started. Visit the [usage guide](docs/usage.yaml) for more customizations and sample use cases!

### Dry-run mode

 This fakes the injection while still going through the process of selecting targets, creating chaos pods, and simulating the disruption as much as possible. Put a different way, all "read" operations (like knowing which network interface to disrupt) will be executed while all "write" operations won't be (like creating what's needed to drop packets). Checkout this [example](config/samples/dry_run.yaml).

### Level

A disruption can be applied either at the `pod` level or at the `node` level:

* When applied at the `pod` level, the controller will target pods and will affect only the targeted pods. Other pods running on the same node as those targeted may still be affected depending on the injected disruption.
* When applied at the `node` level, the controller will target nodes and will potentially affect everything running on the node (other processes).

### Targeting

The `Disruption` resource uses [label selectors](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) to target pods and nodes. The controller will retrieve all pods or nodes matching the given label selector and will randomly select a number (defined in the `count` field) of matching targets. Once applied, you can see the targeted pods/nodes by describing the `Disruption` resource.

**NOTE:** If you are targeting pods, the disruption must be created in the same namespace as the targeted pods.

## Installation

Please read the [installation guide](docs/installation.md) for instructions on deploying Chaos Controller to your Kubernetes cluster.

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) to develop chaos-controller in your local Minikube environment!
