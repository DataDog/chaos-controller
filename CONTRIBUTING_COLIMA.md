# Contributing

This document explains how to install and run the project on a local colima cluster.

## Signing commits using `gpg`

- Download gpg [here](https://gnupg.org/download/)
- [Generating a new `gpg` key](https://docs.github.com/en/github/authenticating-to-github/managing-commit-signature-verification/generating-a-new-gpg-key)
- [Add `gpg` key to your GitHub account](https://docs.github.com/en/github/authenticating-to-github/managing-commit-signature-verification/adding-a-new-gpg-key-to-your-github-account)
- [Tell git about your signing key](https://docs.github.com/en/github/authenticating-to-github/managing-commit-signature-verification/telling-git-about-your-signing-key)
- [Automatically sign all commits](https://docs.github.com/en/github/authenticating-to-github/managing-commit-signature-verification/signing-commits)

## Requirements

To get started, we need to have the following software installed:

- [docker](https://docs.docker.com/get-docker/)
- [colima >= v0.4.4](https://github.com/abiosoft/colima#installation)
- [golangci-lint](https://github.com/golangci/golangci-lint)
- [Kubebuilder Prerequisites](https://book.kubebuilder.io/quick-start.html#prerequisites) (go, docker, kubectl, kubebuilder, controller-gen)
- [helm](https://helm.sh/docs/intro/quickstart/)
- [envtest](#Installing-envtest)

## Installing Envtest

In order to run `make test` to run the unit tests, you'll need to install envtest with the following commands:

```
export ENVTEST_ASSETS_DIR="/usr/local/kubebuilder"
mkdir -p ${ENVTEST_ASSETS_DIR}
test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.8.3/hack/setup-envtest.sh
source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools ${ENVTEST_ASSETS_DIR}; setup_envtest_env ${ENVTEST_ASSETS_DIR};
```

## Developing Locally

### Quick start with Colima

Once you have installed the above requirements, run the following commands:

- start colima with containerd engine and a k3s cluster
  - `make colima-start`
- deploy cert-manager
  - `make install-cert-manager`
- install `nerdctl` alias to target colima containerd runtime to build container images (requires sudo)
  - `make colima-install-nerdctl`
- build the new image of the controller with your local files and load them into containerd runtime
  - `make colima-build`
- deploy the CRD and the controller on the colima cluster
  - `make colima-install`

### Deploying Local Changes to Colima

To deploy a new version of your local controller code when a version is already deployed, run:

- `make colima-build`
- `make colima-install`
- `make colima-restart`

To deploy a new version of the CRD by modifying your local `api/v1beta1/disruption_types.go` (or a particular Subspec by modifying `api/v1beta1/disruption_types.go`), run:

- `make colima-install`
- `make colima-restart`

### Testing Local Changes in Colima

#### Testing manually

The [samples](examples/) contains sample data which can be used to test your changes.

[demo.yaml](examples/demo.yaml) contains testing resources you can apply directly to your cluster. First, create the `chaos-demo` namespace, then bring up the demo pods:

- `kubectl apply -f examples/namespace.yaml`
- `kubectl apply -f examples/demo.yaml`

To see whether curls are succeeding, by using kubectl to tail the pod's logs, run:

- `kubectl -n chaos-demo logs -f -l app=demo-curl -c curl`

Once you define your test manifest, run:

- `kubectl apply -f examples/<manifest>.yaml`

> NB: a good example to start with is `kubectl apply -f examples/network_drop.yaml` that will block all outgoing traffic from `demo-curl` pod.

To remove the disruption, run:

- `kubectl delete -f examples/<manifest>.yaml` (`kubectl delete -f examples/network_drop.yaml`)

See [development guide](docs/development.md) for more robust documentation and tips!

#### Testing disk pressure manually

Once you have installed standard requirements, in order to `reliably` test disk throttling locally, you may want to use `longhorn` as a storage class.

> NB: to detect if you need `longhorn` or not, simply install a disk disruption, e.g. `kubectl apply -f examples/disk_pressure_read.yaml`, if the `chaos-engineering/chaos-disk-pressure-read-XXXXX` is NOT in status `Running` and output logs similar to `error initializing the disk pressure injector","disruptionName":"disk-pressure-read","disruptionNamespace":"chaos-demo","targetName":"demo-curl-588bd4ffc8-q5wnk","targetNodeName":"colima","error":"error initializing disk informer: error executing ls command: exit status 2\noutput: ls: cannot access '/dev/disk/by-label/data-volume': No such file or directory\n` you will need to install `longhorn` as explained below. Delete failing disk disruption before proceeding to the next sections:

```bash
kubectl delete -f examples/disk_pressure_read.yaml
kubectl patch \
    --type json \
    --patch='[ { "op": "remove", "path": "/metadata/finalizers" } ]' \
    -n chaos-engineering pods/<disruption-pod-name>
```

First install `longhorn`:

- `make install-longhorn`

Wait for `longhorn` components to be `Running` (this may takes several minutes):

- `kubectl get pods -n longhorn-system`

Then, if you already created `examples/demo.yaml`, delete it and re-create it:

- `kubectl delete -f examples/demo.yaml`
- `kubectl apply -f examples/demo.yaml`

This aims to ensure the PVC is created with the proper DEFAULT storage class, that has been changed following longhorn installation.

> NB: if you encounter an error like `Error from server (Forbidden): error when creating "examples/demo.yaml": persistentvolumeclaims "demo" is forbidden: Internal error occurred: 2 default StorageClasses were found` you may want to edit storage classes and keep a single default storage class `kubectl edit storageClass` look for `storageclass.kubernetes.io/is-default-class` annotation and remove the annotation to the storageClasses not being `longhorn`, once done, re-apply `demo`: `kubectl apply -f examples/demo.yaml`

Once demo are now runnning, you can now look at throttling in the dedicated containers:

- To validate read throttling, look at `read-file` container logs
  - `kubectl logs --follow --timestamps --prefix --since 1m --tail 20 -c read-file -n chaos-demo -l app=demo-curl`
- Then apply read pressure to the pod:
  - `kubectl apply -f examples/disk_pressure_read.yaml`
- Once disruption is applied (chaos-enginnering/pod status is RUNNING), you should witness the slower reading speed (limited at `1.0 kB/s`)
  - `kubectl delete -f examples/disk_pressure_read.yaml`
- Following the disruption deletion, reading speed should be back to normal

- To validate write throttling, look at `write-file` container logs
  - `kubectl logs --follow --timestamps --prefix --since 1m --tail 20 -c write-file -n chaos-demo -l app=demo-curl`
- Then apply write pressure to the pod:
  - `kubectl apply -f examples/disk_pressure_write.yaml`
- Once disruption is applied (chaos-enginnering/pod status is RUNNING), you should witness the slower writing speed (limited at `1.0 kB/s`)
  - `kubectl delete -f examples/disk_pressure_write.yaml`
- Following the disruption deletion, writing speed should be back to normal

### 3rd-party licenses

3rd-party references and licenses are kept in the [LICENSE-3rdparty.csv](LICENSE-3rdparty.csv) file. This file has been generated by the [tasks/thirdparty.py](tasks/thirdparty.py) script. If any vendor is updated, added or removed, this file must be updated as well.
