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

Sample `kubectl` output:
```
NAME                                    READY   STATUS        RESTARTS   AGE
chaos-dogfood-client-59bccfd49c-rn5wl   1/1     Running       0          41s
chaos-dogfood-server-854cc5f49d-gjbnc   1/1     Running       0          4s
```

#### Sample client logs:
`chaos-controller/dogfood >>  kubectl -n chaos-demo logs chaos-dogfood-client-84596b6c5-8kdxl`

Might output:
```
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
#### Sample server logs:
`chaos-controller/dogfood >>  kubectl -n chaos-demo logs chaos-dogfood-server-5fdcff889f-hblj2`
Might output:
```
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

## Clean up
* Run `make uninstall` to `kubectl delete` both charts as well as remove the namespace.
