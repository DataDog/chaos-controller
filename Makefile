.PHONY: *
.SILENT: release

GOOS = $(shell go env GOOS)
GOARCH = $(shell go env GOARCH)

# GOBIN can be provided (gitlab), defined (custom user setup), or empty/guessed (default go setup)
GOBIN ?= $(shell go env GOBIN)
ifeq (,$(GOBIN))
GOBIN = $(shell go env GOPATH)/bin
endif

INSTALL_DATADOG_AGENT = false
LIMA_INSTALL_SINK = noop
ifdef STAGING_DATADOG_API_KEY
ifdef STAGING_DATADOG_APP_KEY
INSTALL_DATADOG_AGENT = true
LIMA_INSTALL_SINK = datadog
endif
endif

ifndef CONTROLLER_APP_VERSION
CONTROLLER_APP_VERSION = $(shell git rev-parse HEAD)$(shell git diff --quiet || echo '-dirty')
endif

# Lima requires to have images built on a specific namespace to be shared to the Kubernetes cluster when using containerd runtime
# https://github.com/abiosoft/colima#interacting-with-image-registry
CONTAINERD_REGISTRY_PREFIX ?= k8s.io

# Image URL to use all building/pushing image targets
MANAGER_IMAGE ?= ${CONTAINERD_REGISTRY_PREFIX}/chaos-controller:latest
INJECTOR_IMAGE ?= ${CONTAINERD_REGISTRY_PREFIX}/chaos-injector:latest
HANDLER_IMAGE ?= ${CONTAINERD_REGISTRY_PREFIX}/chaos-handler:latest

LIMA_PROFILE ?= lima
LIMA_CONFIG ?= lima
# default instance name will be connected user name
LIMA_INSTANCE ?= $(shell whoami | tr "." "-")

KUBECTL ?= limactl shell $(LIMA_INSTANCE) sudo kubectl
PROTOC_VERSION = 3.17.3
PROTOC_OS ?= osx
PROTOC_ZIP = protoc-${PROTOC_VERSION}-${PROTOC_OS}-x86_64.zip
# you might also want to change ~/lima.yaml k3s version
KUBERNETES_MAJOR_VERSION ?= 1.26
KUBERNETES_VERSION ?= v$(KUBERNETES_MAJOR_VERSION).0
KUBEBUILDER_VERSION ?= 3.1.0
USE_VOLUMES ?= false

HELM_VALUES ?= dev.yaml
HELM_VERSION = v3.11.3
HELM_INSTALLED_VERSION = $(shell (helm version --template="{{ .Version }}" || echo "") | awk '{ print $$1 }')

GOLANGCI_LINT_VERSION = 1.52.2
GOLANGCI_LINT_INSTALLED_VERSION = $(shell (golangci-lint --version || echo "") | sed -E 's/.*version ([^ ]+).*/\1/')

CONTROLLER_GEN_VERSION = v0.11.4
CONTROLLER_GEN_INSTALLED_VERSION = $(shell (controller-gen --version || echo "") | awk '{ print $$2 }')

MOCKERY_VERSION = 2.28.2
MOCKERY_INSTALLED_VERSION = $(shell mockery --version --quiet --config="" 2>/dev/null || echo "")

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

# we define target specific variables values https://www.gnu.org/software/make/manual/html_node/Target_002dspecific.html
injector handler: BINARY_PATH=./cli/$(BINARY_NAME)
manager: BINARY_PATH=.

docker-build-injector: IMAGE_TAG=$(INJECTOR_IMAGE)
docker-build-handler: IMAGE_TAG=$(HANDLER_IMAGE)
docker-build-manager: IMAGE_TAG=$(MANAGER_IMAGE)

docker-build-ebpf:
	docker buildx build --platform linux/$(GOARCH) --build-arg ARCH=$(GOARCH) -t ebpf-builder-$(GOARCH) -f bin/ebpf-builder/Dockerfile ./bin/ebpf-builder/
	-rm -r bin/injector/ebpf/
ifeq (true,$(USE_VOLUMES))
# create a dummy container with volume to store files
# circleci remote docker does not allow to use volumes, locally we fallbakc to standard volume but can call this target with USE_VOLUMES=true to debug if necessary
# https://circleci.com/docs/building-docker-images/#mounting-folders
	-docker rm ebpf-volume
	-docker create --platform linux/$(GOARCH) -v /app --name ebpf-volume ebpf-builder-$(GOARCH) /bin/true
	-docker cp . ebpf-volume:/app
	-docker rm ebpf-builder
	docker run --platform linux/$(GOARCH) --volumes-from ebpf-volume --name=ebpf-builder ebpf-builder-$(GOARCH)
	docker cp ebpf-builder:/app/bin/injector/ebpf bin/injector/ebpf
	docker rm ebpf-builder
