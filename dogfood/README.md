 # Installing protoc

Run `brew install protobuf` or `make install-protobuf`

## Running the chaos dogfood server & client locally using Colima

### Setup

#### (0) Ensure Colima is running

From the root directory, make sure you have already setup Colima:
`chaos-controller >> make colima-start`

#### (1) Build the gRPC client and server images

Go into the `dogfood` directory to use its `Makefile`, and build the two images:
`chaos-controller >> cd dogfood`
`chaos-controller/dogfood >> make colima-build-dogfood`

They will be pushed your local docker repository as `k8s.io/chaos-dogfood-client` & `k8s.io/chaos-dogfood-server`.

#### (2) Deploy a gRPC client and server to Colima

Create the `chaos-demo` namespace (if necessary) and `kubectl apply` both Helm charts with this target:
`chaos-controller/dogfood >> make install`

### Development

#### (3) See your pods

Get pod name (such as `chaos-dogfood-client-84596b6c5-8kdxl` or `chaos-dogfood-server-5fdcff889f-hblj2`):
`chaos-controller/dogfood >> kubectl -n chaos-demo get pods -o wide`

Sample `kubectl` output:

```bash
NAME                                    READY   STATUS        RESTARTS   AGE
chaos-dogfood-client-59bccfd49c-rn5wl   1/1     Running       0          41s
chaos-dogfood-server-854cc5f49d-gjbnc   1/1     Running       0          4s
```

##### Sample client logs

`chaos-controller/dogfood >> kubectl -n chaos-demo logs -l app=chaos-dogfood-client`

Might output:

```bash
connecting to chaos-dogfood-server.chaos-demo.svc.cluster.local:50051...
x
| catalog: 0 items returned ()
| ERROR ordering food: rpc error: code = Unavailable desc = connection error: desc = "transport: Error while dialing dial tcp 10.96.24.54:50051: connect: connection refused"
| ordered:
x
| catalog: 3 items returned (cat, dog, cow)
| ordered: Meowmix is on its way!
x
| catalog: 3 items returned (cow, cat, dog)
| ERROR ordering food: rpc error: code = Unknown desc = Sorry, we don't deliver food for your mouse =(
| ordered:
x
| catalog: 3 items returned (cow, cat, dog)
| ordered: Chewey is on its way!
x
| catalog: 3 items returned (cat, dog, cow)
| ordered: Meowmix is on its way!
x
| catalog: 3 items returned (cat, dog, cow)
| ERROR ordering food: rpc error: code = Unknown desc = Sorry, we don't deliver food for your mouse =(
| ordered:
x
```

##### Sample server logs

`chaos-controller/dogfood >> kubectl -n chaos-demo logs -l app=chaos-dogfood-server`
Might output:

```bash
listening on :50050...
x
| returned catalog
| proccessed order - animal:"cat"
x
| returned catalog
| * DECLINED ORDER - animal:"mouse"
x
| returned catalog
| proccessed order - animal:"dog"
x
| returned catalog
| proccessed order - animal:"cat"
x
| returned catalog
| * DECLINED ORDER - animal:"mouse"
x
| returned catalog
| proccessed order - animal:"dog"
x
```

#### (4) Apply your disruptions

You can `kubectl apply -f examples/<disruption.yaml>` for any `example/` disruption files.
For gRPC disruption, you can follow these [detailed steps](../docs/grpc_disruption/demo_instructions.md).

### Sending Metrics to Datadog

For the purposes of testing disruptions/workflows, you should make sure that the datadog agent is properly installed
on the cluster that the client and server are running on. 3 of the major disruptive resources properly send metrics
to Datadog (CPU, Network, Disk). The client contains computation related to these disruptions and can be tested using
the disruptions mentioned.

### Clean up

- Run `make uninstall` to `kubectl delete` both charts as well as remove the namespace.
