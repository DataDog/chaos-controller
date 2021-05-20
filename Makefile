.PHONY: manager injector release
.SILENT: release

# Image URL to use all building/pushing image targets
MANAGER_IMAGE ?= chaos-controller:latest
INJECTOR_IMAGE ?= chaos-injector:latest

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
	docker save -o out/manager.tar ${MANAGER_IMAGE}
	scp -i $$(minikube ssh-key) -o StrictHostKeyChecking=no out/manager.tar docker@$$(minikube ip):/tmp
	minikube ssh -- sudo ctr -n=k8s.io images import /tmp/manager.tar

minikube-build-injector: minikube-ssh-host injector
	mkdir -p out
	docker build -t ${INJECTOR_IMAGE} -f bin/injector/Dockerfile ./bin/injector/
	docker save -o out/injector.tar ${INJECTOR_IMAGE}
	scp -i $$(minikube ssh-key) -o StrictHostKeyChecking=no out/injector.tar docker@$$(minikube ip):/tmp
	minikube ssh -- sudo ctr -n=k8s.io images import /tmp/injector.tar

minikube-build: minikube-build-manager minikube-build-injector

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

# fixing kubernetes version at 1.19.X because of this issue: https://github.com/kubernetes/kubernetes/issues/97288
# once the following fix is released (https://github.com/kubernetes/kubernetes/pull/97980), planned for 1.21, we can use the latest
# Kubernetes version again
minikube-start:
	minikube start \
		--vm-driver=virtualbox \
		--container-runtime=containerd \
		--memory=${minikube-memory} \
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

release:
ifeq (,$(VERSION))
	echo "You must specify a tag to release: VERSION=1.0.0 make release"
	exit 1
endif
ifneq (,$(shell git tag -l $(VERSION)))
	echo "Tag $(VERSION) already exists"
	exit 1
endif
ifneq (main,$(shell git branch --show-current))
	echo "You must run this target on main branch"
	exit 1
endif
ifneq (,$(shell git status --short))
	echo "You can't have pending changes when running this target, please stash or push any changes"
	exit 1
endif
ifneq (,$(shell git fetch --dry-run))
	echo "Your local main branch is not up-to-date with the remote main branch, please pull"
	exit 1
endif
	echo "Generating install manifest..."
	helm template chart/ --set images.tag=$(VERSION) > ./chart/install.yaml
	git add ./chart/install.yaml
	git commit -m "Generate install manifest for version $(VERSION)"
	echo "Creating git tag..."
	git tag -a $(VERSION) -m "Release $(VERSION)"
	echo "All done! Please run the following command when you feel ready:"
	echo "\t --> git push origin main --follow-tags <--"
