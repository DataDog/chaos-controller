# Installing protoc

Run `brew install protobuf` or `make install-protobuf-macos`

# Running the chaos dogfood server & client locally using Minikube

## Setup

### (0) Ensure Minikube is running

From the root directory, make sure you have already setup Minikube:
`chaos-controller >>  make minikube-start`

### (1) Build the gRPC client and server images

Go into the `dogfood` directory to use its `Makefile`, and build the two images:
`chaos-controller >>  cd dogfood`
`chaos-controller/dogfood >>  make minikube-build-dogfood`

They will be pushed your local docker repository as `docker.io/library/chaos-dogfood-client` & `docker.io/library/chaos-dogfood-server`.

### (2) Deploy a gRPC client and server to Minikube

Create the `chaos-demo` namespace (if necessary) and `kubectl apply` both Helm charts with this target:
`chaos-controller/dogfood >>  make install`

## Development

### (1) See your pods

Get pod name (such as `chaos-dogfood-client-84596b6c5-8kdxl` or `chaos-dogfood-server-5fdcff889f-hblj2`):
`chaos-controller/dogfood >>  kubectl -n chaos-demo get pods -o wide`

#### Sample client output:
`chaos-controller/dogfood >>  kubectl -n chaos-demo logs chaos-dogfood-client-84596b6c5-8kdxl`

Might output:
```
connecting to chaos-dogfood-server.chaos-demo.svc.cluster.local:50051...
x
| got catalog: 0 items returned
| ordered: Mock Reply
x
| got catalog: 0 items returned
| ordered: Mock Reply
x
| got catalog: 0 items returned
| ordered: Mock Reply
x
```
#### Sample client output:
`chaos-controller/dogfood >>  kubectl -n chaos-demo logs chaos-dogfood-server-5fdcff889f-hblj2`
Might output:
```
listening on :50051...
x
| catalog delivered
| cat food ordered
x
| catalog delivered
| cat food ordered
x
| catalog delivered
| cat food ordered
x
```

## Clean up
* Run `make uninstall` to `kubectl delete` both charts as well as remove the namespace.

## Advanced

### Testing code changes

* `make minikube-build-dogfood` to rebuild both client and server iamges.
  * `make minikube-build-dogfood-client` to just build client.
  * `make minikube-build-dogfood-server` to just build server.
 
* `make install` to apply recent code changes or Helm chart changes.
* `make restart` to pick up changes by recreating the pods.
  * `make restart-client` to only recreate the client pod.
  * `make restart-server` to only recreate the server pod.

### Testing Helm chart changes

* `make install` to apply recent code changes or Helm chart changes.
* `make restart` to pick up changes by recreating the pods.

If problem persists:
* `make uninstall`
* `make install`

# Applying gRPC disruption
Coming soon
