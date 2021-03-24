.PHONY: manager injector

# Image URL to use all building/pushing image targets
MANAGER_IMAGE ?= chaos-controller:latest
INJECTOR_IMAGE ?= chaos-injector:latest
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
	go test ./... -coverprofile cover.out

# Build manager binary
manager: generate
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/manager/manager main.go

# Build injector binary
injector:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/injector/injector ./cli/injector/

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate manifests
	go run ./main.go

# Install CRDs and controller into a cluster
install: manifests
	kustomize build config/crd | kubectl apply -f -
	cd config/manager && kustomize edit set image controller=${MANAGER_IMAGE}
	kustomize build config/default | kubectl -n chaos-engineering apply -f -

# Uninstall CRDs and controller from a cluster
uninstall: manifests
	kustomize build config/crd | kubectl delete -f -
	kustomize build config/default | kubectl -n chaos-engineering delete -f -

restart:
	kubectl -n chaos-engineering rollout restart deployment chaos-controller-controller-manager

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
docker-build-manager: minikube-ssh-host manager
	mkdir -p out
	docker build -t ${MANAGER_IMAGE} -f bin/manager/Dockerfile ./bin/manager/
	docker save -o out/manager.tar ${MANAGER_IMAGE}
	scp -i $$(minikube ssh-key) -o StrictHostKeyChecking=no out/manager.tar docker@$$(minikube ip):/tmp
	minikube ssh -- sudo ctr -n=k8s.io images import /tmp/manager.tar

docker-build-injector: minikube-ssh-host injector
	mkdir -p out
	docker build -t ${INJECTOR_IMAGE} -f bin/injector/Dockerfile ./bin/injector/
	docker save -o out/injector.tar ${INJECTOR_IMAGE}
	scp -i $$(minikube ssh-key) -o StrictHostKeyChecking=no out/injector.tar docker@$$(minikube ip):/tmp
	minikube ssh -- sudo ctr -n=k8s.io images import /tmp/injector.tar

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

# fixing kubernetes version at 1.19.X because of this issue: https://github.com/kubernetes/kubernetes/issues/97288
# once the following fix is released (https://github.com/kubernetes/kubernetes/pull/97980), planned for 1.21, we can use the latest
# Kubernetes version again
minikube-start:
	minikube start \
		--vm-driver=virtualbox \
		--container-runtime=containerd \
		--memory=4096 \
		--cpus=4 \
		--kubernetes-version=1.19.9 \
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

minikube-ssh-host:
	ssh-keygen -R $(shell minikube ip)