else
	docker run --rm --platform linux/$(GOARCH) -v $(shell pwd):/app ebpf-builder-$(GOARCH)
endif

lima-push-injector lima-push-handler lima-push-manager: FAKE_FOR=COMPLETION

_injector:;
_handler:;
_manager: generate

_docker-build-injector:
ifneq (true,$(SKIP_EBPF))
	$(MAKE) docker-build-ebpf
endif
_docker-build-handler:;
_docker-build-manager:;

# we define the template we expect for each target
# $(1) is the target name: injector|handler|manager
define TARGET_template
$(1): BINARY_NAME=$(1)

_$(1)_arm:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o ./bin/$(1)/$(1)_arm64 $$(BINARY_PATH)

_$(1)_amd:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./bin/$(1)/$(1)_amd64 $$(BINARY_PATH)

$(1): _$(1) _$(1)_arm _$(1)_amd

docker-build-$(1): _docker-build-$(1) $(1)
	docker buildx build --build-arg TARGETARCH=$(GOARCH) -t $$(IMAGE_TAG) -f bin/$(1)/Dockerfile ./bin/$(1)/
	docker save $$(IMAGE_TAG) -o ./bin/$(1)/$(1).tar.gz

lima-push-$(1): docker-build-$(1)
	limactl copy ./bin/$(1)/$(1).tar.gz $(LIMA_INSTANCE):/tmp/
	limactl shell $(LIMA_INSTANCE) -- sudo k3s ctr i import /tmp/$(1).tar.gz

minikube-load-$(1):
# let's fail if the file does not exists so we know, mk load is not failing
	ls -la ./bin/$(1)/$(1).tar.gz
	minikube image load --daemon=false --overwrite=true ./bin/$(1)/$(1).tar.gz
endef

# we define the targers we want to generate make target for
TARGETS = injector handler manager

# we generate the exact same rules as for target specific variables, hence completion works and no duplication 😎
$(foreach tgt,$(TARGETS),$(eval $(call TARGET_template,$(tgt))))

# Build actions
all: $(TARGETS)

docker-build-all: $(addprefix docker-build-,$(TARGETS))
lima-push-all: $(addprefix lima-push-,$(TARGETS))
minikube-load-all: $(addprefix minikube-load-,$(TARGETS))

# Build chaosli
chaosli:
	GOOS=darwin GOARCH=${OS_ARCH} CGO_ENABLED=0 go build -ldflags="-X github.com/DataDog/chaos-controller/cli/chaosli/cmd.Version=$(VERSION)" -o bin/chaosli/chaosli_darwin_${OS_ARCH} ./cli/chaosli/

# https://onsi.github.io/ginkgo/#recommended-continuous-integration-configuration
_ginkgo_test:
# Run the test and write a file if succeed
# Do not stop on any error
	-go run github.com/onsi/ginkgo/v2/ginkgo --fail-on-pending --keep-going \
		--cover --coverprofile=cover.profile --randomize-all \
		--race --trace --json-report=report-$(GO_TEST_REPORT_NAME).json --junit-report=report-$(GO_TEST_REPORT_NAME).xml \
		--compilers=4 --procs=4 \
		--poll-progress-after=15s --poll-progress-interval=15s \
		$(GINKGO_TEST_ARGS) \
			&& touch report-$(GO_TEST_REPORT_NAME)-succeed
# Try upload test reports if allowed and necessary prerequisites exists
ifneq (true,$(GO_TEST_SKIP_UPLOAD)) # you can bypass test upload
ifdef DATADOG_API_KEY # if no API key bypass is guaranteed
ifneq (,$(shell which datadog-ci)) # same if no test binary
	-DD_ENV=$(DD_ENV) datadog-ci junit upload --service chaos-controller --tags="team:chaos-engineering,type:$(GO_TEST_REPORT_NAME)" report-$(GO_TEST_REPORT_NAME).xml
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
test: generate manifests
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
ci-install-minikube:
	curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube_latest_amd64.deb
	sudo dpkg -i minikube_latest_amd64.deb
	minikube start --vm-driver=docker --container-runtime=containerd --kubernetes-version=${KUBERNETES_VERSION}
	minikube status

