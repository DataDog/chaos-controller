# Installing protoc

Run `brew install protobuf` or `make install-protobuf-macos`

# Running chaos dogfood server & client in order to test gRPC disruption

To locally run the dogfood server, call `go run grpcdogfood/server/dogfood_server.go` from the root directory of this project.

Expected output:
```
connecting to localhost:50051...
```

To locally run the dogfood client, call `go run grpcdogfood/client/dogfood_client.go` from the root directory of this project.

Expected output:
```
listening on localhost:50051...
| got catalog: 0 items returned
| ordered: Mock Reply
| got catalog: 0 items returned
| ordered: Mock Reply
| got catalog: 0 items returned
| ordered: Mock Reply
| got catalog: 0 items returned
| ordered: Mock Reply
| got catalog: 0 items returned
| ordered: Mock Reply
```

# Running containerized chaos dogfood server & client
Coming soon

# Applying gRPC disruption
Coming soon
