# Image URL to use all building/pushing image targets
MANAGER_IMAGE ?= chaos-fi-controller:latest
INJECTOR_IMAGE ?= chaos-fi:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager injector

# Run tests
test: generate manifests
	go test ./... -coverprofile cover.out -gcflags=-l

# Build manager binary
manager: generate
	go build -o bin/manager main.go

# Build injector binary
injector:
	GOOS=linux GOARCH=amd64 go build -o bin/injector ./cli/injector/

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate manifests
	go run ./main.go

# Install CRDs into a cluster
install: manifests
	kustomize build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests
	kustomize build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	cd config/manager && kustomize edit set image controller=${MANAGER_IMAGE}
	kustomize build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Run golangci-lint against code
lint:
	golangci-lint run

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths="./..."

# Build the docker image
docker-build-manager:
	mkdir -p out
	docker build . -t ${MANAGER_IMAGE} --target manager
	docker save -o out/manager.tar ${MANAGER_IMAGE}
	scp -i $$(minikube ssh-key) out/manager.tar docker@$$(minikube ip):/tmp
	minikube ssh -- sudo ctr cri load /tmp/manager.tar

docker-build-injector:
	mkdir -p out
	docker build . -t ${INJECTOR_IMAGE} --target injector
	docker save -o out/injector.tar ${INJECTOR_IMAGE}
	scp -i $$(minikube ssh-key) out/injector.tar docker@$$(minikube ip):/tmp
	minikube ssh -- sudo ctr cri load /tmp/injector.tar

docker-build: docker-build-manager docker-build-injector

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.4 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

minikube-start:
	minikube start \
		--vm-driver=virtualbox \
		--container-runtime=containerd \
		--memory=4096 \
		--cpus=4 \
		--disk-size=50GB \
		--extra-config=apiserver.runtime-config=settings.k8s.io/v1alpha1=true \
		--extra-config=apiserver.enable-admission-plugins=NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,DefaultTolerationSeconds,NodeRestriction,MutatingAdmissionWebhook,ValidatingAdmissionWebhook,ResourceQuota,PodPreset \
		--iso-url=file://$(shell pwd)/minikube/iso/minikube.iso
