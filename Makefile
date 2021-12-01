.PHONY: manager injector handler release
.SILENT: release

# Image URL to use all building/pushing image targets
MANAGER_IMAGE ?= docker.io/chaos-controller:latest
INJECTOR_IMAGE ?= docker.io/chaos-injector:latest
HANDLER_IMAGE ?= docker.io/chaos-handler:latest

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
e2e-test: generate manifests
	USE_EXISTING_CLUSTER=true go test ./controllers/... -coverprofile cover.out

# Build manager binary
manager: generate
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/manager/manager main.go

# Build injector binary
injector:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/injector/injector ./cli/injector/

# Build handler binary
handler:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/handler/handler ./cli/handler/

# Build chaosli
chaosli:
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o bin/chaosli/chaosli_darwin_amd64 ./cli/chaosli/

# Install CRDs and controller into a cluster
install: manifests
	helm template --set controller.enableSafeguards=false ./chart | minikube kubectl -- apply -f -

# Uninstall CRDs and controller from a cluster
uninstall: manifests
	helm template ./chart | minikube kubectl -- delete -f -

restart:
	minikube kubectl -- -n chaos-engineering rollout restart deployment chaos-controller

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) rbac:roleName=chaos-controller-role crd:trivialVersions=true,preserveUnknownFields=false,crdVersions=v1beta1 paths="./..." output:crd:dir=./chart/templates/crds/ output:rbac:dir=./chart/templates/

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Run golangci-lint against code
lint:
	golangci-lint run --timeout 2m0s

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths="./..."

# Build the docker images
minikube-build-manager: manager
	docker build -t ${MANAGER_IMAGE} -f bin/manager/Dockerfile ./bin/manager/
	minikube image load --daemon=false --overwrite=true ${MANAGER_IMAGE}

minikube-build-injector: injector
	docker build -t ${INJECTOR_IMAGE} -f bin/injector/Dockerfile ./bin/injector/
	minikube image load --daemon=false --overwrite=true ${INJECTOR_IMAGE}

minikube-build-handler: handler
	docker build -t ${HANDLER_IMAGE} -f bin/handler/Dockerfile ./bin/handler/
	minikube image load --daemon=false --overwrite=true ${HANDLER_IMAGE}

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
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.6.2 ;\
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

minikube-start:
	minikube start \
		--vm-driver=virtualbox \
		--container-runtime=${container-runtime} \
		--memory=${minikube-memory} \
		--cpus=4 \
		--kubernetes-version=1.19.14 \
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

generate-protobuf:
	cd grpc/disruptionlistener && \
	go get google.golang.org/protobuf/cmd/protoc-gen-go@v1.27.1 && \
	go get google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1.0 && \
	protoc --proto_path=. --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative disruptionlistener.proto

release:
	VERSION=$(VERSION) ./tasks/release.sh
