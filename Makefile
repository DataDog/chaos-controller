.PHONY: manager injector handler release
.SILENT: release

# Image URL to use all building/pushing image targets
MANAGER_IMAGE ?= chaos-controller:latest
INJECTOR_IMAGE ?= chaos-injector:latest
HANDLER_IMAGE ?= chaos-handler:latest

DOCKER_REGISTRY_PREFIX ?= docker.io
# Colima requires to have images built on a specific namespace to be shared to the Kubernetes cluster when using containerd runtime
# https://github.com/abiosoft/colima#interacting-with-image-registry
CONTAINERD_REGISTRY_PREFIX ?= k8s.io

OS_ARCH=amd64
ifeq (arm,$(shell uname -p))
OS_ARCH=arm64
endif

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager injector handler

# Run unit tests
test: generate manifests
	go test $(shell go list ./... | grep -v chaos-controller/controllers) -coverprofile cover.out

# Run e2e tests (against a real cluster)
e2e-test: generate minikube-install
	USE_EXISTING_CLUSTER=true go test ./controllers/... -coverprofile cover.out

# Build manager binary
manager: generate
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/manager/manager_amd64 main.go
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o bin/manager/manager_arm64 main.go

# Build injector binary
injector:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/injector/injector_amd64 ./cli/injector/
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o bin/injector/injector_arm64 ./cli/injector/

# Build handler binary
handler:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/handler/handler_amd64 ./cli/handler/
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o bin/handler/handler_arm64 ./cli/handler/

# Build chaosli
chaosli:
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-X github.com/DataDog/chaos-controller/cli/chaosli/cmd.Version=$(VERSION)" -o bin/chaosli/chaosli_darwin_amd64 ./cli/chaosli/

# Test chaosli API portability
chaosli-test:
	docker build -f ./cli/chaosli/chaosli.DOCKERFILE -t test-chaosli-image .

# Install CRDs and controller into a minikube cluster
minikube-install: manifests
	helm template --set controller.enableSafeguards=false ./chart | minikube kubectl -- apply -f -

# Install CRDs and controller into a colima k3s cluster
# In order to use already built images inside the containerd runtime
# we override images for all of our components to the expected namespace
colima-install: manifests
	helm template \
		--set controller.enableSafeguards=false \
		--set images.controller=${CONTAINERD_REGISTRY_PREFIX}/${MANAGER_IMAGE} \
		--set images.injector=${CONTAINERD_REGISTRY_PREFIX}/${INJECTOR_IMAGE} \
		--set images.handler=${CONTAINERD_REGISTRY_PREFIX}/${HANDLER_IMAGE} \
		./chart | kubectl apply -f -

# Uninstall CRDs and controller from a minikube cluster
minikube-uninstall: manifests
	helm template ./chart | minikube kubectl -- delete -f -

# Uninstall CRDs and controller from a colima k3s cluster
colima-uninstall:
	helm template ./chart | kubectl delete -f -

restart:
	minikube kubectl -- -n chaos-engineering rollout restart deployment chaos-controller

colima-restart:
	kubectl -n chaos-engineering rollout restart deployment chaos-controller

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) rbac:roleName=chaos-controller-role crd:crdVersions=v1 paths="./..." output:crd:dir=./chart/templates/crds/ output:rbac:dir=./chart/templates/

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Run golangci-lint against code
lint:
	golangci-lint run --timeout 5m0s

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths="./..."

colima-build-manager: manager
	nerdctl build --namespace ${CONTAINERD_REGISTRY_PREFIX} --build-arg TARGETARCH=${OS_ARCH} -t ${CONTAINERD_REGISTRY_PREFIX}/${MANAGER_IMAGE} -f bin/manager/Dockerfile ./bin/manager/

colima-build-injector: injector
	nerdctl build --namespace ${CONTAINERD_REGISTRY_PREFIX} --build-arg TARGETARCH=${OS_ARCH} -t ${CONTAINERD_REGISTRY_PREFIX}/${INJECTOR_IMAGE} -f bin/injector/Dockerfile ./bin/injector/

colima-build-handler: handler
	nerdctl build --namespace ${CONTAINERD_REGISTRY_PREFIX} --build-arg TARGETARCH=${OS_ARCH} -t ${CONTAINERD_REGISTRY_PREFIX}/${HANDLER_IMAGE} -f bin/handler/Dockerfile ./bin/handler/

