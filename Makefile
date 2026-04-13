.PHONY: *
.SILENT: release

NOW_ISO8601 = $(shell date -u +"%Y-%m-%dT%H:%M:%S")

GOOS = $(shell go env GOOS)
GOARCH = $(shell go env GOARCH)

# change also github actions go build version "GO_VERSION:" if you change the version below
# https://github.com/DataDog/chaos-controller/blob/main/.github/workflows/ci.yml#L13
BUILDGOVERSION = 1.26.2

# GOBIN can be provided (gitlab), defined (custom user setup), or empty/guessed (default go setup)
GOBIN ?= $(shell go env GOBIN)
ifeq (,$(GOBIN))
GOBIN = $(shell go env GOPATH)/bin
endif

# Local binary directory – tools are installed here to avoid polluting GOBIN/GOPATH
LOCALBIN ?= $(shell pwd)/bin/tools

# Per-tool path variables – override with e.g. HELM=/usr/bin/helm make lima-install
CONTROLLER_GEN = $(LOCALBIN)/controller-gen
YAMLFMT        = $(LOCALBIN)/yamlfmt
GOLANGCI_LINT  = $(LOCALBIN)/golangci-lint
HELM           = $(LOCALBIN)/helm
WATCHEXEC      = $(LOCALBIN)/watchexec
MOCKERY        = $(LOCALBIN)/mockery
DATADOG_CI     = $(LOCALBIN)/datadog-ci
PROTOC              = $(LOCALBIN)/protoc
PROTOC_INCLUDE      = $(shell pwd)/bin/protoc-include
PROTOC_GEN_GO       = $(LOCALBIN)/protoc-gen-go
PROTOC_GEN_GO_GRPC  = $(LOCALBIN)/protoc-gen-go-grpc

INSTALL_DATADOG_AGENT = false
LIMA_INSTALL_SINK = noop
ifdef STAGING_DATADOG_API_KEY
ifdef STAGING_DATADOG_APP_KEY
INSTALL_DATADOG_AGENT = true
LIMA_INSTALL_SINK = datadog
endif
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

LIMA_PROFILE ?= lima
LIMA_CONFIG ?= lima
# default instance name will be connected user name
LIMA_INSTANCE ?= $(shell whoami | tr "." "-")

# cluster name is used by e2e-test to target the node with a disruption
# CI need to be able to override this value
E2E_TEST_CLUSTER_NAME ?= lima-$(LIMA_INSTANCE)
E2E_TEST_KUBECTL_CONTEXT ?= lima

KUBECTL ?= limactl shell $(LIMA_INSTANCE) sudo kubectl
PROTOC_VERSION = 3.17.3
PROTOC_OS ?= osx
PROTOC_ZIP = protoc-${PROTOC_VERSION}-${PROTOC_OS}-x86_64.zip
# you might also want to change ~/lima.yaml k3s version
KUBERNETES_MAJOR_VERSION ?= 1.28
KUBERNETES_VERSION ?= v$(KUBERNETES_MAJOR_VERSION).0
USE_VOLUMES ?= false

HELM_VALUES ?= dev.yaml
HELM_VERSION = v3.19.0
HELM_INSTALLED_VERSION = $(shell ($(LOCALBIN)/helm version --template="{{ .Version }}" 2>/dev/null || echo "") | awk '{ print $$1 }')

# TODO: reenable depguard in .golangci.yml after upgrading golangci-lint again
GOLANGCI_LINT_VERSION = 2.11.3
GOLANGCI_LINT_INSTALLED_VERSION = $(shell ($(LOCALBIN)/golangci-lint --version 2>/dev/null || echo "") | sed -E 's/.*version ([^ ]+).*/\1/')

CONTROLLER_GEN_VERSION = v0.19.0
CONTROLLER_GEN_INSTALLED_VERSION = $(shell ($(LOCALBIN)/controller-gen --version 2>/dev/null || echo "") | awk '{ print $$2 }')

MOCKERY_VERSION = 2.53.5
MOCKERY_ARCH = $(GOARCH)
ifeq (amd64,$(GOARCH))
MOCKERY_ARCH = x86_64
endif
MOCKERY_INSTALLED_VERSION = $(shell $(LOCALBIN)/mockery --version --quiet --config="" 2>/dev/null || echo "")

