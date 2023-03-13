.PHONY: dogfood-go-client dogfood-go-server restart-client restart-server restart colima-build-dogfood

OS_ARCH=amd64
ifeq (arm64,$(shell uname -m))
OS_ARCH=arm64
endif

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Image URL to use all building/pushing image targets
CLIENT_IMAGE ?= k8s.io/chaos-dogfood-client:latest
SERVER_IMAGE ?= k8s.io/chaos-dogfood-server:latest

# Build docker image
colima-build-dogfood: colima-build-dogfood-client colima-build-dogfood-server

dogfood-go-client:
	GOOS=linux GOARCH=${OS_ARCH} CGO_ENABLED=0 go build -o bin/built_go_client ./client

colima-build-dogfood-client: dogfood-go-client
	nerdctl build --namespace k8s.io --build-arg TARGETARCH=${OS_ARCH} -t ${CLIENT_IMAGE} -f client/Dockerfile ./bin/

dogfood-go-server:
	GOOS=linux GOARCH=${OS_ARCH} CGO_ENABLED=0 go build -o bin/built_go_server ./server

colima-build-dogfood-server: dogfood-go-server
	nerdctl build --namespace k8s.io --build-arg TARGETARCH=${OS_ARCH} -t ${SERVER_IMAGE} -f server/Dockerfile ./bin/

# INSTALL

install:
	kubectl apply -f ../examples/namespace.yaml
	helm template ./client/chart | kubectl apply -f -
	helm template ./server/chart | kubectl apply -f -

uninstall:
	helm template ./client/chart | kubectl delete -f -
	helm template ./server/chart | kubectl delete -f -

reinstall: uninstall install

restart-client:
	kubectl -n chaos-demo rollout restart deployment chaos-dogfood-client

restart-server:
	kubectl -n chaos-demo rollout restart deployment chaos-dogfood-server

restart: restart-client restart-server
