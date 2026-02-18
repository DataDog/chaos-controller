.PHONY: *
.SILENT: release

include scripts/versions.env

NOW_ISO8601 = $(shell date -u +"%Y-%m-%dT%H:%M:%S")

GOOS = $(shell go env GOOS)
GOARCH = $(shell go env GOARCH)

# GOBIN can be provided (gitlab), defined (custom user setup), or empty/guessed (default go setup)
GOBIN ?= $(shell go env GOBIN)
ifeq (,$(GOBIN))
GOBIN = $(shell go env GOPATH)/bin
endif

# Lima requires to have images built on a specific namespace to be shared to the Kubernetes cluster when using containerd runtime
# https://github.com/abiosoft/colima#interacting-with-image-registry
CONTAINER_REGISTRY ?= k8s.io
CONTAINER_TAG ?= latest
CONTAINER_VERSION ?= $(shell git rev-parse HEAD)$(shell git diff --quiet || echo '-dirty')
CONTAINER_BUILD_EXTRA_ARGS ?=

SIGN_IMAGE ?= false

SKIP_GENERATE ?= false

# Image URL to use all building/pushing image targets
MANAGER_IMAGE ?= ${CONTAINER_REGISTRY}/chaos-controller
INJECTOR_IMAGE ?= ${CONTAINER_REGISTRY}/chaos-injector
HANDLER_IMAGE ?= ${CONTAINER_REGISTRY}/chaos-handler

# default instance name will be connected user name
LIMA_INSTANCE ?= $(shell whoami | tr "." "-")

# cluster name is used by e2e-test to target the node with a disruption
# CI need to be able to override this value
E2E_TEST_CLUSTER_NAME ?= lima-$(LIMA_INSTANCE)
E2E_TEST_KUBECTL_CONTEXT ?= lima

KUBECTL ?= limactl shell $(LIMA_INSTANCE) sudo kubectl

HELM_VALUES ?= dev.yaml

# Additional args to provide to test runner (ginkgo)
# examples:
# `make test TEST_ARGS=--until-it-fails` to run tests randomly and repeatedly until a failure might occur (help to detect flaky tests or wrong tests setup)
# `make test TEST_ARGS=injector` will focus on package injector to run tests
TEST_ARGS ?=

# Docker builds now happen entirely within Docker using multi-stage builds
# All binaries (Go + EBPF) are built inside Docker - no local build steps required

# Set container names for each component
docker-build-injector docker-build-only-injector: CONTAINER_NAME=$(INJECTOR_IMAGE)
docker-build-handler docker-build-only-handler: CONTAINER_NAME=$(HANDLER_IMAGE)
docker-build-manager docker-build-only-manager: CONTAINER_NAME=$(MANAGER_IMAGE)

lima-push-injector lima-push-handler lima-push-manager: FAKE_FOR=COMPLETION

# Generate manifests before building manager
# Skip generate if SKIP_GENERATE=true (useful in CI when manifests are already committed)
ifneq ($(SKIP_GENERATE),true)
docker-build-manager docker-build-only-manager: generate
endif

# Define template for docker build targets
# $(1) is the target name: injector|handler|manager
define TARGET_template

docker-build-$(1): docker-build-only-$(1)
	docker save $$(CONTAINER_NAME):$(CONTAINER_TAG) -o ./bin/$(1)/$(1).tar.gz

docker-build-only-$(1):
	docker buildx build \
		--build-arg BUILDGOVERSION=$(BUILDGOVERSION) \
		--build-arg BUILDSTAMP=$(NOW_ISO8601) \
		-t $$(CONTAINER_NAME):$(CONTAINER_TAG) \
		--metadata-file ./bin/$(1)/docker-metadata.json \
		$(CONTAINER_BUILD_EXTRA_ARGS) \
		-f bin/$(1)/Dockerfile .

	if [ "$${SIGN_IMAGE}" = "true" ]; then \
		ddsign sign $$(CONTAINER_NAME):$(CONTAINER_VERSION) --docker-metadata-file ./bin/$(1)/docker-metadata.json; \
	fi

lima-push-$(1): docker-build-$(1)
	limactl copy ./bin/$(1)/$(1).tar.gz $(LIMA_INSTANCE):/tmp/
	limactl shell $(LIMA_INSTANCE) -- sudo k3s ctr i import /tmp/$(1).tar.gz

minikube-load-$(1):
	ls -la ./bin/$(1)/$(1).tar.gz
	minikube image load --daemon=false --overwrite=true ./bin/$(1)/$(1).tar.gz