YAMLFMT_VERSION = 0.9.0

WATCHEXEC_VERSION = 2.2.1
WATCHEXEC_OS = unknown-linux-musl
ifeq (darwin,$(GOOS))
WATCHEXEC_OS = apple-darwin
endif
WATCHEXEC_ARCH_WE = $(GOARCH)
ifeq (amd64,$(GOARCH))
WATCHEXEC_ARCH_WE = x86_64
endif
ifeq (arm64,$(GOARCH))
WATCHEXEC_ARCH_WE = aarch64
endif
WATCHEXEC_ARCHIVE = watchexec-$(WATCHEXEC_VERSION)-$(WATCHEXEC_ARCH_WE)-$(WATCHEXEC_OS)
WATCHEXEC_INSTALLED_VERSION = $(shell $(LOCALBIN)/watchexec --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)

PROTOC_INSTALLED_VERSION = $(shell $(LOCALBIN)/protoc --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)

PROTOC_GEN_GO_VERSION          = v1.27.1
PROTOC_GEN_GO_INSTALLED_VERSION = $(shell $(LOCALBIN)/protoc-gen-go --version 2>&1 | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' | head -1)

PROTOC_GEN_GO_GRPC_VERSION          = 1.1.0
PROTOC_GEN_GO_GRPC_INSTALLED_VERSION = $(shell $(LOCALBIN)/protoc-gen-go-grpc --version 2>&1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)

# SHA256 checksums for downloaded tool archives.
# Update these values whenever a tool version is bumped.
# Sources are the official checksums files published alongside each release.

# golangci-lint v$(GOLANGCI_LINT_VERSION)
# Source: https://github.com/golangci/golangci-lint/releases/download/v$(GOLANGCI_LINT_VERSION)/golangci-lint-$(GOLANGCI_LINT_VERSION)-checksums.txt
GOLANGCI_LINT_SHA256_darwin_amd64  = f93bda1f2cc981fd1326464020494be62f387bbf262706e1b3b644e5afacc440
GOLANGCI_LINT_SHA256_darwin_arm64  = 30ee39979c516b9d1adca289a3f93429d130c4c0fda5e57d637850894221f6cc
GOLANGCI_LINT_SHA256_linux_amd64   = 87bb8cddbcc825d5778b64e8a91b46c0526b247f4e2f2904dea74ec7450475d1
GOLANGCI_LINT_SHA256_linux_arm64   = ee3d95f301359e7d578e6d99c8ad5aeadbabc5a13009a30b2b0df11c8058afe9
GOLANGCI_LINT_SHA256               = $(GOLANGCI_LINT_SHA256_$(GOOS)_$(GOARCH))

# helm v$(HELM_VERSION)
# Source: https://get.helm.sh/helm-$(HELM_VERSION)-{os}-{arch}.tar.gz.sha256sum
HELM_SHA256_darwin_amd64           = 09a108c0abda42e45af172be65c49125354bf7cd178dbe10435e94540e49c7b9
HELM_SHA256_darwin_arm64           = 31513e1193da4eb4ae042eb5f98ef9aca7890cfa136f4707c8d4f70e2115bef6
HELM_SHA256_linux_amd64            = a7f81ce08007091b86d8bd696eb4d86b8d0f2e1b9f6c714be62f82f96a594496
HELM_SHA256_linux_arm64            = 440cf7add0aee27ebc93fada965523c1dc2e0ab340d4348da2215737fc0d76ad
HELM_SHA256                        = $(HELM_SHA256_$(GOOS)_$(GOARCH))

# mockery v$(MOCKERY_VERSION)
# Source: https://github.com/vektra/mockery/releases/download/v$(MOCKERY_VERSION)/checksum.txt
MOCKERY_SHA256_darwin_amd64        = e079356a96abea6001d889ab89e7408c66b33657c12f97d4becd751cec0ab10c
MOCKERY_SHA256_darwin_arm64        = b1792e65417a9670db4125c53b770e4c5974cf7aafb45db6a47bcb90ff7d24b0
MOCKERY_SHA256_linux_amd64         = 1f2028be64f20cb28983cf8a80fd4a7b036b8a9e389d443c295e531e77025641
MOCKERY_SHA256_linux_arm64         = c9bb64143068ef9f876ecaf4b2850ea57f909c16412cb6fc562857b3569bb125
MOCKERY_SHA256                     = $(MOCKERY_SHA256_$(GOOS)_$(GOARCH))

# yamlfmt v0.9.0
# Source: https://github.com/google/yamlfmt/releases/download/v0.9.0/checksums.txt
YAMLFMT_SHA256_darwin_amd64        = ad8d81279b63e6f6cb55ff9c1da654477494b727f882b6531ba3ed8715aa3634
YAMLFMT_SHA256_darwin_arm64        = dbfbcc105961444cd027e0e8dd321df920f3f606398b35e4070ca1d9aea45ea8
YAMLFMT_SHA256_linux_amd64         = dd5a0304167c6a42660f7ff3fd0d7c68bcf1dd9512e3f4e5645f7e4c5c21b1a8
YAMLFMT_SHA256_linux_arm64         = 2194995728713476c77454cea867660426b3a9d68158f2940d9bb1c29e68252b
YAMLFMT_SHA256                     = $(YAMLFMT_SHA256_$(GOOS)_$(GOARCH))

# watchexec v$(WATCHEXEC_VERSION)
# Source: https://github.com/watchexec/watchexec/releases/download/v$(WATCHEXEC_VERSION)/SHA256SUMS
WATCHEXEC_SHA256_darwin_amd64      = 2728f16bf287d7ed9545762c8c70925174e264dae4c229e7a85a2b5310b66b2b
WATCHEXEC_SHA256_darwin_arm64      = ac9db54f84d76763709b5526c699c46e99286976341be2cd999ce4e2c98d9998
WATCHEXEC_SHA256_linux_amd64       = 74651d6f450bca5436eee35b7828f1b97388d3b3976da313db36e3a91f7ada44
WATCHEXEC_SHA256_linux_arm64       = 87ec2094f2e883a090cb4a72a073f9b44f4aba7f50481f068e175f993d15c581
WATCHEXEC_SHA256                   = $(WATCHEXEC_SHA256_$(GOOS)_$(GOARCH))

# Portable SHA256 verification: macOS ships shasum, Linux ships sha256sum
SHA256SUM = sha256sum
ifeq (darwin,$(GOOS))
SHA256SUM = shasum -a 256
endif

# Additional args to provide to test runner (ginkgo)
# examples:
# `make test TEST_ARGS=--until-it-fails` to run tests randomly and repeatedly until a failure might occur (help to detect flaky tests or wrong tests setup)
# `make test TEST_ARGS=injector` will focus on package injector to run tests
TEST_ARGS ?=

DD_ENV = local
# https://circleci.com/docs/variables/
# we rely on standard CI env var to adapt test upload configuration automatically
ifeq (true,$(CI))
DD_ENV = ci
endif

LIMA_CGROUPS=v1
ifeq (v2,$(CGROUPS))
LIMA_CGROUPS=v2
endif

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
docker-build-manager docker-build-only-manager: generate-controller
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

_ginkgo_test:
# Run the test and write a file if succeed
# Do not stop on any error
	-go run github.com/onsi/ginkgo/v2/ginkgo --fail-on-pending --keep-going --vv \
		--cover --coverprofile=cover.profile --randomize-all \
		--race --trace --json-report=report-$(GO_TEST_REPORT_NAME).json --junit-report=report-$(GO_TEST_REPORT_NAME).xml \
		--compilers=$(GINKGO_PROCS) --procs=$(GINKGO_PROCS) \
		--poll-progress-after=10s --poll-progress-interval=10s \
		$(GINKGO_TEST_ARGS) \
			&& touch report-$(GO_TEST_REPORT_NAME)-succeed
# Try upload test reports if allowed and necessary prerequisites exists
ifneq (true,$(GO_TEST_SKIP_UPLOAD)) # you can bypass test upload
ifdef DATADOG_API_KEY # if no API key bypass is guaranteed
ifneq (,$(wildcard $(LOCALBIN)/datadog-ci)) # same if no test binary
	-DD_ENV=$(DD_ENV) $(DATADOG_CI) junit upload --service chaos-controller --tags="team:chaos-engineering,type:$(GO_TEST_REPORT_NAME)" report-$(GO_TEST_REPORT_NAME).xml
else
	@echo "datadog-ci binary is not installed, run 'make install-datadog-ci' to upload tests results to datadog"
endif
else
	@echo "DATADOG_API_KEY env var is not defined, create a local API key https://app.datadoghq.com/personal-settings/application-keys if you want to upload your local tests results to datadog"
endif
else
	@echo "datadog-ci junit upload SKIPPED"
endif
# Fail if succeed file does not exists
	[ -f report-$(GO_TEST_REPORT_NAME)-succeed ] && rm -f report-$(GO_TEST_REPORT_NAME)-succeed || exit 1

# Tests & CI
## Run unit tests
test: generate-controller manifests
	$(if $(GOPATH),,$(error GOPATH is not set. Please set GOPATH before running make test))
	$(MAKE) _ginkgo_test GO_TEST_REPORT_NAME=$@ \
		GINKGO_TEST_ARGS="-r --skip-package=controllers --randomize-suites --timeout=10m $(TEST_ARGS)"

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
MINIKUBE_CPUS ?= 6
MINIKUBE_MEMORY ?= 28672

ci-install-minikube:
	curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube_latest_amd64.deb
	sudo dpkg -i minikube_latest_amd64.deb
	minikube start --cpus='$(MINIKUBE_CPUS)' --memory='$(MINIKUBE_MEMORY)' --vm-driver=docker --container-runtime=containerd --kubernetes-version=${KUBERNETES_VERSION}
	minikube status

SKIP_DEPLOY ?=

## Run e2e tests (against a real cluster)
## to run them locally you first need to run `make install-kubebuilder`
e2e-test: generate-controller manifests
ifneq (true,$(SKIP_DEPLOY)) # we can only wait for a controller if it exists, local.yaml does not deploy the controller
	$(MAKE) lima-install HELM_VALUES=ci.yaml
endif
	E2E_TEST_CLUSTER_NAME=$(E2E_TEST_CLUSTER_NAME) E2E_TEST_KUBECTL_CONTEXT=$(E2E_TEST_KUBECTL_CONTEXT) $(MAKE) _ginkgo_test GO_TEST_REPORT_NAME=$@ \
		GINKGO_TEST_ARGS="--flake-attempts=3 --timeout=25m controllers"

# Test chaosli API portability
chaosli-test:
	docker buildx build -f ./cli/chaosli/chaosli.DOCKERFILE -t test-chaosli-image .

# Go actions
## Generate manifests e.g. CRD, RBAC etc.
manifests: install-controller-gen install-yamlfmt
	$(CONTROLLER_GEN) rbac:roleName=chaos-controller crd:crdVersions=v1 paths="./..." output:crd:dir=./chart/templates/generated/ output:rbac:dir=./chart/templates/generated/
# ensure generated files stays formatted as expected
	$(YAMLFMT) chart/templates/generated

## Run go fmt against code
fmt:
	go fmt ./...

## Run go vet against code
vet:
	go vet ./...

## Run golangci-lint against code
lint: install-golangci-lint
# By using GOOS=linux we aim to validate files as if we were on linux
# you can use a similar trick with gopls to have vs-code linting your linux platform files instead of darwin
	GOOS=linux $(GOLANGCI_LINT) run -E ginkgolinter ./...
	GOOS=linux $(GOLANGCI_LINT) run

## Generate all generated artifacts (code, manifests, mocks, protobufs)
generate: generate-controller manifests generate-mocks generate-disruptionlistener-protobuf generate-chaosdogfood-protobuf

## Generate controller code
generate-controller: install-controller-gen
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths="./..."

# Lima actions
## Create a new lima cluster and deploy the chaos-controller into it
lima-all: lima-start lima-install-datadog-agent lima-install-cert-manager lima-push-all lima-install
	kubens chaos-engineering

## Rebuild the chaos-controller images, re-install the chart and restart the chaos-controller pods
lima-redeploy: lima-push-all lima-install lima-restart

## Install cert-manager chart
lima-install-cert-manager:
	$(KUBECTL) apply -f https://github.com/jetstack/cert-manager/releases/download/v1.9.1/cert-manager.yaml
	$(KUBECTL) -n cert-manager rollout status deployment/cert-manager-webhook --timeout=180s

lima-install-demo:
	$(KUBECTL) apply -f - < ./examples/namespace.yaml
	$(KUBECTL) apply -f - < ./examples/demo.yaml
	$(KUBECTL) -n chaos-demo rollout status deployment/demo-curl --timeout=60s
	$(KUBECTL) -n chaos-demo rollout status deployment/demo-nginx --timeout=60s

## Install CRDs and controller into a lima k3s cluster
## In order to use already built images inside the containerd runtime
## we override images for all of our components to the expected namespace
lima-install: manifests install-helm
	$(HELM) template \
		--set=controller.version=$(CONTAINER_VERSION) \
		--set=controller.metricsSink=$(LIMA_INSTALL_SINK) \
		--set=controller.profilerSink=$(LIMA_INSTALL_SINK) \
		--set=controller.tracerSink=$(LIMA_INSTALL_SINK) \
		--values ./chart/values/$(HELM_VALUES) \
		./chart | $(KUBECTL) apply -f -
ifneq (local.yaml,$(HELM_VALUES)) # we can only wait for a controller if it exists, local.yaml does not deploy the controller
	$(KUBECTL) -n chaos-engineering rollout status deployment/chaos-controller --timeout=60s
endif

## Uninstall CRDs and controller from a lima k3s cluster
lima-uninstall: install-helm
	$(HELM) template --set=skipNamespace=true --values ./chart/values/$(HELM_VALUES) ./chart | $(KUBECTL) delete -f -

## Restart the chaos-controller pod
lima-restart:
ifneq (local.yaml,$(HELM_VALUES)) # we can only wait for a controller if it exists, local.yaml does not deploy the controller
	$(KUBECTL) -n chaos-engineering rollout restart deployment/chaos-controller
	$(KUBECTL) -n chaos-engineering rollout status deployment/chaos-controller --timeout=60s
endif

## Remove lima references from kubectl config
lima-kubectx-clean:
	-kubectl config delete-cluster ${LIMA_PROFILE}
	-kubectl config delete-context ${LIMA_PROFILE}
	-kubectl config delete-user ${LIMA_PROFILE}
	kubectl config unset current-context

lima-kubectx:
	limactl shell $(LIMA_INSTANCE) sudo sed 's/default/lima/g' /etc/rancher/k3s/k3s.yaml > ~/.kube/config_lima
	KUBECONFIG=${KUBECONFIG}:~/.kube/config:~/.kube/config_lima kubectl config view --flatten > /tmp/config
	rm ~/.kube/config_lima
	mv /tmp/config ~/.kube/config
	chmod 600 ~/.kube/config
	kubectx ${LIMA_PROFILE}

## Stop and delete the lima cluster
lima-stop:
	limactl stop -f $(LIMA_INSTANCE)
	limactl delete $(LIMA_INSTANCE)
	$(MAKE) lima-kubectx-clean

## Start the lima cluster, pre-cleaning kubectl config
lima-start: lima-kubectx-clean
	LIMA_CGROUPS=${LIMA_CGROUPS} LIMA_CONFIG=${LIMA_CONFIG} LIMA_INSTANCE=${LIMA_INSTANCE} ./scripts/lima_start.sh
	$(MAKE) lima-kubectx

# Longhorn is used as an alternative StorageClass in order to enable "reliable" disk throttling accross various local setup
# It aims to bypass some issues encountered with default StorageClass (local-path --> tmpfs) that led to virtual unnamed devices
# unnamed devices are linked to 0 as a major device identifier, that blkio does not support
# https://super-unix.com/unixlinux/can-you-throttle-the-bandwidth-to-a-tmpfs-based-ramdisk/
lima-install-longhorn:
	$(KUBECTL) apply -f https://raw.githubusercontent.com/longhorn/longhorn/v1.4.0/deploy/longhorn.yaml

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
	@echo "Updating Python dependencies..."
	@pip install -q uv
	@uv pip compile --python-platform linux tasks/requirements.in -o tasks/requirements.txt
	@echo "Updated tasks/requirements.txt"
	@echo "Please commit both tasks/requirements.in and tasks/requirements.txt"

deps: godeps license

generate-disruptionlistener-protobuf: install-protobuf install-protoc-gen-go install-protoc-gen-go-grpc
	cd grpc/disruptionlistener && \
	PATH=$(LOCALBIN):$$PATH protoc --proto_path=. --proto_path=$(PROTOC_INCLUDE) --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative disruptionlistener.proto

generate-chaosdogfood-protobuf: install-protobuf install-protoc-gen-go install-protoc-gen-go-grpc
	cd dogfood/chaosdogfood && \
	PATH=$(LOCALBIN):$$PATH protoc --proto_path=. --proto_path=$(PROTOC_INCLUDE) --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative chaosdogfood.proto

clean-mocks:
	find . -type f -name "*mock*.go" -not -path "./vendor/*" -exec rm {} \;
	rm -rf mocks/

generate-mocks: clean-mocks install-mockery
	PATH=$(LOCALBIN):$$PATH go generate ./...
	$(MAKE) header-fix

release:
	VERSION=$(VERSION) ./tasks/release.sh

_pre_local: generate-controller manifests
	@$(shell $(KUBECTL) get deploy chaos-controller 2> /dev/null)
ifeq (0,$(.SHELLSTATUS))
# uninstall using a non local value to ensure deployment is deleted
	-$(MAKE) lima-uninstall HELM_VALUES=dev.yaml
	$(MAKE) lima-install HELM_VALUES=local.yaml
	$(KUBECTL) -n chaos-engineering get cm chaos-controller -oyaml | yq '.data["config.yaml"]' > .local.yaml
	yq -i '.controller.webhook.certDir = "chart/certs"' .local.yaml
else
	@echo "Chaos controller is not installed, skipped!"
endif

debug: _pre_local
	@echo "now you can launch through vs-code or your favorite IDE a controller in debug with appropriate configuration (--config=chart/values/local.yaml + CONTROLLER_NODE_NAME=local)"

run:
	CONTROLLER_NODE_NAME=local go run . --config=.local.yaml

watch: _pre_local install-watchexec
	$(WATCHEXEC) make SKIP_EBPF=true lima-push-injector run

$(LOCALBIN):
	mkdir -p $(LOCALBIN)

install-protobuf: | $(LOCALBIN)
ifneq ($(PROTOC_INSTALLED_VERSION),$(PROTOC_VERSION))
	$(info protoc version $(PROTOC_VERSION) is not installed or version differs ($(PROTOC_VERSION) != $(PROTOC_INSTALLED_VERSION)))
	$(info installing protoc v$(PROTOC_VERSION)...)
	curl -sSLo /tmp/${PROTOC_ZIP} https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/${PROTOC_ZIP}
	unzip -o -j /tmp/${PROTOC_ZIP} bin/protoc -d $(LOCALBIN)
	mkdir -p $(shell pwd)/bin/protoc-include
	unzip -o /tmp/${PROTOC_ZIP} 'include/*' -d /tmp/protoc-extract
	cp -r /tmp/protoc-extract/include/. $(shell pwd)/bin/protoc-include/
	rm -rf /tmp/protoc-extract
	rm -f /tmp/${PROTOC_ZIP}
endif

install-golangci-lint: | $(LOCALBIN)
ifneq ($(GOLANGCI_LINT_VERSION),$(GOLANGCI_LINT_INSTALLED_VERSION))
	$(info golangci-lint version $(GOLANGCI_LINT_VERSION) is not installed or version differ ($(GOLANGCI_LINT_VERSION) != $(GOLANGCI_LINT_INSTALLED_VERSION)))
	$(info installing golangci-lint v$(GOLANGCI_LINT_VERSION)...)
	curl -sSfLo /tmp/golangci-lint.tar.gz "https://github.com/golangci/golangci-lint/releases/download/v$(GOLANGCI_LINT_VERSION)/golangci-lint-$(GOLANGCI_LINT_VERSION)-$(GOOS)-$(GOARCH).tar.gz"
	echo "$(GOLANGCI_LINT_SHA256)  /tmp/golangci-lint.tar.gz" | $(SHA256SUM) -c -
	tar -xzf /tmp/golangci-lint.tar.gz --directory=$(LOCALBIN) --strip-components=1 golangci-lint-$(GOLANGCI_LINT_VERSION)-$(GOOS)-$(GOARCH)/golangci-lint
	rm /tmp/golangci-lint.tar.gz
endif

install-kubebuilder: | $(LOCALBIN)
ifeq (,$(wildcard $(LOCALBIN)/kubebuilder))
# download kubebuilder and install locally.
	curl -sSLo $(LOCALBIN)/kubebuilder https://go.kubebuilder.io/dl/latest/$(GOOS)/$(GOARCH)
	chmod u+x $(LOCALBIN)/kubebuilder
endif
ifeq (,$(wildcard $(LOCALBIN)/setup-envtest))
# download setup-envtest and install related binaries locally
	GOBIN=$(LOCALBIN) go install -v sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
# setup-envtest use -p path $(KUBERNETES_MAJOR_VERSION).x
endif

install-helm: | $(LOCALBIN)
ifneq ($(HELM_INSTALLED_VERSION),$(HELM_VERSION))
	$(info helm version $(HELM_VERSION) is not installed or version differ ($(HELM_VERSION) != $(HELM_INSTALLED_VERSION)))
	$(info installing helm $(HELM_VERSION)...)
	curl -sSLo /tmp/helm.tar.gz "https://get.helm.sh/helm-$(HELM_VERSION)-$(GOOS)-$(GOARCH).tar.gz"
	echo "$(HELM_SHA256)  /tmp/helm.tar.gz" | $(SHA256SUM) -c -
	tar -xvzf /tmp/helm.tar.gz --directory=$(LOCALBIN) --strip-components=1 $(GOOS)-$(GOARCH)/helm
	rm /tmp/helm.tar.gz
endif

# install controller-gen expected version
install-controller-gen: | $(LOCALBIN)
ifneq ($(CONTROLLER_GEN_INSTALLED_VERSION),$(CONTROLLER_GEN_VERSION))
	$(info controller-gen version $(CONTROLLER_GEN_VERSION) is not installed or version differ ($(CONTROLLER_GEN_VERSION) != $(CONTROLLER_GEN_INSTALLED_VERSION)))
	$(info installing controller-gen $(CONTROLLER_GEN_VERSION)...)
	CGO_ENABLED=0 GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_GEN_VERSION)