colima-build: colima-build-manager colima-build-injector colima-build-handler

# Build the docker images
# to ease minikube/colima base minikube docker image load, we are using tarball
# we encountered issue while using docker+containerd on colima to load images from docker to containerd
minikube-build-manager: manager
	docker build --build-arg TARGETARCH=${OS_ARCH} -t ${DOCKER_REGISTRY_PREFIX}/${MANAGER_IMAGE} -f bin/manager/Dockerfile ./bin/manager/
	docker save -o ./bin/manager/manager.tar ${DOCKER_REGISTRY_PREFIX}/${MANAGER_IMAGE}
	minikube image load --daemon=false --overwrite=true ./bin/manager/manager.tar

minikube-build-injector: injector
	docker build --build-arg TARGETARCH=${OS_ARCH} -t ${DOCKER_REGISTRY_PREFIX}/${INJECTOR_IMAGE} -f bin/injector/Dockerfile ./bin/injector/
	docker save -o ./bin/injector/injector.tar ${DOCKER_REGISTRY_PREFIX}/${INJECTOR_IMAGE}
	minikube image load --daemon=false --overwrite=true ./bin/injector/injector.tar

minikube-build-handler: handler
	docker build --build-arg TARGETARCH=${OS_ARCH} -t ${DOCKER_REGISTRY_PREFIX}/${HANDLER_IMAGE} -f bin/handler/Dockerfile ./bin/handler/
	docker save -o ./bin/handler/handler.tar ${DOCKER_REGISTRY_PREFIX}/${HANDLER_IMAGE}
	minikube image load --daemon=false --overwrite=true ./bin/handler/handler.tar

minikube-build: minikube-build-manager minikube-build-injector minikube-build-handler

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.7.0 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

minikube-memory := 4096
minikube-start-big: minikube-memory := 8192
minikube-start-big: minikube-start

container-runtime := containerd
minikube-start-docker: container-runtime := docker
minikube-start-docker: minikube-start

colima-stop:
	colima stop
	colima stop --profile docker

colima-start:
	colima start --runtime containerd --kubernetes --cpu 4 --memory 4 --disk 50

colima-install-nerdctl:
	sudo colima nerdctl install

colima-start-docker:
	colima start --profile docker --cpu 5 --memory 5 --disk 60

minikube-start:
	minikube start \
		--driver=virtualbox \
		--container-runtime=${container-runtime} \
		--memory=${minikube-memory} \
		--cpus=4 \
		--kubernetes-version=1.22.10 \
		--disk-size=50GB \
		--extra-config=apiserver.enable-admission-plugins=NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,DefaultTolerationSeconds,NodeRestriction,MutatingAdmissionWebhook,ValidatingAdmissionWebhook,ResourceQuota \
		--iso-url=https://public-chaos-controller.s3.amazonaws.com/minikube/minikube-2021-01-18.iso

venv:
	test -d .venv || python3 -m venv .venv
	source .venv/bin/activate; pip install -qr tasks/requirements.txt

header-check: venv
	source .venv/bin/activate; inv header-check

license-check: venv
	source .venv/bin/activate; inv license-check

godeps:
	go mod tidy; go mod vendor

deps: godeps license-check

install-protobuf-macos:
	PROTOC_VERSION=3.17.3
	PROTOC_ZIP=protoc-$PROTOC_VERSION-osx-x86_64.zip
	curl -OL https://github.com/protocolbuffers/protobuf/releases/download/v$PROTOC_VERSION/$PROTOC_ZIP
	sudo unzip -o $PROTOC_ZIP -d /usr/local bin/protoc
	sudo unzip -o $PROTOC_ZIP -d /usr/local 'include/*'
	rm -f $PROTOC_ZIP

generate-disruptionlistener-protobuf:
	cd grpc/disruptionlistener && \
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.27.1 && \
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1.0 && \
	protoc --proto_path=. --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative disruptionlistener.proto

generate-chaosdogfood-protobuf:
	cd dogfood/chaosdogfood && \
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.27.1 && \
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1.0 && \
	protoc --proto_path=. --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative chaosdogfood.proto

release:
	VERSION=$(VERSION) ./tasks/release.sh