endef

# Define targets we want to generate make targets for
TARGETS = injector handler manager

# Generate docker build rules for each target
$(foreach tgt,$(TARGETS),$(eval $(call TARGET_template,$(tgt))))

# Build actions
docker-build-all: $(addprefix docker-build-,$(TARGETS))

docker-build-only-all: $(addprefix docker-build-only-,$(TARGETS))

lima-push-all: $(addprefix lima-push-,$(TARGETS))
minikube-load-all: $(addprefix minikube-load-,$(TARGETS))

# Build chaosli
chaosli:
	GOOS=darwin GOARCH=$(GOARCH) CGO_ENABLED=0 go build -ldflags="-X github.com/DataDog/chaos-controller/cli/chaosli/cmd.Version=$(VERSION)" -o bin/chaosli/chaosli_darwin_$(GOARCH) ./cli/chaosli/

# https://onsi.github.io/ginkgo/#recommended-continuous-integration-configuration
GINKGO_PROCS ?= 4

# Tests & CI
## Run unit tests
test: generate
	$(if $(GOPATH),,$(error GOPATH is not set. Please set GOPATH before running make test))
	GO_TEST_REPORT_NAME=$@ GINKGO_PROCS=$(GINKGO_PROCS) \
		GINKGO_TEST_ARGS="-r --skip-package=controllers --randomize-suites --timeout=10m $(TEST_ARGS)" \
		./scripts/run-tests.sh

spellcheck:
	./scripts/spellcheck.sh check

spellcheck-report:
	./scripts/spellcheck.sh report

spellcheck-docker:
	./scripts/spellcheck.sh docker

spellcheck-format-spelling:
	./scripts/spellcheck.sh format-spelling

ci-install-minikube:
	MINIKUBE_CPUS="$(MINIKUBE_CPUS)" MINIKUBE_MEMORY="$(MINIKUBE_MEMORY)" ./scripts/install-minikube.sh

SKIP_DEPLOY ?=

## Run e2e tests (against a real cluster)
## to run them locally you first need to run `make install-dev-tools`
e2e-test: generate
ifneq (true,$(SKIP_DEPLOY)) # we can only wait for a controller if it exists, local.yaml does not deploy the controller
	$(MAKE) lima-install HELM_VALUES=ci.yaml
endif
	GO_TEST_REPORT_NAME=$@ GINKGO_PROCS=$(GINKGO_PROCS) \
		E2E_TEST_CLUSTER_NAME=$(E2E_TEST_CLUSTER_NAME) E2E_TEST_KUBECTL_CONTEXT=$(E2E_TEST_KUBECTL_CONTEXT) \
		GINKGO_TEST_ARGS="--flake-attempts=3 --timeout=25m controllers" \
		./scripts/run-tests.sh

# Test chaosli API portability
chaosli-test:
	docker buildx build -f ./cli/chaosli/chaosli.DOCKERFILE -t test-chaosli-image .

# Go actions
## Generate manifests e.g. CRD, RBAC etc.
manifests:
	./scripts/install-controller-gen.sh
	./scripts/install-yamlfmt.sh
	controller-gen rbac:roleName=chaos-controller crd:crdVersions=v1 paths="./..." output:crd:dir=./chart/templates/generated/ output:rbac:dir=./chart/templates/generated/
# ensure generated files stays formatted as expected
	yamlfmt chart/templates/generated

## Run go fmt against code
fmt:
	go fmt ./...

## Run go vet against code
vet:
	go vet ./...

## Run golangci-lint against code
lint:
	./scripts/install-golangci-lint.sh
# By using GOOS=linux we aim to validate files as if we were on linux
# you can use a similar trick with gopls to have vs-code linting your linux platform files instead of darwin
	GOOS=linux golangci-lint run -E ginkgolinter ./...
	GOOS=linux golangci-lint run

## Generate all code (CRDs, RBAC, DeepCopy, mocks, protobuf, headers)
generate:
	./scripts/install-controller-gen.sh
	./scripts/install-yamlfmt.sh
	./scripts/install-mockery.sh
	./scripts/install-protobuf.sh
	controller-gen object:headerFile=./hack/boilerplate.go.txt paths="./..."
	controller-gen rbac:roleName=chaos-controller crd:crdVersions=v1 paths="./..." output:crd:dir=./chart/templates/generated/ output:rbac:dir=./chart/templates/generated/
	yamlfmt chart/templates/generated
	$(MAKE) clean-mocks
	go generate ./...
	cd grpc/disruptionlistener && \
		protoc --proto_path=. --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative disruptionlistener.proto
	cd dogfood/chaosdogfood && \
		protoc --proto_path=. --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative chaosdogfood.proto
	$(MAKE) header-fix

