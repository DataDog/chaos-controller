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