endif

install-datadog-ci: | $(LOCALBIN)
ifeq (,$(wildcard $(LOCALBIN)/datadog-ci))
	curl -L --fail "https://github.com/DataDog/datadog-ci/releases/latest/download/datadog-ci_$(GOOS)-x64" --output "$(LOCALBIN)/datadog-ci" && chmod u+x $(LOCALBIN)/datadog-ci
endif

install-mockery: | $(LOCALBIN)
# recommended way to install mockery is through their released binaries, NOT go install...
# https://vektra.github.io/mockery/installation/#github-release
ifneq ($(MOCKERY_INSTALLED_VERSION),v$(MOCKERY_VERSION))
	$(info mockery version $(MOCKERY_VERSION) is not installed or version differ (v$(MOCKERY_VERSION) != $(MOCKERY_INSTALLED_VERSION)))
	$(info installing mockery v$(MOCKERY_VERSION)...)
	curl -sSLo /tmp/mockery.tar.gz https://github.com/vektra/mockery/releases/download/v$(MOCKERY_VERSION)/mockery_$(MOCKERY_VERSION)_$(GOOS)_$(MOCKERY_ARCH).tar.gz
	echo "$(MOCKERY_SHA256)  /tmp/mockery.tar.gz" | $(SHA256SUM) -c -
	tar -xvzf /tmp/mockery.tar.gz --directory=$(LOCALBIN) mockery
	rm /tmp/mockery.tar.gz
