.PHONY: manager injector handler release generate generate-mocks clean-mocks all lima-push-all lima-redeploy lima-all e2e-test test lima-install manifests
.SILENT: release

# Lima requires to have images built on a specific namespace to be shared to the Kubernetes cluster when using containerd runtime
# https://github.com/abiosoft/colima#interacting-with-image-registry
CONTAINERD_REGISTRY_PREFIX ?= k8s.io

# Image URL to use all building/pushing image targets
MANAGER_IMAGE ?= ${CONTAINERD_REGISTRY_PREFIX}/chaos-controller:latest
INJECTOR_IMAGE ?= ${CONTAINERD_REGISTRY_PREFIX}/chaos-injector:latest
HANDLER_IMAGE ?= ${CONTAINERD_REGISTRY_PREFIX}/chaos-handler:latest

LIMA_PROFILE ?= lima
LIMA_CONFIG ?= lima
KUBECTL ?= limactl shell default sudo kubectl
PROTOC_VERSION = 3.17.3
PROTOC_OS ?= osx
PROTOC_ZIP = protoc-${PROTOC_VERSION}-${PROTOC_OS}-x86_64.zip
# you might also want to change ~/lima.yaml k3s version
KUBERNETES_MAJOR_VERSION ?= 1.26
KUBERNETES_VERSION ?= v$(KUBERNETES_MAJOR_VERSION).0
GOLANGCI_LINT_VERSION ?= 1.51.0
HELM_VERSION ?= 3.11.3
KUBEBUILDER_VERSION ?= 3.1.0
USE_VOLUMES ?= false

# expired disruption gc delay enable to speed up chaos controller disruption removal for e2e testing
# it's used to check if disruptions are deleted as expected as soon as the expiration delay occurs
EXPIRED_DISRUPTION_GC_DELAY ?= 10m

OS_ARCH=amd64
ifeq (arm64,$(shell uname -m))
OS_ARCH=arm64
endif

OS = $(shell go env GOOS)

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

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# we define target specific variables values https://www.gnu.org/software/make/manual/html_node/Target_002dspecific.html
injector handler: BINARY_PATH=./cli/$(BINARY_NAME)
manager: BINARY_PATH=main.go

docker-build-injector: IMAGE_TAG=$(INJECTOR_IMAGE)
docker-build-handler: IMAGE_TAG=$(HANDLER_IMAGE)
docker-build-manager: IMAGE_TAG=$(MANAGER_IMAGE)

docker-build-ebpf:
	docker build --platform linux/$(OS_ARCH) --build-arg ARCH=$(OS_ARCH) -t ebpf-builder-$(OS_ARCH) -f bin/ebpf-builder/Dockerfile ./bin/ebpf-builder/
	-rm -r bin/injector/ebpf/
ifeq (true,$(USE_VOLUMES))
# create a dummy container with volume to store files
# circleci remote docker does not allow to use volumes, locally we fallbakc to standard volume but can call this target with USE_VOLUMES=true to debug if necessary
# https://circleci.com/docs/building-docker-images/#mounting-folders
	-docker rm ebpf-volume
	-docker create --platform linux/$(OS_ARCH) -v /app --name ebpf-volume ebpf-builder-$(OS_ARCH) /bin/true
	-docker cp . ebpf-volume:/app
	-docker rm ebpf-builder
	docker run --platform linux/$(OS_ARCH) --volumes-from ebpf-volume --name=ebpf-builder ebpf-builder-$(OS_ARCH)
	docker cp ebpf-builder:/app/bin/injector/ebpf bin/injector/ebpf
	docker rm ebpf-builder
else
	docker run --rm --platform linux/$(OS_ARCH) -v $(shell pwd):/app ebpf-builder-$(OS_ARCH)
endif

lima-push-injector lima-push-handler lima-push-manager: FAKE_FOR=COMPLETION

_injector:;
_handler:;
_manager: generate

_docker-build-injector: docker-build-ebpf
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
	docker build --build-arg TARGETARCH=$(OS_ARCH) -t $$(IMAGE_TAG) -f bin/$(1)/Dockerfile ./bin/$(1)/
	docker save $$(IMAGE_TAG) -o ./bin/$(1)/$(1).tar.gz

