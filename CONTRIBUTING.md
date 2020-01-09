# Contributing

## chaos-fi-controller chart

Note that the Helm chart is located in the `k8s-resources` [repo](https://github.com/DataDog/k8s-resources/tree/master/k8s/chaos-fi-controller).

Remember to update the chart with any updates to the CRDs or RBAC rules.

## Testing the controller locally

If you want to test the controller locally (without having to redeploy a new image on a staging cluster), please use the [minikube project](https://kubernetes.io/docs/setup/learning-environment/minikube/) as described below:

* start minikube with containerd engine
  * `make minikube-start`
* build the new image of the controller with your local changes
  * `make docker-build`
* deploy the CRD and the controller on the minikube cluster
  * `make install && make deploy`

If the controller is already deployed, you'll have to remove the running pod for changes to be applied.

**Known issue: the pod preset injecting the fake Datadog statsd environment variable is created at the end of the apply. The preset may not be applied on the chaos controller pod, making it to panic. If it's the case, you have to remove the pod so it's created again with the pod preset."**

### Minikube ISO

We need some specific kernel modules to be enabled to do some of the injections. Because some of them were not enabled by default in the ISO, we built a custom one following the [official guide](https://minikube.sigs.k8s.io/docs/contributing/iso/) which is stored in the [minikube/iso] directory.

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
