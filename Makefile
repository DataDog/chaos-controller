
# Image URL to use all building/pushing image targets
IMG ?= chaos-fi-controller:latest

all: test manager

# Run tests
test: generate fmt vet manifests
	go test ./pkg/... ./cmd/... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager github.com/DataDog/chaos-fi-controller/cmd/manager

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
	docker build . -t ${IMG}
	minikube ssh -- sudo docker save -o image.tar ${IMG}
	minikube ssh -- sudo ctr cri load image.tar
	@echo "updating kustomize image patch file for manager resource"
	sed -i'' -e 's@image: .*@image: '"docker.io/library/${IMG}"'@' ./config/default/manager_image_patch.yaml

# Push the docker image
docker-push:
	docker push ${IMG}

minikube-start:
	minikube start \
		--container-runtime=containerd \
		--memory=4096 \
		--cpus=4 \
		--extra-config=apiserver.runtime-config=settings.k8s.io/v1alpha1=true \
		--extra-config=apiserver.enable-admission-plugins=NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,DefaultTolerationSeconds,NodeRestriction,MutatingAdmissionWebhook,ValidatingAdmissionWebhook,ResourceQuota,PodPreset
