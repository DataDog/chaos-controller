> **Latest Kubernetes version supported is 1.19.x!** _This [Kubernetes issue](https://github.com/kubernetes/kubernetes/issues/97288") prevents the controller from running properly on Kubernetes 1.20.x which will not be supported. The [fix](https://github.com/kubernetes/kubernetes/pull/97980) will be released with Kubernetes 1.21._

# Chaos Controller

Datadog runs regular chaos experiments to test the resilience of our distributed cloud applications hosted in Kubernetes. The Chaos Controller facilitates automation of these experiments by simulating common "disruptions" including but not limited to: poor network quality, exhaustion of computational resources, or unexpected node failures. All you need to do to get started is define a `yaml` file which contains all of the specifications needed by our custom Kubernetes Resource to run the preferred disruption `Kind`!

> **Disclaimer:**
> _The **Chaos Controller** allows you to disrupt your Kubernetes infrastructure through various means including but not limited to: bringing down resources you have provisioned and preventing critical data from being transmitted between resources. The use of **Chaos Controller** on your production system is done at your own discretion and risk._

# Installation

**NOTE:** Datadog engineers should consult Chaos Engineering team to deploy chaos-controller to a new cluster.

Make sure [cert-manager](https://cert-manager.io/docs/installation/kubernetes/) is installed for the admission controller to get its own self-signed certificate, then run:

```
kubectl apply -f https://raw.githubusercontent.com/DataDog/chaos-controller/main/chart/install.yaml
```

This `install.yaml` is generated for each new release and will always point to the latest stable version of the controller. If you already have a Kubernetes environment, you can install **Chaos Controller** from Docker Hub.

> _The [Advanced Installation Docus](docs/installation.md) contain flags to customize webhooks, annotate injector pods, etc._

> _The [Contributing Docs](CONTRIBUTING.md) explain how to spin up a local Minikube with demo pods so you can run your first disruption!_

# Getting Started

Disruptions are built as short-living resources which should be manually created and removed once your experiments are done. They should not be part of any application deployment. The `Disruption` resource is **immutable**. Once applied, you can't edit it. If you need to change the disruption definition, you need to delete the existing resource and to re-create it.

Getting started is as simple as creating a Kubernetes resource:

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

To disrupt your cluster, run `kubectl apply -f <disruption_file.yaml>`. You can clean up the disruption with `kubectl delete -f <disruption_file>.yaml`. For your safety, we recommend you get started with the `dry-run` mode enabled.

> _The [usage guide](docs/usage.md) contains usecases and sample disruption files!_

# Useful Links

* [Examples](docs/usage.md#examples)
* [Design](docs/design.md)
* [Metrics](docs/metrics.md)
* [FAQ](docs/faq.md)
* [Contributing](CONTRIBUTING.md)