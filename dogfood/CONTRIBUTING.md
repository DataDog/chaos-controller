# Contributing to Chaos Dogfood Application

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
