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

# Run tests
test: generate manifests
	go test ./... -coverprofile cover.out

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
	helm template ./chart | kubectl apply -f -

# Uninstall CRDs and controller from a cluster
uninstall: manifests
	helm template ./chart | kubectl delete -f -

restart:
	kubectl -n chaos-engineering rollout restart deployment chaos-controller

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) rbac:roleName=chaos-controller-role crd:trivialVersions=true paths="./..." output:crd:dir=./chart/templates/crds/ output:rbac:dir=./chart/templates/

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
minikube-build-manager: minikube-ssh-host manager
	mkdir -p out
	docker build -t ${MANAGER_IMAGE} -f bin/manager/Dockerfile ./bin/manager/
	rm -f out/manager.tar
	docker save -o out/manager.tar ${MANAGER_IMAGE}
	scp -o IdentitiesOnly=yes -i $$(minikube ssh-key) -o StrictHostKeyChecking=no out/manager.tar docker@$$(minikube ip):/tmp
	minikube ssh -- sudo ctr -n=k8s.io images import /tmp/manager.tar

minikube-build-injector: minikube-ssh-host injector
	mkdir -p out
	docker build -t ${INJECTOR_IMAGE} -f bin/injector/Dockerfile ./bin/injector/
	rm -f out/injector.tar
	docker save -o out/injector.tar ${INJECTOR_IMAGE}
	scp -o IdentitiesOnly=yes -i $$(minikube ssh-key) -o StrictHostKeyChecking=no out/injector.tar docker@$$(minikube ip):/tmp
	minikube ssh -- sudo ctr -n=k8s.io images import /tmp/injector.tar

minikube-build-handler: minikube-ssh-host handler
	mkdir -p out
	docker build -t ${HANDLER_IMAGE} -f bin/handler/Dockerfile ./bin/handler/
	rm -f out/handler.tar
	docker save -o out/handler.tar ${HANDLER_IMAGE}
	scp -o IdentitiesOnly=yes -i $$(minikube ssh-key) -o StrictHostKeyChecking=no out/handler.tar docker@$$(minikube ip):/tmp
	minikube ssh -- sudo ctr -n=k8s.io images import /tmp/handler.tar

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
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.4 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

minikube-memory := 4096
minikube-start-big: minikube-memory := 8192
minikube-start-big: minikube-start

minikube-start:
	minikube start \
		--vm-driver=virtualbox \
		--container-runtime=containerd \
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

minikube-ssh-host:
	ssh-keygen -R $(shell minikube ip)

release:
	VERSION=$(VERSION) ./tasks/release.sh
