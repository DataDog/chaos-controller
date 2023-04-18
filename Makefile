.PHONY: manager injector handler release
.SILENT: release

# Lima requires to have images built on a specific namespace to be shared to the Kubernetes cluster when using containerd runtime
# https://github.com/abiosoft/colima#interacting-with-image-registry
CONTAINERD_REGISTRY_PREFIX ?= k8s.io

# Image URL to use all building/pushing image targets
MANAGER_IMAGE ?= ${CONTAINERD_REGISTRY_PREFIX}/chaos-controller:latest
INJECTOR_IMAGE ?= ${CONTAINERD_REGISTRY_PREFIX}/chaos-injector:latest
HANDLER_IMAGE ?= ${CONTAINERD_REGISTRY_PREFIX}/chaos-handler:latest

LIMA_PROFILE ?= lima
LIMA_CONFIG ?= lima
KUBECTL ?= limactl shell default sudo kubectl
UNZIP_BINARY ?= sudo unzip
KUBERNETES_VERSION ?= v1.26.0
GOLANGCI_LINT_VERSION ?= 1.45.2

# expired disruption gc delay enable to speed up chaos controller disruption removal for e2e testing
# it's used to check if disruptions are deleted as expected as soon as the expiration delay occurs
EXPIRED_DISRUPTION_GC_DELAY ?= 10m

OS_ARCH=amd64
ifeq (arm64,$(shell uname -m))
OS_ARCH=arm64
endif

LIMA_CGROUPS=v1
ifeq (v2,$(CGROUPS))
LIMA_CGROUPS=v2
endif

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Build actions
all: manager injector handler

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
	GOOS=darwin GOARCH=${OS_ARCH} CGO_ENABLED=0 go build -ldflags="-X github.com/DataDog/chaos-controller/cli/chaosli/cmd.Version=$(VERSION)" -o bin/chaosli/chaosli_darwin_${OS_ARCH} ./cli/chaosli/

# Tests & CI
## Run unit tests
test: generate manifests
	CGO_ENABLED=1 go test -race $(shell go list ./... | grep -v chaos-controller/controllers) -coverprofile cover.out

spellcheck-deps:
ifeq (, $(shell which npm))
	@{\
		echo "please install npm or run 'make spellcheck-docker' for a slow but platform-agnistic run" ;\
		exit 1 ;\
	}
endif
ifeq (, $(shell which mdspell))
	@{\
		echo "installing mdspell through npm -g... (might require sudo run)" ;\
		npm -g i markdown-spellcheck ;\
	}
endif

spellcheck: spellcheck-deps
	mdspell --en-us --ignore-acronyms --ignore-numbers \
		$(shell find . -name vendor -prune -o -name '*.md' -print);

spellcheck-report: spellcheck-deps
	mdspell --en-us --ignore-acronyms --ignore-numbers --report \
		$(shell find . -name vendor -prune -o -name '*.md' -print);

spellcheck-docker:
	docker run --rm -ti -v $(shell pwd):/workdir tmaier/markdown-spellcheck:latest --ignore-numbers --ignore-acronyms --en-us \
		$(shell find . -name vendor -prune -o -name '*.md' -print);

spellcheck-format-spelling:
	sort < .spelling | sort | uniq | grep -v '^-' | tee .spelling.tmp > /dev/null && mv .spelling.tmp .spelling

## This target is dedicated for CI and aims to reuse the Kubernetes version defined here as the source of truth
ci-install-minikube:
	curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube_latest_amd64.deb
	sudo dpkg -i minikube_latest_amd64.deb
	minikube start --vm-driver=docker --container-runtime=containerd --kubernetes-version=${KUBERNETES_VERSION}
	minikube status

## Run e2e tests (against a real cluster)
e2e-test: generate
	$(MAKE) lima-install EXPIRED_DISRUPTION_GC_DELAY=10s
	USE_EXISTING_CLUSTER=true CGO_ENABLED=1 go test -race ./controllers/... -coverprofile cover.out

# Test chaosli API portability
chaosli-test:
	docker build -f ./cli/chaosli/chaosli.DOCKERFILE -t test-chaosli-image .

