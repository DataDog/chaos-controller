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
