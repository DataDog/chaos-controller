# Chaos controller

The controller was created to facilitate automation requirements in Datadog chaos workflows and pipelines. It helps to deal with failures during gamedays by abstracting them, especially when dealing with big deployments or complex network operations.

The `controller` is deployed as a `Deployment`. It watches for changes on the `Disruption` CRD, as well as their child resources.

## Table of content

* [Usage](#usage)
* [Installation](#installation)
* [Design](docs/design.md)
* [Metrics](docs/metrics.md)
* [FAQ](docs/faq.md)
* [Contributing](#contributing)

## Usage

The controller works with a custom Kubernetes resource named `Disruption` describing the wanted failures and the pods to target. By creating this resource in the namespace of the pods you want to affect, it'll create pods to inject the needed failures.

Please take a look at the different disruptions documentation linked in the table of content for more information about what they can do and how to use them.

Here is [a full example of the disruption resource](config/samples/complete.yaml) with comments. You can also have a look at the following use cases:

* [I want to randomly kill one of my node](docs/node_failure.md)
* [I want to drop the connection between my pods and a service](docs/network_disruption.md)
* [I want to randomly corrupt network packets between my pods and a service](docs/network_disruption.md)
* [I want to add network latency to my pods](docs/network_disruption.md)
* [I want to restrict my pods bandwidth](docs/network_disruption.md)
* [I want to put CPU pressure against my pods](docs/cpu_pressure.md)
* [I want to throttle my disk to simulate slow IO](docs/disk_pressure.md)

## Installation

To deploy it on your cluster, you just have to run the `make install` command and it will create the CRD for the `Disruption` kind and apply the needed manifests to create the controller deployment.

You can uninstall it the same way, by using the `make uninstall` command.

### Chaos pod template

The [manager configmap](config/manager/config.yaml) contains the chaos pod template (`pod-template.json`) used to generate injection and cleanup pods. This template can be customized but you have to keep in mind that some fields are overridden by the controller itself when generating the pod:

* `.Metadata.GenerateName` is filled with a name like `chaos-<instace_name>-<mode>-` so it generates chaos pod names automatically
* `.Metadata.Namespace` is filled with the same namespace as the targeted pod
* `.Metadata.Labels`: a bunch of labels are added to existing labels
	* `chaos.datadoghq.com/pod-mode`: the mode of the pod (`inject` or `cleanup`)
	* `chaos.datadoghq.com/target-pod`: the pod targeted by the chaos pod
	* `chaos.datadoghq.com/disruption-kind`: the chaos pod failure kind
* `.Spec.NodeName` is filled with the same value as the targeted pod node name to fix the chaos pod on the same node as the targeted pod
* `.Spec.Containers[0].Image` is filled with the chaos injector image
* `.Spec.Containers[0].Args` is filled with arguments built for the chaos injector image

## Contributing

Please read the [contributing documentation](CONTRIBUTING.md) for more information.
