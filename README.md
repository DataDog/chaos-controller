# Chaos Failures Injection Controller

This project has been created using [kubebuilder][]. Please follow the documentation to make any changes in this project. Here are the few things you have to know.

This repository contains the configuration and code for the `chaos-fi-controller` Kubernetes [controller][what-is-a-controller] and its associated [`CRDs`][crd].

[kubebuilder]: https://github.com/kubernetes-sigs/kubebuilder
[what-is-a-controller]: https://book.kubebuilder.io/basics/what_is_a_controller.html
[crd]: https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/

## Table of content

* [What is the chaos-fi-controller?](#what-is-the-chaos-fi-controller)
* Disruptions
  * [Network failure](docs/network_failure.md)
  * [Network latency](docs/network_latency.md)
  * [Node failure](docs/node_failure.md)
* [Design](docs/design.md)
* [Metrics](docs/metrics.md)
* [FAQ](docs/faq.md)
* [Contributing](#contributing)

## What is the chaos-fi-controller?

The controller was created to facilitate automation requirements in [chaos-engineering][]. It can also help to deal with failures during gamedays by abstracting them, especially when dealing with big deployments or complex network operations.

The lack of resources available to achieve the different [chaos testing levels][levels] led to the creation of this [rfc][]. The `chaos-fi-controller` is an implementation of the recommended solution.

The `controller` is deployed as a `Deployment`. It watches for changes on the `Disruption` CRD, as well as their child resources.

The Helm chart is described in the chaos-fi-controller chart [section](#chaos-fi-controller-chart).

[chaos-engineering]: https://github.com/DataDog/chaos-engineering
[levels]: https://github.com/DataDog/chaos-engineering#chaos-testing-levels
[rfc]: https://github.com/DataDog/architecture/blob/3e8dd537946fb373599fe09259f146e756ec12fe/rfcs/chaos-engineering-dependencies-failures-injection/rfc.md#recommended-solution

## How to use it?

The controller works with a custom Kubernetes resource describing the wanted failures and the pods to target. By creating this resource in the namespace of the pods you want to affect, it'll create pods to inject the needed failures.

Please take a look at the different failures documentations linked in the table of content of this repository for more information about what they are doing and how to use them.

You can have [a full example of the disruption resource](config/samples/chaos_v1beta1_disruption.yaml) with comments.

## Contributing

Please read the [contributing documentation](CONTRIBUTING.md) for more information.
