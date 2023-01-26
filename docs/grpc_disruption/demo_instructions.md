# Applying gRPC disruption to chaos dogfood demo application

:warning: To learn how to set up a gRPC disruption on your own applications, follow [these instructions](instructions.md).

### (0) Turn chaos on

Make sure `chaosEnabled` is set to `true` in [`dogfood_server.go`](../../dogfood/server/dogfood_server.go):

```
const chaosEnabled = true // In your application, make this a feature flag

var serverAddr string

func init() {
```

Make sure you have already setup chaos-controller on your local Colima by following the main [CONTRIBUTING.md](../../CONTRIBUTING.md) guidelines.

To run the dogfood application, go through the `build` and `install` instructions in [dogfood/CONTRIBUTING.md](../../dogfood/CONTRIBUTING.md). If you already have the application running, `restart` it.

### (1) Follow client logs

Get pod name (such as `chaos-dogfood-client-84596b6c5-8kdxl` or `chaos-dogfood-server-5fdcff889f-hblj2`) by running:

```
kubectl -n chaos-demo get pods -o wide
```

Follow live logs with the `--follow` option on the `logs` command:

```
kubectl -n chaos-demo logs --follow <pod name such as chaos-dogfood-client-84596b6c5-8kdxl>
```

### (2) Make dogfood application return errors

From the root of the repository, run `kubectl apply -f examples/grpc_error.yaml`.

Which applies the following `disruption` spec ([examples/grpc_error.yaml](../../examples/grpc_error.yaml)):

```
spec:
  level: pod
  selector:
    app: chaos-dogfood-server
  count: 100%
  grpc:
    port: 50050
    endpoints:
      - endpoint: /chaosdogfood.ChaosDogfood/getCatalog # gRPC service endpoint to disrupt
        error: UNAVAILABLE # gRPC error code to return instead computed response
      - endpoint: /chaosdogfood.ChaosDogfood/order # gRPC service endpoint to disrupt
        error: NOT_FOUND # gRPC error code to return instead computed response
```

To break this down:

