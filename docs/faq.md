# FAQ

## How can I debug a failing failure?

During injection and cleanup phases, new pods are created near the targeted pods (in the same namespace). If those pods are in error state, you can check the logs to understand what happened.

Please note that if an error occurred during the cleanup phase, those pods won't be removed in order to let you check the logs.

## My failure resource is stuck on removal

If an error occurred during the cleanup of the failure (which occurs on removal), the finalizer won't be removed in order to be able to debug what happened and potentially do some manual cleaning.

Once you're sure you want to remove everything related to your failure resource, just edit it and remove the finalizer from the list.

``` sh
k edit dis my-disruption
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