endif

YAMLFMT_ARCH = $(GOARCH)
ifeq (amd64,$(GOARCH))
YAMLFMT_ARCH = x86_64
endif

install-yamlfmt: | $(LOCALBIN)
ifeq (,$(wildcard $(LOCALBIN)/yamlfmt))
	$(info installing yamlfmt v$(YAMLFMT_VERSION)...)
	curl -sSLo /tmp/yamlfmt.tar.gz https://github.com/google/yamlfmt/releases/download/v$(YAMLFMT_VERSION)/yamlfmt_$(YAMLFMT_VERSION)_$(GOOS)_$(YAMLFMT_ARCH).tar.gz
	echo "$(YAMLFMT_SHA256)  /tmp/yamlfmt.tar.gz" | $(SHA256SUM) -c -
	tar -xvzf /tmp/yamlfmt.tar.gz --directory=$(LOCALBIN) yamlfmt
	rm /tmp/yamlfmt.tar.gz
endif

install-watchexec: | $(LOCALBIN)
ifneq ($(WATCHEXEC_INSTALLED_VERSION),$(WATCHEXEC_VERSION))
	$(info watchexec version $(WATCHEXEC_VERSION) is not installed or version differs ($(WATCHEXEC_VERSION) != $(WATCHEXEC_INSTALLED_VERSION)))
	$(info installing watchexec v$(WATCHEXEC_VERSION)...)
	curl -sSLo /tmp/watchexec.tar.xz "https://github.com/watchexec/watchexec/releases/download/v$(WATCHEXEC_VERSION)/$(WATCHEXEC_ARCHIVE).tar.xz"
	echo "$(WATCHEXEC_SHA256)  /tmp/watchexec.tar.xz" | $(SHA256SUM) -c -
	tar -xJf /tmp/watchexec.tar.xz --directory=$(LOCALBIN) --strip-components=1 $(WATCHEXEC_ARCHIVE)/watchexec
	rm /tmp/watchexec.tar.xz