SKIP_DEPLOY ?=

## Run e2e tests (against a real cluster)
## to run them locally you first need to run `make install-kubebuilder`
e2e-test: generate manifests
ifneq (true,$(SKIP_DEPLOY)) # we can only wait for a controller if it exists, local.yaml does not deploy the controller
	$(MAKE) lima-install HELM_VALUES=ci.yaml
endif
	USE_EXISTING_CLUSTER=true $(MAKE) _ginkgo_test GO_TEST_REPORT_NAME=$@ \
		GINKGO_TEST_ARGS="--flake-attempts=3 --timeout=15m controllers"

# Test chaosli API portability
chaosli-test:
	docker buildx build -f ./cli/chaosli/chaosli.DOCKERFILE -t test-chaosli-image .

# Go actions
## Generate manifests e.g. CRD, RBAC etc.
manifests: install-controller-gen install-yamlfmt
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
lint: install-golangci-lint
# By using GOOS=linux we aim to validate files as if we were on linux
# you can use a similar trick with gopls to have vs-code linting your linux platform files instead of darwin
	GOOS=linux golangci-lint run --no-config -E ginkgolinter ./...
	GOOS=linux golangci-lint run

## Generate code
generate: install-controller-gen
	controller-gen object:headerFile=./hack/boilerplate.go.txt paths="./..."

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
lima-install: manifests
	helm template \
		--set=controller.version=$(CONTROLLER_APP_VERSION) \
		--set=controller.metricsSink=$(LIMA_INSTALL_SINK) \
		--set=controller.profilerSink=$(LIMA_INSTALL_SINK) \
		--set=controller.tracerSink=$(LIMA_INSTALL_SINK) \
		--values ./chart/values/$(HELM_VALUES) \
		./chart | $(KUBECTL) apply -f -
ifneq (local.yaml,$(HELM_VALUES)) # we can only wait for a controller if it exists, local.yaml does not deploy the controller
	$(KUBECTL) -n chaos-engineering rollout status deployment/chaos-controller --timeout=60s
endif

## Uninstall CRDs and controller from a lima k3s cluster
lima-uninstall:
	helm template --set=skipNamespace=true --values ./chart/values/$(HELM_VALUES) ./chart | $(KUBECTL) delete -f -

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
	source .venv/bin/activate; pip install -qr tasks/requirements.txt

header: venv
	source .venv/bin/activate; inv header-check

header-fix:
# First re-generate header, it should complain as just (re)generated mocks does not contains them
	-$(MAKE) header
# Then, re-generate header, it should succeed as now all files contains headers as expected, and command return with an happy exit code
	$(MAKE) header

license: venv
	source .venv/bin/activate; inv license-check

godeps:
	go mod tidy; go mod vendor

deps: godeps license

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

clean-mocks:
	find . -type file -name "*mock*.go" -not -path "./vendor/*" -exec rm {} \;
	rm -rf mocks/

generate-mocks: clean-mocks install-mockery
	go generate ./...
	$(MAKE) header-fix

release:
	VERSION=$(VERSION) ./tasks/release.sh

_pre_local: generate manifests
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
	watchexec make SKIP_EBPF=true lima-push-injector run

install-protobuf:
	curl -sSLo /tmp/${PROTOC_ZIP} https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/${PROTOC_ZIP}
	unzip -o /tmp/${PROTOC_ZIP} -d ${GOPATH} bin/protoc
	unzip -o /tmp/${PROTOC_ZIP} -d ${GOPATH} 'include/*'
	rm -f /tmp/${PROTOC_ZIP}

install-golangci-lint:
ifneq ($(GOLANGCI_LINT_VERSION),$(GOLANGCI_LINT_INSTALLED_VERSION))
	$(info golangci-lint version $(GOLANGCI_LINT_VERSION) is not installed or version differ (v$(GOLANGCI_LINT_VERSION) != $(GOLANGCI_LINT_INSTALLED_VERSION)))
	$(info installing golangci-lint v$(GOLANGCI_LINT_VERSION)...)
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOBIN) v$(GOLANGCI_LINT_VERSION)
endif

install-kubebuilder:
# download kubebuilder and install locally.
	curl -sSLo $(GOBIN)/kubebuilder https://go.kubebuilder.io/dl/latest/$(GOOS)/$(GOARCH)
	chmod u+x $(GOBIN)/kubebuilder
# download setup-envtest and install related binaries locally
	go install -v sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
