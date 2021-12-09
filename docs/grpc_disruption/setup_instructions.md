# Installing protoc

Run `brew install protobuf` or `make install-protobuf-macos`

# Running chaos dogfood server & client in order to test gRPC disruption

To locally run the dogfood server, call `go run grpcdogfood/server/dogfood_server.go` from the root directory of this project.

Expected output:
```
listening on localhost:50051...
```

To locally run the dogfood client, call `go run grpcdogfood/client/dogfood_client.go` from the root directory of this project.

Expected output:
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
```

If the client is connected to the correct server, your server should output corresponding updates:

```
listening on port :50051...
x
| cat food ordered
x
| catalog delivered
x
| cat food ordered
x
| catalog delivered
x
| cat food ordered
x
| catalog delivered
```

# Running containerized chaos dogfood server & client
Coming soon

# Applying gRPC disruption
Coming soon