# Lima actions
## Create a new lima cluster and deploy the chaos-controller into it
lima-all: lima-start lima-install-datadog-agent lima-install-cert-manager lima-push-all lima-install
	kubens chaos-engineering

## Rebuild the chaos-controller images, re-install the chart and restart the chaos-controller pods
lima-redeploy: lima-push-all lima-install lima-restart

lima-start:
	./scripts/lima.sh start

lima-stop:
	./scripts/lima.sh stop

lima-kubectx-clean:
	./scripts/lima.sh kubectx-clean

lima-kubectx:
	./scripts/lima.sh kubectx

lima-install: manifests
	KUBECTL="$(KUBECTL)" HELM_VALUES="$(HELM_VALUES)" CONTAINER_VERSION="$(CONTAINER_VERSION)" \
		./scripts/lima.sh install

lima-uninstall:
	KUBECTL="$(KUBECTL)" HELM_VALUES="$(HELM_VALUES)" ./scripts/lima.sh uninstall

lima-restart:
	KUBECTL="$(KUBECTL)" HELM_VALUES="$(HELM_VALUES)" ./scripts/lima.sh restart

lima-install-cert-manager:
	KUBECTL="$(KUBECTL)" ./scripts/lima.sh install-cert-manager

lima-install-demo:
	KUBECTL="$(KUBECTL)" ./scripts/lima.sh install-demo

lima-install-longhorn:
	KUBECTL="$(KUBECTL)" ./scripts/lima.sh install-longhorn

# CI-specific actions

venv:
	test -d .venv || python3 -m venv .venv
	. .venv/bin/activate; pip install -qr tasks/requirements.txt

header: venv
	. .venv/bin/activate; inv header-check

header-fix:
# First re-generate header, it should complain as just (re)generated mocks does not contains them
	-$(MAKE) header
# Then, re-generate header, it should succeed as now all files contains headers as expected, and command return with an happy exit code
	$(MAKE) header

license: venv
	. .venv/bin/activate; inv license-check

godeps:
	go mod tidy; go mod vendor

update-deps:
	./scripts/update-deps.sh

deps: godeps license

clean-mocks:
	find . -type f -name "*mock*.go" -not -path "./vendor/*" -exec rm {} \;
	rm -rf mocks/

release:
	VERSION=$(VERSION) ./tasks/release.sh

_pre_local: generate
	KUBECTL="$(KUBECTL)" HELM_VALUES="$(HELM_VALUES)" ./scripts/lima.sh pre-local

debug: _pre_local
	@echo "now you can launch through vs-code or your favorite IDE a controller in debug with appropriate configuration (--config=chart/values/local.yaml + CONTROLLER_NODE_NAME=local)"

run:
	CONTROLLER_NODE_NAME=local go run . --config=.local.yaml

watch: _pre_local install-watchexec
	watchexec make SKIP_EBPF=true lima-push-injector run

install-watchexec:
	./scripts/install-watchexec.sh

install-go:
	BUILDGOVERSION=$(BUILDGOVERSION) ./scripts/install-go

# Grouped tool installation targets
install-lint-tools:
	./scripts/install-golangci-lint.sh
	./scripts/install-controller-gen.sh

install-test-tools:
	./scripts/install-controller-gen.sh
	./scripts/install-yamlfmt.sh
	./scripts/install-kubebuilder.sh
	./scripts/install-datadog-ci.sh

install-generate-tools:
	./scripts/install-controller-gen.sh
	./scripts/install-yamlfmt.sh
	./scripts/install-mockery.sh
	./scripts/install-protobuf.sh

install-e2e-tools:
	./scripts/install-controller-gen.sh
	./scripts/install-yamlfmt.sh
	./scripts/install-helm.sh
	./scripts/install-kubebuilder.sh
	./scripts/install-datadog-ci.sh

install-dev-tools:
	./scripts/install-golangci-lint.sh
	./scripts/install-controller-gen.sh
	./scripts/install-mockery.sh
	./scripts/install-yamlfmt.sh
	./scripts/install-protobuf.sh
	./scripts/install-kubebuilder.sh
	./scripts/install-helm.sh
	./scripts/install-datadog-ci.sh

lima-install-datadog-agent:
	KUBECTL="$(KUBECTL)" ./scripts/lima.sh install-datadog-agent

open-dd:
	./scripts/lima.sh open-dd
