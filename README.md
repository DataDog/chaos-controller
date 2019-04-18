# Chaos Failures Injection controller

This project has been created using [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder). Please follow the documentation to make any changes in this project. Here are the few things you have to know.

## Releasing a new version of the controller

You can manually build images on build-stable and staging (and prod when on master) environment from Gitlab. It'll then take the short commit SHA as a tag.

However, to release a proper version of the controller, you have to create a tag from the `master` branch:

```
git tag -a 1.0.0
git push --follow-tags origin master
```

It'll then automatically run jobs to push the image with the defined tag on every environment.

## Re-generating the CRD

When the API package is changed, the CRD (custom resource definition) must be re-generated. To achieve that, just run the `make` command (or `make generate` if you don't want to trigger linters, not recommended).

## Force deleting the CRD 
If you need to delete an existing CRD object from a cluster, you will need to remove the finalizer `clean.nfi.finalizer.datadog.com`.

This can be done by first editing the object, and then deleting it:
```bash
k edit nfi {NAME}
# remove finalizer
# Alternatively, you can run k patch nfi/{NAME} -p '{"metadata":{"finalizers":[]}}' --type=merge
k delete nfi {NAME}
```

## Testing

Tests are found under the `/pkg` directory, in `*test.go` files.

### Running tests

The controller tests should be run in `minikube`, since they **will actually create Kubernetes objects**.

The tests use the [test environment][test-env] supplied by [controller-runtime][controller-runtime], but this does not
currently support the [controller-manager][controller-manager-support]. As such, testing within an actual cluster provides the best means of testing the controller's actual behaviour.

**Note: the test environment does not have garbage collection.**

### Requirements:

* [minikube][minikube] **running as the current context**
* [Ginkgo](https://github.com/onsi/ginkgo)
  ```bash
  go get -u github.com/onsi/ginkgo/ginkgo
  ```
* [Gomega](https://github.com/onsi/gomega)
    ```bash
  go get -u github.com/onsi/gomega/...
  ```

### Running tests

The test environment will always do a new install of the `crds`, so before running tests, these need to be deleted:
```bash
k delete crd/networkfailureinjections.chaos.datadoghq.com
```

You can use the supplied `Makefile`:
```bash
# Delete crd
make test
```

For more detailed output:
```bash
# Delete crd
make test-ginkgo
```

### Adding tests
Please ensure that any added tests handle deletion of created resources, since the [test environment][test-env] does not support garbage collection.

[minikube]: https://kubernetes.io/docs/setup/minikube/
[test-env]: https://godoc.org/sigs.k8s.io/controller-runtime/pkg/envtest
[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
[controller-manager-support]: https://github.com/kubernetes-sigs/testing_frameworks/pull/41