lima-push-$(1): docker-build-$(1)
	limactl copy ./bin/$(1)/$(1).tar.gz default:/tmp/
	limactl shell default -- sudo k3s ctr i import /tmp/$(1).tar.gz

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
		--race --trace --json-report=$(GO_TEST_REPORT_NAME).json --junit-report=$(GO_TEST_REPORT_NAME).xml \
		$(GO_TEST_ARGS) \
			&& touch $(GO_TEST_REPORT_NAME)-succeed
# Try upload test reports if allowed and necessary prerequisites exists
ifneq (true,$(GO_TEST_SKIP_UPLOAD)) # you can bypass test upload
ifdef DATADOG_API_KEY # if no API key bypass is guaranteed
ifneq (,$(shell which datadog-ci)) # same if no test binary
	-DD_ENV=$(DD_ENV) datadog-ci junit upload --service chaos-controller $(GO_TEST_REPORT_NAME).xml
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
	[ -f $(GO_TEST_REPORT_NAME)-succeed ] && rm -f $(GO_TEST_REPORT_NAME)-succeed || exit 1

# Tests & CI
## Run unit tests
test: generate manifests
	$(MAKE) _ginkgo_test GO_TEST_REPORT_NAME=report-$@ \
		GO_TEST_ARGS="-r --compilers=4 --procs=4 --skip-package=controllers \
--randomize-suites --timeout=10m --poll-progress-after=5s --poll-progress-interval=5s"

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

## Run e2e tests (against a real cluster)
e2e-test: generate
	$(MAKE) lima-install EXPIRED_DISRUPTION_GC_DELAY=10s
	USE_EXISTING_CLUSTER=true $(MAKE) _ginkgo_test GO_TEST_REPORT_NAME=report-$@ \
		GO_TEST_ARGS="controllers --timeout=15m --poll-progress-after=15s \
--poll-progress-interval=15s"

# Test chaosli API portability
chaosli-test:
	docker build -f ./cli/chaosli/chaosli.DOCKERFILE -t test-chaosli-image .

# Go actions
## Generate manifests e.g. CRD, RBAC etc.
manifests: install-controller-gen
	$(CONTROLLER_GEN) rbac:roleName=chaos-controller-role crd:crdVersions=v1 paths="./..." output:crd:dir=./chart/templates/crds/ output:rbac:dir=./chart/templates/

## Run go fmt against code
fmt:
	go fmt ./...

## Run go vet against code
vet:
	go vet ./...

## Run golangci-lint against code
lint: install-lint-deps
	golangci-lint version
	golangci-lint run

## Generate code
generate: install-controller-gen
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths="./..."

# Lima actions
## Create a new lima cluster and deploy the chaos-controller into it
lima-all: lima-start lima-install-cert-manager lima-push-all lima-install
	kubens chaos-engineering

## Rebuild the chaos-controller images, re-install the chart and restart the chaos-controller pods
lima-redeploy: lima-push-all lima-install lima-restart

## Install cert-manager chart
lima-install-cert-manager:
	$(KUBECTL) apply -f https://github.com/jetstack/cert-manager/releases/download/v1.9.1/cert-manager.yaml
	$(KUBECTL) -n cert-manager rollout status deployment/cert-manager-webhook --timeout=180s

lima-install-demo:
	kubectl apply -f ./examples/namespace.yaml
	kubectl apply -f ./examples/demo.yaml
	$(KUBECTL) -n chaos-demo rollout status deployment/demo-curl --timeout=60s
	$(KUBECTL) -n chaos-demo rollout status deployment/demo-nginx --timeout=60s


## Install CRDs and controller into a lima k3s cluster
## In order to use already built images inside the containerd runtime
## we override images for all of our components to the expected namespace
lima-install: manifests
	helm template \
		--set controller.enableSafeguards=false \
		--set controller.expiredDisruptionGCDelay=${EXPIRED_DISRUPTION_GC_DELAY} \
		--values ./chart/values/dev.yaml \
		./chart | $(KUBECTL) apply -f -
	$(KUBECTL) -n chaos-engineering rollout status deployment/chaos-controller --timeout=60s

