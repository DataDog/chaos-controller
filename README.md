# Chaos Failures Injection controller

This project has been created using [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder). Please follow the documentation to make any changes in this project. Here are the few things you have to know.

## Re-generating the CRD

When the API package is changed, the CRD (custom resource definition) must be re-generated. To achieve that, just run the `make` command (or `make generate` if you don't want to trigger tests and linters, not recommended).

## Force deleting the CRD 
If you need to delete an existing CRD object from a cluster, you will need to remove the finalizer `clean.nfi.finalizer.datadog.com`.

This can be done by first editing the object, and then deleting it:
```bash
k edit nfi {NAME}
# remove finalizer
k delete nfi {NAME}
```