# Go actions
## Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) rbac:roleName=chaos-controller-role crd:crdVersions=v1 paths="./..." output:crd:dir=./chart/templates/crds/ output:rbac:dir=./chart/templates/

## Run go fmt against code
fmt:
	go fmt ./...

## Run go vet against code
vet:
	go vet ./...

## install golangci-lint at the correct version if not
lint-deps:
ifeq (, $(shell which golangci-lint))
	@{ \
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v${GOLANGCI_LINT_VERSION} ;\
	}
else
ifneq (${GOLANGCI_LINT_VERSION}, $(shell golangci-lint version --format short))
	@{ \
	echo "warning: this projects expects golangci-lint v${GOLANGCI_LINT_VERSION}; it's currently at version" $(shell golangci-lint version --format short) ;\
	echo "\nresult may differ from CI; fix this by uninstalling golangci-lint and run 'make lint' again\n";\
	}
endif
endif

## Run golangci-lint against code
lint: lint-deps
	golangci-lint run

## Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths="./..."

# Lima actions
## Create a new lima cluster and deploy the chaos-controller into it
lima-all: lima-start lima-install-cert-manager lima-build-all lima-install
	kubens chaos-engineering

## Rebuild the chaos-controller images, re-install the chart and restart the chaos-controller pods
lima-redeploy: lima-build-all lima-install lima-restart

## Install cert-manager chart
lima-install-cert-manager:
	$(KUBECTL) apply -f https://github.com/jetstack/cert-manager/releases/download/v1.9.1/cert-manager.yaml
	$(KUBECTL) -n cert-manager rollout status deployment/cert-manager-webhook --timeout=180s

lima-install-demo:
	kubectl apply -f ./examples/namespace.yaml
	kubectl apply -f ./examples/demo.yaml
	$(KUBECTL) -n chaos-demo rollout status deployment/demo-curl --timeout=60s
	$(KUBECTL) -n chaos-demo rollout status deployment/demo-nginx --timeout=60s


## Install CRDs and controller into a lima k3s cluster
## In order to use already built images inside the containerd runtime
## we override images for all of our components to the expected namespace
lima-install: manifests
	helm template \
		--set controller.enableSafeguards=false \
		--set controller.expiredDisruptionGCDelay=${EXPIRED_DISRUPTION_GC_DELAY} \
		--values ./chart/values/dev.yaml \
		./chart | $(KUBECTL) apply -f -
	$(KUBECTL) -n chaos-engineering rollout status deployment/chaos-controller --timeout=60s

## Uninstall CRDs and controller from a lima k3s cluster
lima-uninstall:
	helm template ./chart | $(KUBECTL) delete -f -

## Restart the chaos-controller pod
lima-restart:
	$(KUBECTL) -n chaos-engineering rollout restart deployment chaos-controller

## Build all images and import them in lima
lima-build-all: lima-build-manager lima-build-injector lima-build-handler

docker-build-manager: manager
	docker build --build-arg TARGETARCH=${OS_ARCH} -t ${MANAGER_IMAGE} -f bin/manager/Dockerfile ./bin/manager/
	docker save ${MANAGER_IMAGE} -o ./bin/manager/manager.tar.gz

docker-build-ebpf:
	docker build --platform linux/${OS_ARCH} --build-arg ARCH=${OS_ARCH} -t ebpf-builder-${OS_ARCH} -f bin/ebpf-builder/Dockerfile ./bin/ebpf-builder/
	rm -r bin/injector/ebpf/ || true
	docker run --rm -v ${shell pwd}:/app ebpf-builder-${OS_ARCH}

docker-build-injector: docker-build-ebpf injector
	docker build --build-arg TARGETARCH=${OS_ARCH} -t ${INJECTOR_IMAGE} -f bin/injector/Dockerfile ./bin/injector/
	docker save ${INJECTOR_IMAGE} -o ./bin/injector/injector.tar.gz

docker-build-handler: handler
	docker build --build-arg TARGETARCH=${OS_ARCH} -t ${HANDLER_IMAGE} -f bin/handler/Dockerfile ./bin/handler/
	docker save ${HANDLER_IMAGE} -o ./bin/handler/handler.tar.gz

