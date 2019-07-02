# Chaos Failures Injection Controller

This project has been created using [kubebuilder][]. Please follow the documentation to make any changes in this project. Here are the few things you have to know.

This repository contains the configuration and code for the `chaos-fi-controller` Kubernetes [controller][what-is-a-controller] and its associated [`CRDs`][crd].

[kubebuilder]: https://github.com/kubernetes-sigs/kubebuilder
[what-is-a-controller]: https://book.kubebuilder.io/basics/what_is_a_controller.html
[crd]: https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/

## Table of content

* [What is the chaos-fi-controller](#what-is-the-chaos-fi-controller)
* Failures
  * [NetworkFailureInjection](docs/network_failure.md)
  * [NodeFailureInjection](docs/node_failure.md)
* [Design](docs/design.md)

## What is the chaos-fi-controller

The controller was created to facilitate automation requirements in [chaos-engineering][]. 

The lack of resources available to achieve the different [chaos testing levels][levels] led to the creation of this [rfc][]. The `chaos-fi-controller` is an implementation of the recommended solution.

The `controller` is deployed as a `StatefulSet`. It watches for changes on the supported `CRDs`, as well as their child resources. See the [CRDs section](#crds) for more details on specific behaviour for a particular `CRD`.

The Helm chart is described in the chaos-fi-controller chart [section](#chaos-fi-controller-chart).

[chaos-engineering]: https://github.com/DataDog/chaos-engineering
[levels]: https://github.com/DataDog/chaos-engineering#chaos-testing-levels
[rfc]: https://github.com/DataDog/architecture/blob/3e8dd537946fb373599fe09259f146e756ec12fe/rfcs/chaos-engineering-dependencies-failures-injection/rfc.md#recommended-solution

## How to use it?

The controller works with custom Kubernetes resources describing the wanted failures and the pods to target. By creating those resources in the namespace of the pods you want to affect, it'll create pods to inject the needed failures.

Please take a look at the different failures documentations linked in the table of content of this repository for more information about what they are doing and how to use them.

## chaos-fi-controller chart

Note that the Helm chart is located in the `k8s-resources` [repo](https://github.com/DataDog/k8s-resources/tree/master/k8s/chaos-fi-controller).

Remember to update the chart with any updates to the CRDs or RBAC rules.

## Testing the controller locally

If you want to test the controller locally (without having to redeploy a new image on a staging cluster), please use the [minikube project](https://kubernetes.io/docs/setup/learning-environment/minikube/) as described below:

* start minikube with containerd engine
  * `make minikube-start`
* build the new image of the controller with your local changes
  * `make docker-build`
* deploy the controller on the minikube cluster
  * `make deploy`

If the controller is already deployed, you'll have to remove the running pod for changes to be applied

The controller relies on the [chaos-fi](https://github.com/DataDog/chaos-fi) image to inject failures. Please build it locally as well if you want to injection and cleanup pods created by the controller to succeed.

## Releasing a new version of the controller

You can manually build images on build-stable and staging (and prod when on master) environment from Gitlab. It'll then take the short commit SHA as a tag.

However, to release a proper version of the controller, you have to create a tag from the `master` branch:

```
git tag -a 1.0.0
git push --follow-tags origin master
```

It'll then automatically run jobs to push the image with the defined tag on every environment.

## Re-generating the CRD

When the API package is changed, the CRD (custom resource definition) must be re-generated. To achieve that, just run the `make` command (or `make generate` if you don't want to trigger tests and linters, not recommended).

## Force deleting the CRD 
If you need to delete an existing CRD object from a cluster, you will need to remove the finalizer `clean.nfi.finalizer.datadog.com`.

This can be done by first editing the object, and then deleting it:
```bash
# remove the clean.nfi.finalizer.datadog.com finalizer
k edit nfi mynfi
# or 
k patch nfi/mynfi -p '{"metadata":{"finalizers":[]}}' --type=merge


# delete nfi
k delete nfi mynfi
```
