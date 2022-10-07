.PHONY: manager injector handler release
.SILENT: release

# Colima requires to have images built on a specific namespace to be shared to the Kubernetes cluster when using containerd runtime
# https://github.com/abiosoft/colima#interacting-with-image-registry
CONTAINERD_REGISTRY_PREFIX ?= k8s.io

# Image URL to use all building/pushing image targets
MANAGER_IMAGE ?= ${CONTAINERD_REGISTRY_PREFIX}/chaos-controller:latest
INJECTOR_IMAGE ?= ${CONTAINERD_REGISTRY_PREFIX}/chaos-injector:latest
HANDLER_IMAGE ?= ${CONTAINERD_REGISTRY_PREFIX}/chaos-handler:latest

KUBECTL ?= kubectl
UNZIP_BINARY ?= sudo unzip
KUBERNETES_VERSION ?= v1.22.13

CLUSTER_NAME ?= colima

# expired disruption gc delay enable to speed up chaos controller disruption removal for e2e testing
# it's used to check if disruptions are deleted as expected as soon as the expiration delay occurs
EXPIRED_DISRUPTION_GC_DELAY ?= 10m

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

all: manager injector handler

# Run unit tests
test: generate manifests
	CGO_ENABLED=1 go test -race $(shell go list ./... | grep -v chaos-controller/controllers) -coverprofile cover.out

# This target is dedicated for CI and aims to reuse the Kubernetes version defined here as the source of truth
ci-install-minikube:
	curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube_latest_amd64.deb
	sudo dpkg -i minikube_latest_amd64.deb
	minikube start --vm-driver=docker --container-runtime=containerd --kubernetes-version=${KUBERNETES_VERSION}
	minikube status

# Run e2e tests (against a real cluster)
e2e-test: generate
	$(MAKE) colima-install EXPIRED_DISRUPTION_GC_DELAY=10s
	USE_EXISTING_CLUSTER=true CLUSTER_NAME=${CLUSTER_NAME} CGO_ENABLED=1 go test -race ./controllers/... -coverprofile cover.out

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

# Test chaosli API portability
chaosli-test:
	docker build -f ./cli/chaosli/chaosli.DOCKERFILE -t test-chaosli-image .

colima-all:
	$(MAKE) colima-start
	$(MAKE) install-cert-manager
#	$(MAKE) install-longhorn
	$(MAKE) colima-build
	$(MAKE) colima-install

colima-deploy: colima-build colima-install colima-restart

install-cert-manager:
	$(KUBECTL) apply -f https://github.com/jetstack/cert-manager/releases/download/v1.9.1/cert-manager.yaml

# Longhorn is used as an alternative StorageClass in order to enable "reliable" disk throttling accross various local setup
# It aims to bypass some issues encountered with default StorageClass (local-path --> tmpfs) that led to virtual unnamed devices
# unnamed devices are linked to 0 as a major device identifier, that blkio does not support
# https://super-unix.com/unixlinux/can-you-throttle-the-bandwidth-to-a-tmpfs-based-ramdisk/
install-longhorn:
# https://longhorn.io/docs/1.3.1/deploy/install/#installation-requirements
# https://longhorn.io/docs/1.3.1/advanced-resources/os-distro-specific/csi-on-k3s/
# > Longhorn relies on iscsiadm on the host to provide persistent volumes to Kubernetes
	colima ssh -- sudo apk add open-iscsi
	colima ssh -- sudo /etc/init.d/iscsid start
# https://kubernetes.io/docs/concepts/storage/volumes/#mount-propagation
# > Another feature that CSI depends on is mount propagation.
# > It allows the sharing of volumes mounted by one container with other containers in the same pod, or even to other pods on the same node
# https://github.com/longhorn/longhorn/issues/2402#issuecomment-806556931
# below directories where discovered in an empirical manner, if you encounter an error similar to:
# >  spec: failed to generate spec: path "/var/lib/longhorn/" is mounted on "/var/lib" but it is not a shared mount
# you may want to add the directory mentioned in the error below
	colima ssh -- sudo mount --make-rshared /var/lib/
	colima ssh -- sudo mount --make-rshared /
	$(KUBECTL) apply -f https://raw.githubusercontent.com/longhorn/longhorn/v1.3.1/deploy/longhorn.yaml