## Build and import the manager image in lima
lima-build-manager: docker-build-manager
	limactl copy ./bin/manager/manager.tar.gz default:/tmp/
	limactl shell default -- sudo k3s ctr i import /tmp/manager.tar.gz

## Build and import the injector image in lima
lima-build-injector: docker-build-injector
	limactl copy ./bin/injector/injector.tar.gz default:/tmp/
	limactl shell default -- sudo k3s ctr i import /tmp/injector.tar.gz

## Build and import the handler image in lima
lima-build-handler: docker-build-handler
	limactl copy ./bin/handler/handler.tar.gz default:/tmp/
	limactl shell default -- sudo k3s ctr i import /tmp/handler.tar.gz

## Remove lima references from kubectl config
lima-kubectx-clean:
	-kubectl config delete-cluster ${LIMA_PROFILE}
	-kubectl config delete-context ${LIMA_PROFILE}
	-kubectl config delete-user ${LIMA_PROFILE}
	kubectl config unset current-context

lima-kubectx:
	limactl shell default sudo sed 's/default/lima/g' /etc/rancher/k3s/k3s.yaml >> ~/.kube/config_lima
	KUBECONFIG=${KUBECONFIG}:~/.kube/config:~/.kube/config_lima kubectl config view --flatten > /tmp/config
	rm ~/.kube/config_lima
	mv /tmp/config ~/.kube/config
	chmod 600 ~/.kube/config
	kubectx ${LIMA_PROFILE}

## Stop and delete the lima cluster
lima-stop:
	limactl stop -f default
	limactl delete default
	$(MAKE) lima-kubectx-clean

## Start the lima cluster, pre-cleaning kubectl config
lima-start: lima-kubectx-clean
	LIMA_CGROUPS=${LIMA_CGROUPS} LIMA_CONFIG=${LIMA_CONFIG} ./scripts/lima_start.sh
	$(MAKE) lima-kubectx

# Longhorn is used as an alternative StorageClass in order to enable "reliable" disk throttling accross various local setup
# It aims to bypass some issues encountered with default StorageClass (local-path --> tmpfs) that led to virtual unnamed devices
# unnamed devices are linked to 0 as a major device identifier, that blkio does not support
# https://super-unix.com/unixlinux/can-you-throttle-the-bandwidth-to-a-tmpfs-based-ramdisk/
lima-install-longhorn:
	$(KUBECTL) apply -f https://raw.githubusercontent.com/longhorn/longhorn/v1.4.0/deploy/longhorn.yaml

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.11.3 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

# CI-specific actions

## Minikube builds for e2e tests
minikube-build-all: minikube-build-manager minikube-build-injector minikube-build-handler

minikube-build-manager: docker-build-manager
	minikube image load --daemon=false --overwrite=true ./bin/manager/manager.tar.gz

minikube-build-injector: docker-build-injector
	minikube image load --daemon=false --overwrite=true ./bin/injector/injector.tar.gz

minikube-build-handler: docker-build-handler
	minikube image load --daemon=false --overwrite=true ./bin/handler/handler.tar.gz

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

PROTOC_VERSION = 3.17.3
PROTOC_OS ?= osx
PROTOC_ZIP = protoc-${PROTOC_VERSION}-${PROTOC_OS}-x86_64.zip

install-protobuf:
	curl -OL https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/${PROTOC_ZIP}
	$(UNZIP_BINARY) -o ${PROTOC_ZIP} -d /usr/local bin/protoc
	$(UNZIP_BINARY) -o ${PROTOC_ZIP} -d /usr/local 'include/*'
	rm -f ${PROTOC_ZIP}

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

generate-mock:
	go install github.com/vektra/mockery/v2@v2.24.0
	go generate ./...
# First re-generate header, it should complain as just (re)generated mocks does not contains them
	-$(MAKE) header-check
# Then, re-generate header, it should succeed as now all files contains headers as expected, and command return with an happy exit code
	$(MAKE) header-check

release:
	VERSION=$(VERSION) ./tasks/release.sh
