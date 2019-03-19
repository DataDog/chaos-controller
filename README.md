# Chaos Failures Injection controller

This project has been created using [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder). Please follow the documentation to make any changes in this project. Here are the few things you have to know.

## Re-generating the CRD

When the API package is changed, the CRD (custom resource definition) must be re-generated. To achieve that, just run the `make` command (or `make generate` if you don't want to trigger tests and linters, not recommended).
