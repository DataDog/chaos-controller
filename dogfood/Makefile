.PHONY: dogfood-go-client dogfood-go-server restart-client restart-server restart

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Image URL to use all building/pushing image targets
CLIENT_IMAGE ?= docker.io/chaos-dogfood-client:latest
SERVER_IMAGE ?= docker.io/chaos-dogfood-server:latest

# Build docker image
minikube-build-dogfood: minikube-ssh-host minikube-build-dogfood-client minikube-build-dogfood-server

minikube-ssh-host:
	ssh-keygen -R $(shell minikube ip)

dogfood-go-client:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/built_go_client ./client

dogfood-docker-client:
	docker build -t ${CLIENT_IMAGE} -f client/Dockerfile ./bin/

minikube-build-dogfood-client: dogfood-go-client dogfood-docker-client
	minikube image load --daemon=false --overwrite=true ${CLIENT_IMAGE}

dogfood-go-server:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/built_go_server ./server

dogfood-docker-server:
	docker build -t ${SERVER_IMAGE} -f server/Dockerfile ./bin/

minikube-build-dogfood-server: dogfood-go-server dogfood-docker-server
	minikube image load --daemon=false --overwrite=true ${SERVER_IMAGE}

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
