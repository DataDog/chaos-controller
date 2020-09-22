# FAQ

## How can I debug a failing disruption?

During injection and cleanup phases, new pods are created near the targeted pods (in the same namespace). If those pods are in error state, you can check the logs to understand what happened.

Please note that if an error occurred during the cleanup phase, those pods won't be removed in order to let you check the logs.

## My disruption resource is stuck on removal

If an error occurred during the cleanup of the disruption (which occurs on removal), the finalizer won't be removed in order to be able to debug what happened and potentially do some manual cleaning. Here is how to proceed.

### Look at the cleanup pods logs

First thing is to get cleanup pods created for your disruption and look at the logs of the ones which errored.

```sh
kubectl -n <NAMESPACE> get pods -l chaos.datadoghq.com/pod-mode=clean
```

### Eventually re-trigger the cleanup phase

If needed, you can re-trigger the cleanup phase. It'll create cleanup pods for the disruption. Because the cleanup phase is idempotent, trigger the cleanup on an already cleaned up pod is a noop.

You need to edit the disruption, remove the `status.isFinalizing` field, and save.

```sh
kubectl -n <NAMESPACE> edit disruption <DISRUPTION>
```

```yaml
[...]
status:
  isFinalizing: true
[...]
```

**Please note that even if the retry succeeds, the disruption resource won't be removed by itself. Please look at the next step to finish the removal.**

### Force resource removal

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

It'll instantly delete the resource and garbage collect other related resources.

## The controller fails to watch or list disruptions

If you see the following error in controller logs, it is probably because of a malformed label selector:

```
Failed to list *v1beta1.Disruption: v1beta1.DisruptionList.ListMeta: v1.ListMeta.TypeMeta: Kind: Items: []v1beta1.Disruption: v1beta1.Disruption.Spec: v1beta1.DisruptionSpec.Selector: ReadString: expects " or n, but found 1, error found in #10 byte of ...|o","foo":1}}}],"kind|..., bigger context ...|"protocol":"tcp"},"selector":{"app":"demo","foo":1}}}],"kind":"DisruptionList","metadata":{"continue|...
```

Label selectors values should always be string (quoted).
