# Contributing to Chaos Dogfood Application

See [dogfood instructions](README.md) to get the dogfood application running locally.
This tutorial assumes you are in the [dogfood/](/dogfood) directory.

## Testing code changes

- `make colima-build-dogfood` to rebuild both client and server iamges.

  - `make colima-build-dogfood-client` to just build client.
  - `make colima-build-dogfood-server` to just build server.

- `make install` to apply recent code changes or Helm chart changes.
- `make restart` to pick up changes by recreating the pods.
  - `make restart-client` to only recreate the client pod.
  - `make restart-server` to only recreate the server pod.

## Testing Helm chart changes

- `make install` to apply recent code changes or Helm chart changes.
- `make restart` to pick up changes by recreating the pods.

If your changes don't seem to propagate, you can:

- `make uninstall` and `make install`
  or move to the top level directory and run
- `colima delete` and `make colima-start` and redo [dogfood instructions](README.md)

## Testing Datadog Metrics

To apply the datadog agent to your local colima environment, run the following:

```
kubectl apply -f "https://raw.githubusercontent.com/DataDog/datadog-agent/master/Dockerfiles/manifests/rbac/clusterrole.yaml"

kubectl apply -f "https://raw.githubusercontent.com/DataDog/datadog-agent/master/Dockerfiles/manifests/rbac/serviceaccount.yaml"

kubectl apply -f "https://raw.githubusercontent.com/DataDog/datadog-agent/master/Dockerfiles/manifests/rbac/clusterrolebinding.yaml"
```

Then take a look at the file `datadog-agent-all-features.yaml` (Feel free to remove the SECURITY feature as it is 
unnecessary for testing). You will notice that an api key AND a random string encoded in base64 is required. Get yourself
an API key from your Datadog site, think of a random string, then do the following:

```
echo -n '<Your API key>' | base64
# Copy the encoding and paste it where needed in the datadog.yaml
echo -n 'Random string' | base64
# Copy the encoding and paste it where needed in the datadog.yaml
```

By default the Datadog site is set to the US site datadoghq.com. If you're using other sites, you may want to edit the
`DD_SITE` environment variable accordingly.

Deploy the Daemonset:
```
kubectl apply -f datadog-agent-all-features.yaml
```

Verify it is running correctly using `kubectl get daemonset` in the appropriate namespace (`default` is the default)

Once you've verified the daemonset is up and running, you'll need to get Kubernetes State Metrics with the following steps:
1. Download the kube-state manifests folder [here](https://github.com/kubernetes/kube-state-metrics/tree/master/examples/standard).
2. `kubectl apply -f <NAME_OF_THE_KUBE_STATE_MANIFESTS_FOLDER>`

Then you should be set to see metrics for the client and server containers.
