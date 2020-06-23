# Testing

## Requirements

To get started, we need to have the following software installed:

* [minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/)
* [golangci-lint](https://github.com/golangci/golangci-lint)

This project is based on kubebuilder, please make sure the [listed](https://book.kubebuilder.io/quick-start.html#prerequisites) requirements for kubebuilder are installed as well.

## Developing locally (minikube)

For using the chaos-controller on minikube we need our own custom build ISO image available on s3. To find out more about what changes are in this image and the process to create a new version, see [this document](./minikube_image.md)

* Start minikube with **containerd** container runtime:
  * `make minikube-start`
* Build the controller container images locally (_Docker for_) and copy them to minikube:
  * `make docker-build`
* Build and deploy the CRD:
  * `make install`

``` sh
make minikube-start
make docker-build
make install
```

### Applying code changes

Applying code changes, requires you to rebuild the images. Re-running `make docker-build && make install` should make the new images available.

Delete the manager pod to use the new image `kubectl delete pod -l control-plane=controller-manager`

## Running tests

Run `make test` to run the test suite. This will also generate all the require boilerplate code.

## Manual verification

With `minikube` and the `controller-manager` running we can start our chaos experiments. We included a sample application which can be used for verification.

* Start the sample application:
  * `kubectl apply -f config/samples/deployment.yaml`
* Verify that the app is running:
  * `kubectl get pods -l app=demo`

### Applying experiments

Verify the contents of the [chaos_v1beta1_disruption.yaml](config/samples/chaos_v1beta1_disruption.yaml). When running on minikube, disable the `nodeFailure` disruption as this will shutdown minikube.

* Applying the disruption(s):
  * `kubectl apply -f config/samples/chaos_v1beta1_disruption.yaml`

For verification on minikube we created some helper [scripts](scripts/). To use them: `source scripts/common`

* List contents of iptables:
  * `./scripts/list_iptables.sh <pod_name>`
* Show links:
  * `./scripts/list_links.sh <pod_name>`
* List traffic control filters:
  * `./scripts/list_tc_filters.sh <pod_name>`
* Show qdisc:
  * `./scripts/list_tc_qdiscs.sh <pod_name>`

Running your own command inside the pod can be done via `exec_into_pod <pod_name> "<cmd>"`. For example `exec_into_pod demo-548d697fbb-66tp9 "ping 8.8.8.8"`.

When you are finished with the experiments, remove the disruption: `kubectl delete -f config/samples/chaos_v1beta1_disryption.yaml`

## Troubleshooting

Check logs for errors:

* manager: `kubectl logs -l control-plane=controller-manager -c manager`
* injector: `kubectl logs -l chaos.datadoghq.com/pod-mode=inject`

Check pod events to troubleshoot startup issues:

* manager: `kubectl describe pod -l control-plane=controller-manager`
* injector: `kubectl describe pod -l chaos.datadoghq.com/pod-mode=inject`
