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

NOW_ISO8601 = $(shell date -u +"%Y-%m-%dT%H:%M:%S")

GOOS = $(shell go env GOOS)
GOARCH = $(shell go env GOARCH)

# must be changed alongside BUILDGOVERSION in the main Makefile for the chaps-controller
BUILDGOVERSION = 1.22.5

# Image URL to use all building/pushing image targets
CLIENT_IMAGE ?= k8s.io/chaos-dogfood-client:latest
SERVER_IMAGE ?= k8s.io/chaos-dogfood-server:latest

# default instance name will be connected user name
LIMA_INSTANCE ?= $(shell whoami | tr "." "-")

# Build docker image
lima-push-dogfood: lima-push-dogfood-client lima-push-dogfood-server

build-client:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o bin/client/built_go_client ./client

build-server:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o bin/server/built_go_server ./server

docker-build-dogfood-client: build-client
	docker buildx build \
		--build-arg BUILDGOVERSION=$(BUILDGOVERSION) \
		--build-arg BUILDSTAMP=$(NOW_ISO8601) \
		-t $(CLIENT_IMAGE) \
		--metadata-file ./bin/client/docker-metadata.json \
		-f client/Dockerfile ./bin/client/
	docker save $(CLIENT_IMAGE) -o ./bin/client/client.tar.gz

lima-push-dogfood-client: docker-build-dogfood-client
	limactl copy ./bin/client/client.tar.gz $(LIMA_INSTANCE):/tmp/
	limactl shell $(LIMA_INSTANCE) -- sudo k3s ctr i import /tmp/client.tar.gz

docker-build-dogfood-server: build-server
	docker buildx build \
		--build-arg BUILDGOVERSION=$(BUILDGOVERSION) \
		--build-arg BUILDSTAMP=$(NOW_ISO8601) \
		-t $(SERVER_IMAGE) \
		--metadata-file ./bin/server/docker-metadata.json \
		-f server/Dockerfile ./bin/server/
	docker save $(SERVER_IMAGE) -o ./bin/server/server.tar.gz

lima-push-dogfood-server: docker-build-dogfood-server
	#nerdctl build --namespace k8s.io --build-arg TARGETARCH=${OS_ARCH} -t ${SERVER_IMAGE} -f server/Dockerfile ./bin/
	limactl copy ./bin/server/server.tar.gz $(LIMA_INSTANCE):/tmp/
	limactl shell $(LIMA_INSTANCE) -- sudo k3s ctr i import /tmp/server.tar.gz
# INSTALL

install:
	kubectl apply -f ../examples/namespace.yaml
	helm template ./client/chart | kubectl apply -f -
	helm template ./server/chart | kubectl apply -f -

uninstall:
	helm template ./client/chart | kubectl delete -f -
	helm template ./server/chart | kubectl delete -f -
	kubectl delete -f ../examples/namespace.yaml

restart-client:
	kubectl -n chaos-demo rollout restart deployment chaos-dogfood-client

restart-server:
	kubectl -n chaos-demo rollout restart deployment chaos-dogfood-server

restart: restart-client restart-server

all: lima-push-dogfood install