# setup-envtest use -p path $(KUBERNETES_MAJOR_VERSION).x

install-helm:
ifneq ($(HELM_INSTALLED_VERSION),$(HELM_VERSION))
	$(info helm version $(HELM_VERSION) is not installed or version differ ($(HELM_VERSION) != $(HELM_INSTALLED_VERSION)))
	$(info installing helm $(HELM_VERSION)...)
	curl -sSLo /tmp/helm.tar.gz "https://get.helm.sh/helm-$(HELM_VERSION)-$(GOOS)-$(GOARCH).tar.gz"
	tar -xvzf /tmp/helm.tar.gz --directory=$(GOBIN) --strip-components=1 $(GOOS)-$(GOARCH)/helm
	rm /tmp/helm.tar.gz
endif

# install controller-gen expected version
install-controller-gen:
ifneq ($(CONTROLLER_GEN_INSTALLED_VERSION),$(CONTROLLER_GEN_VERSION))
	$(info controller-gen version $(CONTROLLER_GEN_VERSION) is not installed or version differ ($(CONTROLLER_GEN_VERSION) != $(CONTROLLER_GEN_INSTALLED_VERSION)))
	$(info installing controller-gen $(CONTROLLER_GEN_VERSION)...)
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	CGO_ENABLED=0 go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_GEN_VERSION) ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
endif

install-datadog-ci:
	curl -L --fail "https://github.com/DataDog/datadog-ci/releases/latest/download/datadog-ci_$(GOOS)-x64" --output "$(GOBIN)/datadog-ci" && chmod u+x $(GOBIN)/datadog-ci

install-mockery:
# recommended way to install mockery is through their released binaries, NOT go install...
# https://vektra.github.io/mockery/installation/#github-release
ifneq ($(MOCKERY_INSTALLED_VERSION),v$(MOCKERY_VERSION))
	$(info mockery version $(MOCKERY_VERSION) is not installed or version differ (v$(MOCKERY_VERSION) != $(MOCKERY_INSTALLED_VERSION)))
	$(info installing mockery v$(MOCKERY_VERSION)...)
	curl -sSLo /tmp/mockery.tar.gz https://github.com/vektra/mockery/releases/download/v$(MOCKERY_VERSION)/mockery_$(MOCKERY_VERSION)_$(GOOS)_$(GOARCH).tar.gz
	tar -xvzf /tmp/mockery.tar.gz --directory=$(GOBIN) mockery
	rm /tmp/mockery.tar.gz
endif

YAMLFMT_ARCH = $(GOARCH)
ifeq (amd64,$(GOARCH))
YAMLFMT_ARCH = x86_64
endif

install-yamlfmt:
ifeq (,$(wildcard $(GOBIN)/yamlfmt))
	$(info installing yamlfmt...)
	curl -sSLo /tmp/yamlfmt.tar.gz https://github.com/google/yamlfmt/releases/download/v0.9.0/yamlfmt_0.9.0_$(GOOS)_$(YAMLFMT_ARCH).tar.gz
	tar -xvzf /tmp/yamlfmt.tar.gz --directory=$(GOBIN) yamlfmt
	rm /tmp/yamlfmt.tar.gz
endif

install-watchexec:
ifeq (,$(wildcard $(GOBIN)/gow))
	$(info installing watchexec...)
	brew install watchexec
endif

EXISTING_NAMESPACE = $(shell $(KUBECTL) get ns datadog-agent -oname || echo "")

lima-install-datadog-agent:
ifeq (true,$(INSTALL_DATADOG_AGENT))
ifeq (,$(EXISTING_NAMESPACE))
	$(KUBECTL) create ns datadog-agent
	helm repo add --force-update datadoghq https://helm.datadoghq.com
	helm install -n datadog-agent my-datadog-operator datadoghq/datadog-operator
	$(KUBECTL) create secret -n datadog-agent generic datadog-secret --from-literal api-key=${STAGING_DATADOG_API_KEY} --from-literal app-key=${STAGING_DATADOG_APP_KEY}
endif
endif
	$(KUBECTL) apply -f - < examples/datadog-agent.yaml

open-dd:
ifeq (true,$(INSTALL_DATADOG_AGENT))
ifdef STAGING_DD_SITE
	open "${STAGING_DD_SITE}/infrastructure?host=lima-$(LIMA_INSTANCE)&tab=details"
else
	@echo "You need to define STAGING_DD_SITE in your .zshrc or similar to use this feature"
endif
endif
