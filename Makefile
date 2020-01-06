
# Image URL to use all building/pushing image targets
MANAGER_IMAGE ?= chaos-fi-controller:latest
INJECTOR_IMAGE ?= chaos-fi:latest

all: test manager injector

# Run tests
test: generate fmt vet manifests
	go test ./pkg/... ./cmd/... -coverprofile cover.out -gcflags=-l

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager github.com/DataDog/chaos-fi-controller/cmd/manager

# Build injector binary
injector: generate fmt vet
	go build -o bin/injector github.com/DataDog/chaos-fi-controller/cmd/injector

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet
	go run ./cmd/manager/main.go

# Install CRDs into a cluster
install: manifests
	kubectl apply -f config/crds

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	kubectl apply -f config/crds
	kustomize build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests:
	go run vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go all

# Run go fmt against code
fmt:
	go fmt ./pkg/... ./cmd/...

# Run go vet against code
vet:
	go vet ./pkg/... ./cmd/...

# Generate code
generate:
ifndef GOPATH
	$(error GOPATH not defined, please define GOPATH. Run "go help gopath" to learn more about GOPATH)
endif
	go generate ./pkg/... ./cmd/...

# Build the docker image
docker-build:
	if [[ -z "${DOCKER_HOST}" ]]; then echo 'Please run eval $$(minikube docker-env) before running this script'; exit 1; fi
	docker build . -t ${MANAGER_IMAGE} --target manager
	docker build . -t ${INJECTOR_IMAGE} --target injector
	minikube ssh -- sudo docker save -o manager.tar ${MANAGER_IMAGE}
	minikube ssh -- sudo docker save -o injector.tar ${INJECTOR_IMAGE}
	minikube ssh -- sudo ctr cri load manager.tar
	minikube ssh -- sudo ctr cri load injector.tar
	@echo "updating kustomize image patch file for manager resource"
	sed -i'' -e 's@image: .*@image: '"docker.io/library/${MANAGER_IMAGE}"'@' ./config/default/manager_image_patch.yaml

minikube-start:
	minikube start \
		--vm-driver=virtualbox \
		--container-runtime=containerd \
		--memory=4096 \
		--cpus=4 \
		--extra-config=apiserver.runtime-config=settings.k8s.io/v1alpha1=true \
		--extra-config=apiserver.enable-admission-plugins=NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,DefaultTolerationSeconds,NodeRestriction,MutatingAdmissionWebhook,ValidatingAdmissionWebhook,ResourceQuota,PodPreset
	minikube ssh -- sudo systemctl start docker
