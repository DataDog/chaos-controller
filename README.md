# Chaos Controller

Datadog runs regular chaos experiments to test the resilience of our distributed cloud applications hosted in Kubernetes. The Chaos Controller facilitates automation of these experiments by simulating common "disruptions" including but not limited to: poor network quality, exhaustion of computational resources, or unexpected node failures.

To get started, a chaos engineer simply needs to define a `yaml` file which contains all of the specifications needed by our custom Kubernetes Resource to run the preferred disruption `Kind`.

## Kubernetes 1.20.x known issues

**Latest Kubernetes version supported: 1.19.x**

[The following issue](https://github.com/kubernetes/kubernetes/issues/97288) prevents the controller from running properly on Kubernetes 1.20.x. We don't plan to support this version. [The fix](https://github.com/kubernetes/kubernetes/pull/97980) should be released with Kubernetes 1.21.

## Disclaimer

The **Chaos Controller** allows you to disrupt your Kubernetes infrastructure through various means including but not limited to: bringing down resources you have provisioned and preventing critical data from being transmitted between resources. The use of **Chaos Controller** on your production system is done at your own discretion and risk.

## Table of Contents

* [Getting Started](#getting-started)
* [Quick install](#quick-install)
* [Examples](docs/usage.md#examples)
* [Design](docs/design.md)
* [Metrics](docs/metrics.md)
* [FAQ](docs/faq.md)
* [Contributing](CONTRIBUTING.md)

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

Please note that the `Disruption` resource is **immutable**. Once applied, you can't edit it. If you need to change the disruption definition, you need to delete the existing resource and to re-create it.

Visit the [usage guide](docs/usage.md) for more customizations and sample use cases!

## Quick install

**NOTE:** Datadog engineers should consult Chaos Engineering team to deploy chaos-controller to a new cluster.**

Please look at the [advanced installation documentation](docs/installation.md) for any customization.

### Requirements

Install [cert-manager](https://cert-manager.io/docs/installation/kubernetes/) before going further. It is required for the admission controller to get its own self-signed certificate.

### Install the manifests

This file is generate for each new release and will always point to the latest stable version of the controller.

```
kubectl apply -f https://raw.githubusercontent.com/DataDog/chaos-controller/main/chart/install.yaml
```
