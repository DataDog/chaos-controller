# FAQ

## How can I debug a failing failure?

During injection and cleanup phases, new pods are created near the targeted pods (in the same namespace). If those pods are in error state, you can check the logs to understand what happened.

Please note that if an error occured during the cleanup phase, those pods won't be removed in order to let you check the logs.

## My failure resource is stuck on removal

If an error occured during the cleanup of the failure (which occurs on removal), the finalizer won't be removed in order to be able to debug what happened and potentially do some manual cleaning.

Once you're sure you want to remove everything related to your failure resource, just edit it and remove the finalizer from the list.

```
# example with a NetworkFailureInjection resource
k edit nfi my-failure
```

```yaml
[...]
 finalizers:
  - clean.nfi.finalizer.datadog.com
[...]
```

It'll instantly delete the resource and garbage collect other related resources.