- `app: chaos-dogfood-server` - a selector identifying which pods to disrupt (in this case, `chaos-dogfood-server-5fdcff889f-hblj2`).
- `port: 50050` - port to connect to the gRPC application on your pod (it's distinct from the port you expose on the Kubernetes `Service` which in this case would be `50051`, not `50050`).
- Since the `endpoints` do not have `query_percent`s specified, the assigned errors will be applied to all queries for the respective endpoints.

The sample client logs might look like in the moment the disruption is applied:

```
x
| catalog: 3 items returned (cow, cat, dog)
| ERROR ordering food: rpc error: code = Unknown desc = Sorry, we don't deliver food for your mouse =(
| ordered:
x
| catalog: 3 items returned (cat, dog, cow)
| ordered: Chewey is on its way!
x
| catalog: 3 items returned (cat, dog, cow)
| ordered: Meowmix is on its way!
x
| ERROR getting catalog:rpc error: code = Unavailable desc = Chaos Controller injected this error: UNAVAILABLE
| catalog: 0 items returned ()
| ERROR ordering food: rpc error: code = NotFound desc = Chaos Controller injected this error: NOT_FOUND
| ordered:
x
| ERROR getting catalog:rpc error: code = Unavailable desc = Chaos Controller injected this error: UNAVAILABLE
| catalog: 0 items returned ()
| ERROR ordering food: rpc error: code = NotFound desc = Chaos Controller injected this error: NOT_FOUND
| ordered:
x
| ERROR getting catalog:rpc error: code = Unavailable desc = Chaos Controller injected this error: UNAVAILABLE
| catalog: 0 items returned ()
| ERROR ordering food: rpc error: code = NotFound desc = Chaos Controller injected this error: NOT_FOUND
| ordered:
x
```

The first three groups of logs (where a "group" is demarcated by an `x`) are healthy responses, and second three groups are each returning the configured errors for both requests' types.

Run `kubectl delete -f examples/grpc_error.yaml` to remove the disruption. You should see the logs you are following revert back to the original responses which successfully fill orders for `dog` and `cat` but not `mouse`.

### (3) Make dogfood application return an override

From the root of the repository, run `kubectl apply -f examples/grpc_override.yaml`.

Which applies the following `disruption` spec ([examples/grpc_override.yaml](../../examples/grpc_override.yaml)):

```
spec:
  level: pod
  selector:
    app: chaos-dogfood-server
  count: 100%
  grpc:
    port: 50050
    endpoints:
      - endpoint: /chaosdogfood.ChaosDogfood/getCatalog
        override: "{}"
      - endpoint: /chaosdogfood.ChaosDogfood/order
        override: "{}"
```

The sample client logs might look like in the moment the disruption is applied:

```
x
| catalog: 3 items returned (cat, dog, cow)
| ordered: Chewey is on its way!
x
| catalog: 3 items returned (cat, dog, cow)
| ordered: Meowmix is on its way!
x
| catalog: 3 items returned (cat, dog, cow)
| ERROR ordering food: rpc error: code = Unknown desc = Sorry, we don't deliver food for your mouse =(
| ordered:
x
| catalog: 0 items returned ()
| ordered:
x
| catalog: 0 items returned ()
| ordered:
x
| catalog: 0 items returned ()
| ordered:
x
```

For now, the only override available is an empty protobuf message denoted by `"{}"`. Run `kubectl delete -f examples/grpc_override.yaml` to remove the disruption. You should see the logs you are following revert back to the original responses which successfully fill orders for `dog` and `cat` but not `mouse`.

### (4) Complex gRPC disruptions on dogfood application

From the root of the repository, run `kubectl apply -f examples/grpc.yaml`.

Which applies the following `disruption` spec ([examples/grpc.yaml](../../examples/grpc.yaml)):

```
spec:
  level: pod
  selector:
    app: chaos-dogfood-server
  count: 100%
  grpc:
    port: 50050
    endpoints:
      - endpoint: /chaosdogfood.ChaosDogfood/getCatalog
        error: NOT_FOUND
        queryPercent: 25
      - endpoint: /chaosdogfood.ChaosDogfood/getCatalog
        error: ALREADY_EXISTS
        queryPercent: 50
      - endpoint: /chaosdogfood.ChaosDogfood/getCatalog
        override: "{}"
      - endpoint: /chaosdogfood.ChaosDogfood/order # gRPC service endpoint to disrupt
        override: "{}"
        queryPercent: 50
```

The endpoints here read as follows:

- for 25% of `getCatalog` queries, return a `NOT_FOUND` error
- for 50% of `getCatalog` queries, return a `ALREADY_EXISTS` error
- for remaining `getCatalog` queries (which calculates to 25%), return an empty protobuf message
- for 50% of `order` queries, return an empty protobuf message

You can verify approximately that these proportions are honored in the logs:

```
x
| ERROR getting catalog:rpc error: code = NotFound desc = Chaos Controller injected this error: NOT_FOUND
| catalog: 0 items returned ()
| ordered: Chewey is on its way!
x
| ERROR getting catalog:rpc error: code = AlreadyExists desc = Chaos Controller injected this error: ALREADY_EXISTS
| catalog: 0 items returned ()
| ordered:
x
| catalog: 0 items returned ()
| ordered:
x
| ERROR getting catalog:rpc error: code = AlreadyExists desc = Chaos Controller injected this error: ALREADY_EXISTS
| catalog: 0 items returned ()
| ordered:
x
| ERROR getting catalog:rpc error: code = NotFound desc = Chaos Controller injected this error: NOT_FOUND
| catalog: 0 items returned ()
| ERROR ordering food: rpc error: code = Unknown desc = Sorry, we don't deliver food for your mouse =(
| ordered:
x
| ERROR getting catalog:rpc error: code = AlreadyExists desc = Chaos Controller injected this error: ALREADY_EXISTS
| catalog: 0 items returned ()
| ordered:
x
| ERROR getting catalog:rpc error: code = NotFound desc = Chaos Controller injected this error: NOT_FOUND
| catalog: 0 items returned ()
| ordered: Meowmix is on its way!
x
| ERROR getting catalog:rpc error: code = AlreadyExists desc = Chaos Controller injected this error: ALREADY_EXISTS
| catalog: 0 items returned ()
| ordered:
x
| ERROR getting catalog:rpc error: code = NotFound desc = Chaos Controller injected this error: NOT_FOUND
| catalog: 0 items returned ()
| ordered: Chewey is on its way!
x
| ERROR getting catalog:rpc error: code = AlreadyExists desc = Chaos Controller injected this error: ALREADY_EXISTS
| catalog: 0 items returned ()
| ordered: Meowmix is on its way!
x
| ERROR getting catalog:rpc error: code = AlreadyExists desc = Chaos Controller injected this error: ALREADY_EXISTS
| catalog: 0 items returned ()
| ordered:
x
| ERROR getting catalog:rpc error: code = AlreadyExists desc = Chaos Controller injected this error: ALREADY_EXISTS
| catalog: 0 items returned ()
| ordered: Chewey is on its way!
x
| catalog: 0 items returned ()
| ordered: Meowmix is on its way!
x
| ERROR getting catalog:rpc error: code = AlreadyExists desc = Chaos Controller injected this error: ALREADY_EXISTS
| catalog: 0 items returned ()
| ERROR ordering food: rpc error: code = Unknown desc = Sorry, we don't deliver food for your mouse =(
| ordered:
x
| catalog: 0 items returned ()
| ordered:
```

As usual, run `kubectl delete -f examples/grpc.yaml` to remove the disruption.
