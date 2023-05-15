# Contributing

This document explains how to install and run the project on a local Lima cluster.

## Signing commits using `gpg`

- Download gpg [here](https://gnupg.org/download/)
- [Generating a new `gpg` key](https://docs.github.com/en/github/authenticating-to-github/managing-commit-signature-verification/generating-a-new-gpg-key)
- [Add `gpg` key to your GitHub account](https://docs.github.com/en/github/authenticating-to-github/managing-commit-signature-verification/adding-a-new-gpg-key-to-your-github-account)
- [Tell git about your signing key](https://docs.github.com/en/github/authenticating-to-github/managing-commit-signature-verification/telling-git-about-your-signing-key)
- [Automatically sign all commits](https://docs.github.com/en/github/authenticating-to-github/managing-commit-signature-verification/signing-commits)

## Requirements

To get started, we need to have the following software installed:

- [docker](https://docs.docker.com/get-docker/)
- [lima >= v0.14.0](https://github.com/lima-vm/lima)
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

### Quick start with Lima: `make lima-all`

**NOTE: (experimental) you can start the stack using cgroups v2 by running `CGROUPS=v2 make lima-all`. Some features of the chaos-controller may not be working in this mode as of today.**

**NOTE: sometimes, the instance can hang on waiting for ssh for some reasons. When it's the case, best bet is to run `make lima-stop` and re-run `make lima-all` again.**

Once you have installed the above requirements, run the `make lima-all` command to spin up a local stack. This command will run the following targets:

- `make lima-start` to create the lima vm with containerd and Kubernetes (backed by k3s)
- `make lima-kubectx` to add the lima Kubernetes cluster config to your local configs and switch to the lima context
- `make lima-install-cert-manager` to install cert-manager
- `make lima-build` to build the chaos-controller images
- `make lima-install` to render and apply the chaos-controller helm chart

Once the instance is started, you can log into it using either the `lima` or its longer form `limactl shell default` commands.

### Deploying local changes to Lima: `make lima-redeploy`

To deploy changes made to the controller code or chart, run the `make lima-redeploy` command that will run the following targets:

- `make lima-build` to build the chaos-controller images
- `make lima-install` to render and apply the chaos-controller helm chart
- `make lima-restart` to restart the chaos-controller manager pod

### Tearing down the local stack: `make lima-stop`

### Testing local changes in Lima

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

> NB: to detect if you need `longhorn` or not, simply install a disk disruption, e.g. `kubectl apply -f examples/disk_pressure_read.yaml`, if the `chaos-engineering/chaos-disk-pressure-read-XXXXX` is NOT in status `Running` and output logs similar to `error initializing the disk pressure injector","disruptionName":"disk-pressure-read","disruptionNamespace":"chaos-demo","targetName":"demo-curl-588bd4ffc8-q5wnk","targetNodeName":"lima","error":"error initializing disk informer: error executing ls command: exit status 2\noutput: ls: cannot access '/dev/disk/by-label/data-volume': No such file or directory\n` you will need to install `longhorn` as explained below. Delete failing disk disruption before proceeding to the next sections:

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

Once demo are now running, you can now look at throttling in the dedicated containers:

- To validate read throttling, look at `read-file` container logs
  - `kubectl logs --follow --timestamps --prefix --since 1m --tail 20 -c read-file -n chaos-demo -l app=demo-curl`
- Then apply read pressure to the pod:
  - `kubectl apply -f examples/disk_pressure_read.yaml`
- Once disruption is applied (chaos-engineering/pod status is RUNNING), you should witness the slower reading speed (limited at `1.0 kB/s`)
  - `kubectl delete -f examples/disk_pressure_read.yaml`
- Following the disruption deletion, reading speed should be back to normal

- To validate write throttling, look at `write-file` container logs
  - `kubectl logs --follow --timestamps --prefix --since 1m --tail 20 -c write-file -n chaos-demo -l app=demo-curl`
- Then apply write pressure to the pod:
  - `kubectl apply -f examples/disk_pressure_write.yaml`
- Once disruption is applied (chaos-engineering/pod status is RUNNING), you should witness the slower writing speed (limited at `1.0 kB/s`)
  - `kubectl delete -f examples/disk_pressure_write.yaml`
- Following the disruption deletion, writing speed should be back to normal

#### Testing gRPC disruption manually

The [gRPC disruption](docs/grpc_disruption.md) cannot be tested on the nginx client/server pods. To modify and test gRPC disruption [code](grpc/), visit the dogfood [README.md](dogfood/README.md) and the dogfood [CONTRIBUTING.md](dogfood/CONTRIBUTING.md) documents.

### Testing with end-to-end tests

The project contains end-to-end test which are meant to run against a real Kubernetes cluster. You can run them easily with the `make e2e-test` command. Please ensure that the following requirements are met before running the command:

- you deployed your changes locally (see [Deploying local changes to Lima](#deploying-local-changes-to-lima))
- your Kubernetes context is set to Lima (`kubectx lima` for instance)

The end-to-end tests will create a set of dummy pods in the `default` namespace of the Lima cluster that will be used to inject different kind of failures and make different assertions against those injections.

### Uploading test results to datadog

In case you have a Datadog account and want to push the tests results to it, you can do the following:

- Create an API key [here](https://app.datadoghq.com/organization-settings/api-keys)
- Store it securily and add it to your `.zshrc`:

```bash
security add-generic-password -a ${USER} -s datadog_api_key -w
# security delete-generic-password -a ${USER} -s datadog_api_key
# in your .zshrc or similar you can then do:
export DATADOG_API_KEY=$(security find-generic-password -a ${USER} -s datadog_api_key -w)
```

- Install `datadog-ci` by running `make install-datadog-ci`

- Run tests `make test || make e2e-test`

- Go to Datadog you [test-services](https://app.datadoghq.com/ci/test-services?env=local&view=branches&paused=false)

## 3rd-party licenses

3rd-party references and licenses are kept in the [LICENSE-3rdparty.csv](LICENSE-3rdparty.csv) file. This file has been generated by the [tasks/thirdparty.py](tasks/thirdparty.py) script. If any vendor is updated, added or removed, this file must be updated as well.
