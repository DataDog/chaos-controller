# FAQ

## Where can I find the chaos pods for my disruption?

In order to ensure that chaos pods have access to the ClusterRole they need, all chaos pods are created in the same namespace as the `chaos-injector`
 service account. This is configured with a flag when starting the chaos-controller. By default, this is the "chaos-engineering" namespace.

## How can I know if my disruption has been successfully injected or not?

A disruption has an `Injection Status` field in its status that you can see by describing the resource. It can take the following values:

* `NotInjected` when the disruption is not injected yet (no targets are affected)
* `PartiallyInjected` when the disruption is not fully injected yet (at least one target is affected)
* `Injected` when the disruption is fully injected (all targets are affected)

## How can I debug a disruption?

Applying a disruption creates a bunch of pods to inject and clean it. Those are typically created in the same namespace as the chaos-controller. You can look at the logs of those pods to understand what happened.

```sh
kubectl -n <NAMESPACE> get pods -l chaos.datadoghq.com/disruption-name=<DISRUPTION_NAME> -l chaos.datadoghq.com/disruption-namespace=<DISRUPTION_NAMESPACE>
kubectl -n <NAMESPACE> logs <POD_NAME>
```

## My disruption resource is stuck on removal, what should I do?

If an error occurred during the cleanup of the disruption (which occurs on removal), the controller will keep failing pods and the disruption will be marked as stuck on removal to allow you to see what happened and eventually take any manual actions to complete the cleanup before removing everything. The very first thing to do is to look at the logs (cf. section above) to identify what has failed and what are the actions to take (for instance, should I delete the target pod to totally remove the disruption?). The disruption will be kept in this state while there are failed chaos pods. To completely remove a chaos pod, you must remove any finalizers it holds by using one of the following methods.

### I want to remove a single chaos pod

```sh
kubectl -n <NAMESPACE> patch pod <POD_NAME> --type=json -p '[{"op": "remove", "path": "/metadata/finalizers"}]'
kubectl -n <NAMESPACE> delete pod <POD_NAME>
```

### I want to remove all chaos pods for a given disruption

```sh
NAMESPACE=<NAMESPACE> DISRUPTION=<DISRUPTION_NAME> DISRUPTION_NAMESPACE=<DISRUPTION_NAMESPACE>; kubectl -n ${NAMESPACE} get -ojson pods -l chaos.datadoghq.com/disruption-name=${DISRUPTION_NAMESPACE} -l chaos.datadoghq.com/disruption-namespace=<DISRUPTION_NAMESPACE> | jq -r '.items[].metadata.name' | xargs -I{} kubectl -n ${NAMESPACE} patch pod {} --type=json -p '[{"op": "remove", "path": "/metadata/finalizers"}]'
NAMESPACE=<NAMESPACE> DISRUPTION=<DISRUPTION_NAME> DISRUPTION_NAMESPACE=<DISRUPTION_NAMESPACE>; kubectl -n ${NAMESPACE} get -ojson pods -l chaos.datadoghq.com/disruption-name=${DISRUPTION_NAMESPACE} -l chaos.datadoghq.com/disruption-namespace=<DISRUPTION_NAMESPACE> | jq -r '.items[].metadata.name' | xargs -I{} kubectl -n ${NAMESPACE} delete pod {}
```

**Note: the chaos pods deletion can be stuck for some reason, like Kubernetes not being able to delete them. In this case, you might also want to remove the finalizer on the disruption resource itself which will then trigger the garbage collection of all related resources (including chaos pods) by Kubernetes.**

```
kubectl -n <NAMESPACE> patch disruption <DISRUPTION_NAME> --type=json -p '[{"op": "remove", "path": "/metadata/finalizers"}]'
```

## The controller fails to watch or list disruptions

If you see the following error in controller logs, it is probably because of a malformed label selector:

```
Failed to list *v1beta1.Disruption: v1beta1.DisruptionList.ListMeta: v1.ListMeta.TypeMeta: Kind: Items: []v1beta1.Disruption: v1beta1.Disruption.Spec: v1beta1.DisruptionSpec.Selector: ReadString: expects " or n, but found 1, error found in #10 byte of ...|o","foo":1}}}],"kind|..., bigger context ...|"protocol":"tcp"},"selector":{"app":"demo","foo":1}}}],"kind":"DisruptionList","metadata":{"continue|...
```

Label selectors values should always be string (quoted).
