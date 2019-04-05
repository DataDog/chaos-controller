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

When the API package is changed, the CRD (custom resource definition) must be re-generated. To achieve that, just run the `make` command (or `make generate` if you don't want to trigger tests and linters, not recommended).

## Force deleting the CRD 
If you need to delete an existing CRD object from a cluster, you will need to remove the finalizer `clean.dfi.finalizer.datadog.com`.

This can be done by first editing the object, and then deleting it:
```bash
k edit dependencyfailureinjection {NAME}
# remove finalizer
k delete dependencyfailureinjection {NAME}
```
