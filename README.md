# Chaos Failures Injection Controller

This project has been created using [kubebuilder][]. Please follow the documentation to make any changes in this project. Here are the few things you have to know.

This repository contains the configuration and code for the `chaos-fi-controller` Kubernetes [controller][what-is-a-controller] and its associated [`CRDs`][crd].

[kubebuilder]: https://github.com/kubernetes-sigs/kubebuilder
[what-is-a-controller]: https://book.kubebuilder.io/basics/what_is_a_controller.html
[crd]: https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/

## What is the chaos-fi-controller

The controller was created to facilitate automation requirements in [chaos-engineering][]. 

The lack of resources available to achieve the different [chaos testing levels][levels] led to the creation of this [rfc][]. The `chaos-fi-controller` is an implementation of the recommended solution.

The `controller` is deployed as a `StatefulSet`. It watches for changes on the supported `CRDs`, as well as their child resources. See the [CRDs section](#crds) for more details on specific behaviour for a particular `CRD`.

The Helm chart is described in the chaos-fi-controller chart [section](#chaos-fi-controller-chart).

[chaos-engineering]: https://github.com/DataDog/chaos-engineering
[levels]: https://github.com/DataDog/chaos-engineering#chaos-testing-levels
[rfc]: https://github.com/DataDog/architecture/blob/3e8dd537946fb373599fe09259f146e756ec12fe/rfcs/chaos-engineering-dependencies-failures-injection/rfc.md#recommended-solution

## CRDs

The `CRDs` that the controller works with can be found in the `config/crds` [directory][crds-dir].

Examples `YAML` files for these `CRDs` can be found in the `config/samples` [directory][samples-dir].

Currently, the controller works with the following `CRDs`:
* [NetworkFailureInjection][nfi-crd]
  * [Example][nfi-example]
* [NodeFailureInjection][nofi-crd]
  * [Example][nofi-example]

[crds-dir]: https://github.com/DataDog/chaos-fi-controller/tree/master/config/crds
[samples-dir]: https://github.com/DataDog/chaos-fi-controller/tree/master/config/samples
[nfi-crd]: https://github.com/DataDog/chaos-fi-controller/blob/master/config/crds/chaos_v1beta1_networkfailureinjection.yaml
[nfi-example]: https://github.com/DataDog/chaos-fi-controller/blob/master/config/samples/chaos_v1beta1_networkfailureinjection.yaml
[nofi-crd]: https://github.com/DataDog/chaos-fi-controller/blob/master/config/crds/chaos_v1beta1_nodefailureinjection.yaml
[nofi-example]: https://github.com/DataDog/chaos-fi-controller/blob/master/config/samples/chaos_v1beta1_nodefailureinjection.yaml


### NetworkFailureInjections

A `NetworkFailureInjection` provides an automated way of injecting `iptables` [rules][gameday-iptables].

Its behaviour is specified in the `spec`, for example:
```yaml
apiVersion: chaos.datadoghq.com/v1beta1
kind: NetworkFailureInjection
metadata:
  labels:
    controller-tools.k8s.io: "1.0"
  name: networkfailureinjection-sample
  namespace: mynamespace # this should be the same namespace as the pods you want to target
spec:
  failure:
    host: my-service.namespace.svc.barbet.cluster.local
    port: 80
    probability: 50 # probability is a WIP
    protocol: tcp
  selector:
    app: my-app
  numPodsToTarget: 1 # optional
```
* `host`: can be either a FQDN, a single IP or an IP block
* `port`: destination port
* `selector`: label selectors
* `numPodsToTarget`: _(Optional)_ how many random pods to target

Here, we specify that we want _outgoing packets to my-service.namespace.svc.barbet.cluster.local on port 80 to be dropped, with a probability (WIP) of 50%_.

**NOTE: Ensure that you create the nfi in the same namespace as the pods you want to target!**

#### Creating an nfi

```bash
# Using the example provided
k apply -f config/samples/chaos_v1beta1_networkfailureinjection.yaml

# To view nfis with kubectl
k get networkfailureinjection
k get nfi

# Describing nfis provides some events about inject/cleanup pods creation
k describe nfi mynfi
```

If `numPodsToTarget` is not specified, the controller will target **all pods in the same namespace as the nfi, matching the `spec.selector` label selectors**.

Otherwise, it will select **_numPodsToTarget_ random pods in the same namespace as the nfi, matching the `spec.selector` label selectors**.

For each matching pod, an _inject pod_ is created on the same node, which will create the `iptables` rules in a custom chain. The inject pod's owner reference is set as its creating `nfi`, so that they are garbage collected together. The image for the inject pods is built from the [`chaos-fi` repository][chaos-fi].

After creating an `nfi`, `k get po -n <namespace>` will show any created inject pods for an existing `nfi`.

#### Deleting an nfi

```bash
k delete nfi mynfi
```

The `nfi` won't be deleted right away. _Cleanup pods_ get created, one for each pod that had been selected, to flush the custom `iptables` chain. A [`finalizer`][finalizer] called `clean.nfi.finalizer.datadog.com` is used to ensure the `nfi` and its child inject/cleanup pods will only be deleted once all of its cleanup pods run to completion (i.e. have a pod phase of `Succeeded`).

[finalizer]: https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#finalizers


We can look at the `nfi`'s `status` for some additional information:
```bash
k describe nfi networkfailureinjection-sample
# or
k get nfi mynfi -o yaml
# or to only get the status
k get nfi mynfi -o json | jq '.status'
```

Example output:
```json
{
  "finalizing": true,
  "injected": true,
  "pods": [
      "pod1",
      "pod2"
  ]
}
```
* `finalizing`: Whether or not delete has been called on this nfi
* `injected`: Whether or not all inject pods for the selected `pods` (see below) have been successfully created
* `pods`: A list of the names of selected pods



*Note: Since the controller only watches `nfis` and their child inject/cleanup pods, any pods created after the `nfi` has selected its pods will not be affected.*

[gameday-iptables]: https://github.com/Datadog/devops/wiki/Game-Days#iptables
[chaos-fi]: https://github.com/DataDog/chaos-fi

### NodeFailureInjection

This injection basically triggers a kernel panic on the targeted pods' nodes. The only thing to be careful is that it'll make the entire node crash and the pods running on it even if they haven't been targeted by the label selector.

Because it makes the node to crash and because the controller needs to schedule a pod on the node to crash to make it crash, the created injection pods state won't be updated before a few minutes (the time for the Kubelet to be able to recover).

If the failure succeed, the pod should have the `ExitCode:0` status.

## chaos-fi-controller chart

Note that the Helm chart is located in the `k8s-resources` [repo](https://github.com/DataDog/k8s-resources/tree/master/k8s/chaos-fi-controller).

Remember to update the chart with any updates to the CRDs or RBAC rules.

## Testing the controller locally

If you want to test the controller locally (without having to redeploy a new image on a staging cluster), please use the [minikube project](https://kubernetes.io/docs/setup/learning-environment/minikube/) as described below:

* start minikube
* ensure your docker client is configured to use the minikube docker daemon
  * `eval $(minikube docker-env)`
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
