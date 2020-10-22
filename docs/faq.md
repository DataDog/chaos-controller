# FAQ

## How can I debug a failing disruption?

During injection and cleanup phases, new pods are created near the targeted pods (in the same namespace). If those pods are in error state, you can check the logs to understand what happened.

Please note that if an error occurred during the cleanup phase, those pods won't be removed in order to let you check the logs.

## My disruption resource is stuck on removal

If an error occurred during the cleanup of the disruption (which occurs on removal), the controller will retry to cleanup up to 5 times (with a random wait time between each try). If the cleanup still fails, the disruption resource will be marked as `Stuck on removal` and events will be registered in both the disruption and impacted pods to ask for a manual debugging. The finalizer won't be removed in order to be able to debug what happened and potentially do some manual cleaning. Here's what to do in this case:

### Look at the cleanup pods logs

First thing is to get cleanup pods created for your disruption and look at the logs of the ones which errored.

```sh
kubectl -n <NAMESPACE> get pods -l chaos.datadoghq.com/pod-mode=clean
```

### Eventually re-trigger the cleanup phase

If needed, you can re-trigger the cleanup phase. It'll create cleanup pods for the pods not having been cleaned yet. To trigger this, remove any cleanup pods in the `Error` state (you can use the command above to list them). A new cleanup pod will be created during the next reconcile loop (it can take up to one minute to be triggered). If this pod succeeds to cleanup the disruption, it'll then be deleted.

### Eventually force resource removal

Once you're sure you want to remove everything related to your disruption resource, just edit it and remove the finalizer from the list.

```sh
kubectl -n <NAMESPACE> edit disruption <DISRUPTION>
```

```yaml
[...]
 finalizers:
  - finalizer.chaos.datadoghq.com
[...]
```

It'll instantly delete the resource and garbage collect other related resources. Please note that if you took no actions to cleanup what was applied by this disruption, this step won't do it for you!

## The controller fails to watch or list disruptions

If you see the following error in controller logs, it is probably because of a malformed label selector:

```
Failed to list *v1beta1.Disruption: v1beta1.DisruptionList.ListMeta: v1.ListMeta.TypeMeta: Kind: Items: []v1beta1.Disruption: v1beta1.Disruption.Spec: v1beta1.DisruptionSpec.Selector: ReadString: expects " or n, but found 1, error found in #10 byte of ...|o","foo":1}}}],"kind|..., bigger context ...|"protocol":"tcp"},"selector":{"app":"demo","foo":1}}}],"kind":"DisruptionList","metadata":{"continue|...
```

Label selectors values should always be string (quoted).