endif

install-protoc-gen-go: | $(LOCALBIN)
ifneq ($(PROTOC_GEN_GO_INSTALLED_VERSION),$(PROTOC_GEN_GO_VERSION))
	$(info protoc-gen-go $(PROTOC_GEN_GO_VERSION) is not installed or version differs ($(PROTOC_GEN_GO_VERSION) != $(PROTOC_GEN_GO_INSTALLED_VERSION)))
	$(info installing protoc-gen-go $(PROTOC_GEN_GO_VERSION)...)
	GOBIN=$(LOCALBIN) go install google.golang.org/protobuf/cmd/protoc-gen-go@$(PROTOC_GEN_GO_VERSION)
endif

install-protoc-gen-go-grpc: | $(LOCALBIN)
ifneq ($(PROTOC_GEN_GO_GRPC_INSTALLED_VERSION),$(PROTOC_GEN_GO_GRPC_VERSION))
	$(info protoc-gen-go-grpc $(PROTOC_GEN_GO_GRPC_VERSION) is not installed or version differs ($(PROTOC_GEN_GO_GRPC_VERSION) != $(PROTOC_GEN_GO_GRPC_INSTALLED_VERSION)))
	$(info installing protoc-gen-go-grpc v$(PROTOC_GEN_GO_GRPC_VERSION)...)
	GOBIN=$(LOCALBIN) go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v$(PROTOC_GEN_GO_GRPC_VERSION)