## Uninstall CRDs and controller from a lima k3s cluster
lima-uninstall:
	helm template ./chart | $(KUBECTL) delete -f -

## Restart the chaos-controller pod
lima-restart:
	$(KUBECTL) -n chaos-engineering rollout restart deployment chaos-controller
	$(KUBECTL) -n chaos-engineering rollout status deployment/chaos-controller --timeout=60s

## Remove lima references from kubectl config
lima-kubectx-clean:
	-kubectl config delete-cluster ${LIMA_PROFILE}
	-kubectl config delete-context ${LIMA_PROFILE}
	-kubectl config delete-user ${LIMA_PROFILE}
	kubectl config unset current-context

lima-kubectx:
	limactl shell default sudo sed 's/default/lima/g' /etc/rancher/k3s/k3s.yaml >> ~/.kube/config_lima
	KUBECONFIG=${KUBECONFIG}:~/.kube/config:~/.kube/config_lima kubectl config view --flatten > /tmp/config
	rm ~/.kube/config_lima
	mv /tmp/config ~/.kube/config
	chmod 600 ~/.kube/config
	kubectx ${LIMA_PROFILE}

## Stop and delete the lima cluster
lima-stop:
	limactl stop -f default
	limactl delete default
	$(MAKE) lima-kubectx-clean

## Start the lima cluster, pre-cleaning kubectl config
lima-start: lima-kubectx-clean
	LIMA_CGROUPS=${LIMA_CGROUPS} LIMA_CONFIG=${LIMA_CONFIG} ./scripts/lima_start.sh
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

header-check: venv
	source .venv/bin/activate; inv header-check

license-check: venv
	source .venv/bin/activate; inv license-check

godeps:
	go mod tidy; go mod vendor

deps: godeps license-check

install-protobuf:
	curl -sSLo /tmp/${PROTOC_ZIP} https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/${PROTOC_ZIP}
	unzip -o /tmp/${PROTOC_ZIP} -d ${GOPATH} bin/protoc
	unzip -o /tmp/${PROTOC_ZIP} -d ${GOPATH} 'include/*'
	rm -f /tmp/${PROTOC_ZIP}

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

generate-mocks: clean-mocks
	go install github.com/vektra/mockery/v2@v2.25.0
	go generate ./...
# First re-generate header, it should complain as just (re)generated mocks does not contains them
	-$(MAKE) header-check
# Then, re-generate header, it should succeed as now all files contains headers as expected, and command return with an happy exit code
	$(MAKE) header-check

release:
	VERSION=$(VERSION) ./tasks/release.sh

install-kubebuilder:
# download kubebuilder and install locally.
	curl -sSLo ${GOPATH}/bin/kubebuilder https://go.kubebuilder.io/dl/latest/$(OS)/$(OS_ARCH)
	chmod u+x ${GOPATH}/bin/kubebuilder
# download setup-envtest and install related binaries locally
	go install -v sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
# setup-envtest use -p path $(KUBERNETES_MAJOR_VERSION).x

install-helm:
	curl -sSLo /tmp/helm.tar.gz "https://get.helm.sh/helm-v$(HELM_VERSION)-$(OS)-$(OS_ARCH).tar.gz"
	tar -xvzf /tmp/helm.tar.gz --directory=${GOPATH}/bin --strip-components=1 $(OS)-$(OS_ARCH)/helm
	rm /tmp/helm.tar.gz

# find or download controller-gen
# download controller-gen if necessary
install-controller-gen:
ifeq (,$(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	CGO_ENABLED=0 go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.11.3 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

## install golangci-lint at the correct version if not
install-lint-deps:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v${GOLANGCI_LINT_VERSION}

install-datadog-ci:
	curl -L --fail "https://github.com/DataDog/datadog-ci/releases/latest/download/datadog-ci_$(OS)-x64" --output "$(GOBIN)/datadog-ci" && chmod u+x $(GOBIN)/datadog-ci
