# Design

The controller is made of multiple custom resources, each resource describing a failure kind. It has been created with [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) so feel free to check the documentation for more details about the usual controller design.

The logic is quite the same for every resources:

* on resource creation, the controller will
  * target a set of pods according to the given label selector
  * randomly select pods in those targeted pods
  * for each selected pod, it'll create a side pod on the same node to inject the failure
* on resource deletion, the controller will
  * retrieve the infected nodes
  * create a cleanup pod on every infected nodes
  * wait for those cleanup pods to successfully finish their job before garbage collecting everything
    * if the cleanup fails, the controller will
      * wait for X seconds (X being between 5 and 10)
      * requeue the request to reconcile it again
      * create a new cleanup pod
    * the cleanup retries up to 5 times before considering the disruption stuck on removal

## Note on failures cleanup

Some failures don't need cleanup pods (like node failure). In this case, the resource deletion will just lead to the garbage collection.

The cleanup requires a finalizer on the failure resource to avoid the garbage collection to run before the end of the cleanup. In case of error during the cleanup, this finalizer won't be removed so everything related to the deleted failure resource will stay and won't be garbage collected. It allows you to easily inspect pods to know why it has failed and to do a manual cleanup if needed. To unstuck the resource deletion once everything is cleaned, you need to manually remove the finalizer from the finalizers list by editing the failure resource.