endif

install-go:
	BUILDGOVERSION=$(BUILDGOVERSION) ./scripts/install-go

## Install all project tools into $(LOCALBIN)
install-tools: install-golangci-lint install-controller-gen install-mockery install-yamlfmt install-helm install-kubebuilder install-datadog-ci install-watchexec install-protobuf install-protoc-gen-go install-protoc-gen-go-grpc

EXISTING_HELM_RELEASE = $(shell $(HELM) status -n datadog-agent my-datadog-operator --output json 2>/dev/null | grep -q '"status":"deployed"' && echo "true" || echo "")

lima-install-datadog-agent: install-helm
ifeq (true,$(INSTALL_DATADOG_AGENT))
ifeq (,$(EXISTING_HELM_RELEASE))
	$(KUBECTL) create ns datadog-agent 2>/dev/null || true
	$(HELM) repo add --force-update datadoghq https://helm.datadoghq.com
	$(HELM) install -n datadog-agent my-datadog-operator datadoghq/datadog-operator
	$(KUBECTL) create secret -n datadog-agent generic datadog-secret --from-literal api-key=${STAGING_DATADOG_API_KEY} --from-literal app-key=${STAGING_DATADOG_APP_KEY} 2>/dev/null || true
	$(KUBECTL) wait --for=condition=Available -n datadog-agent deployment --all --timeout=120s
endif
	LIMA_INSTANCE=$(LIMA_INSTANCE) envsubst < examples/datadog-agent.yaml | $(KUBECTL) apply -f -
endif

open-dd:
ifeq (true,$(INSTALL_DATADOG_AGENT))
ifdef STAGING_DD_SITE
	open "https://${STAGING_DD_SITE}/infrastructure/hosts?hostType=all-hosts&scope=host%3Alima-$(LIMA_INSTANCE)"
else
	@echo "You need to define STAGING_DD_SITE in your .zshrc or similar to use this feature"
endif
endif