# Install CRDs and controller into a colima k3s cluster
# In order to use already built images inside the containerd runtime
# we override images for all of our components to the expected namespace
colima-install: manifests
	helm template \
		--set controller.enableSafeguards=false \
		--set controller.expiredDisruptionGCDelay=${EXPIRED_DISRUPTION_GC_DELAY} \
		./chart | $(KUBECTL) apply -f -

# Uninstall CRDs and controller from a colima k3s cluster
colima-uninstall:
	helm template ./chart | $(KUBECTL) delete -f -

restart:
	$(KUBECTL) -n chaos-engineering rollout restart deployment chaos-controller

colima-restart: restart

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
	nerdctl build --namespace ${CONTAINERD_REGISTRY_PREFIX} --build-arg TARGETARCH=${OS_ARCH} -t ${MANAGER_IMAGE} -f bin/manager/Dockerfile ./bin/manager/

colima-build-injector: injector
	nerdctl build --namespace ${CONTAINERD_REGISTRY_PREFIX} --build-arg TARGETARCH=${OS_ARCH} -t ${INJECTOR_IMAGE} -f bin/injector/Dockerfile ./bin/injector/

colima-build-handler: handler
	nerdctl build --namespace ${CONTAINERD_REGISTRY_PREFIX} --build-arg TARGETARCH=${OS_ARCH} -t ${HANDLER_IMAGE} -f bin/handler/Dockerfile ./bin/handler/

colima-build: colima-build-manager colima-build-injector colima-build-handler

# Build the docker images
# to ease minikube/colima base minikube docker image load, we are using tarball
# we encountered issue while using docker+containerd on colima to load images from docker to containerd
# this Make target are used by the CI
minikube-build-manager: manager
	docker build --build-arg TARGETARCH=${OS_ARCH} -t ${MANAGER_IMAGE} -f bin/manager/Dockerfile ./bin/manager/
	docker save -o ./bin/manager/manager.tar ${MANAGER_IMAGE}
	minikube image load --daemon=false --overwrite=true ./bin/manager/manager.tar

minikube-build-injector: injector
	docker build --build-arg TARGETARCH=${OS_ARCH} -t ${INJECTOR_IMAGE} -f bin/injector/Dockerfile ./bin/injector/
	docker save -o ./bin/injector/injector.tar ${INJECTOR_IMAGE}
	minikube image load --daemon=false --overwrite=true ./bin/injector/injector.tar

minikube-build-handler: handler
	docker build --build-arg TARGETARCH=${OS_ARCH} -t ${HANDLER_IMAGE} -f bin/handler/Dockerfile ./bin/handler/
	docker save -o ./bin/handler/handler.tar ${HANDLER_IMAGE}
	minikube image load --daemon=false --overwrite=true ./bin/handler/handler.tar

minikube-build-rbac-proxy:
	minikube image load --daemon=false --overwrite=true gcr.io/kubebuilder/kube-rbac-proxy:v0.4.1

minikube-build:
	$(MAKE) -j4 minikube-build-manager minikube-build-injector minikube-build-handler minikube-build-rbac-proxy

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

colima-stop:
	colima stop

colima-start:
	colima start --runtime containerd --kubernetes --kubernetes-version=${KUBERNETES_VERSION}+k3s1 --cpu 4 --memory 4 --disk 50
# We also install iproute2-tc that contains 'tc' and enables to perform debugging from the VM
	colima ssh -- sudo apk add iproute2-tc

colima-install-nerdctl:
	sudo colima nerdctl install

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

release:
	VERSION=$(VERSION) ./tasks/release.sh
