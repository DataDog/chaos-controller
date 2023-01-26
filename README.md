**Oldest Kubernetes version supported: 1.16**

> :warning: **Kubernetes version 1.20.x is not supported!** _This [Kubernetes issue](https://github.com/kubernetes/kubernetes/issues/97288) prevents the controller from running properly on Kubernetes 1.20.0-1.20.4. Earlier versions of Kubernetes as well as 1.20.5 and later are still supported._

# Datadog Chaos Controller

> *:bomb: Disclaimer :bomb:*
>
> _The Chaos Controller allows you to disrupt your Kubernetes infrastructure through various means including but not limited to: bringing down resources you have provisioned and preventing critical data from being transmitted between resources. The use of Chaos Controller on your production system is done at your own discretion and risk._

The Chaos Controller is a Kubernetes controller with which you can inject various systemic failures, at scale, and without caring about the implementation details of your Kubernetes infrastructure. It was created with a specific mindset answering Datadog's internal needs:

* 🐇 **Be fast and operate at scale**
  * At Datadog, we are running experiments injecting and cleaning failures to/from thousands of targets within a few minutes.
* 🚑 **Be safe and operate in highly disrupted environments**
  * The controller is built to be able to limit the blast radius of failures but also to be able to recover by itself in catastrophic scenarios.
* 💡 **Be smart and operate in various technical environments**
  * With Kubernetes, all environments are built differently.
  * Whatever your cluster configuration and implement details choice, the controller is able to inject failures by relying on low-level Linux kernel features such as cgroups, tc or even eBPF.
* 🪙 **Be simple and operate at low cost**
  * Most of the time, your Chaos Engineering platform is waiting and doing nothing.
  * We built this project so it uses resources only when it is really doing something:
    * No DaemonSet or any always-running processes on your nodes for injection, no reserved resources when it's not needed.
    * Injection pods are created only when it is needed, killed once experiment is done, and built to be evicted if necessary to free resources.
    * A single long-running pod, the controller, and nothing else!

## Getting Started

> :bulb: Read the [latest release quick installation guide](https://github.com/DataDog/chaos-controller/releases/latest) and the [configuration guide](docs/configuration.md) to know how to deploy the controller.

Disruptions are built as short-living resources which should be manually created and removed once your experiments are done. They should not be part of any application deployment. The `Disruption` resource is **immutable**. Once applied, you can't edit it. If you need to change the disruption definition, you need to delete the existing resource and to re-create it.

Getting started is as simple as creating a Kubernetes resource:

```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: Disruption
metadata:
  name: node-failure
  namespace: chaos-demo # it must be in the same namespace as targeted resources
spec:
  selector: # a label selector used to target resources
    app: demo-curl
  count: 1 # the number of resources to target, can be a percentage
  duration: 1h # the amount of time before your disruption automatically terminates itself, for safety
  nodeFailure: # trigger a kernel panic on the target node
    shutdown: false # do not force the node to be kept down
```

To disrupt your cluster, run `kubectl apply -f <disruption_file.yaml>`. You can clean up the disruption with `kubectl delete -f <disruption_file>.yaml`. For your safety, we recommend you get started with the `dry-run` mode enabled.

> :open_book: The [features guide](docs/features.md) details all the features of the Chaos Controller.

> :open_book: The [examples guide](docs/examples.md) contains a list of various disruption files that you can use.

> Check out [Chaosli](./cli/chaosli/README.md) if you want some help understanding/creating disruption configurations.

## Contributing

Chaos Engineering is necessarily different from system to system. We encourage you to try out this tool, and extend it for your own use cases. If you want to run the source code locally to make and test implementation changes, visit the [Contributing Doc](CONTRIBUTING.md). By the way, we welcome Pull Requests.

## Useful Links

- [Examples of disruptions](docs/examples.md)
- [General design](docs/design.md)
- [Reported metrics](docs/metrics_events.md)
- [FAQ](docs/faq.md)
