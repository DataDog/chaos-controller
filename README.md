# Chaos controller

This project has been created using [kubebuilder][https://github.com/kubernetes-sigs/kubebuilder].

## Table of content

* [What is this controller for?](#what-is-this-controller-for)
* [How to use it?](#how-to-use-it)
* [How to deploy it on my cluster?](#how-to-deploy-it-on-my-cluster)
* Disruptions
  * [Network failure](docs/network_failure.md)
  * [Network latency](docs/network_latency.md)
  * [Node failure](docs/node_failure.md)
* [Design](docs/design.md)
* [Metrics](docs/metrics.md)
* [FAQ](docs/faq.md)
* [Contributing](#contributing)

## What is this controller for?

The controller was created to facilitate automation requirements in Datadog chaos workflows and pipelines. It helps to deal with failures during gamedays by abstracting them, especially when dealing with big deployments or complex network operations.

The `controller` is deployed as a `Deployment`. It watches for changes on the `Disruption` CRD, as well as their child resources.

## How to use it?

The controller works with a custom Kubernetes resource named `Disruption` describing the wanted failures and the pods to target. By creating this resource in the namespace of the pods you want to affect, it'll create pods to inject the needed failures.

Please take a look at the different disruptions documentation linked in the table of content for more information about what they can do and how to use them.

Here is [a full example of the disruption resource](config/samples/chaos_v1beta1_disruption.yaml) with comments.

## How to deploy it on my cluster?

To deploy it on your cluster, two commands are needed:

* `make install` will create the CRD for the `Disruption` kind
* `make deploy` will apply the needed manifests to create the controller deployment

## Contributing

Please read the [contributing documentation](CONTRIBUTING.md) for more information.
