# mk/tools.mk — Isolated tool management with stamp-based version tracking.
# All tools install into $(LOCALBIN). Stamp files in $(LOCALSTAMP) track
# installed versions so Make skips re-installs when the version hasn't changed.

$(LOCALBIN) $(LOCALSTAMP):
	mkdir -p $@

# ------------------------------------------------------------------------------
# Tool versions (single source of truth)
# ------------------------------------------------------------------------------

GOLANGCI_LINT_VERSION  := 2.8.0
CONTROLLER_GEN_VERSION := v0.19.0
MOCKERY_VERSION        := 2.53.5
YAMLFMT_VERSION        := 0.9.0
HELM_VERSION           := v3.19.0
PROTOC_VERSION         := 3.17.3
PROTOC_GEN_GO_VERSION  := v1.36.11 # Must be aligned with the protobuf version in go.mod.
PROTOC_GEN_GO_GRPC_VERSION := v1.1.0

# ------------------------------------------------------------------------------
# Platform detection
# ------------------------------------------------------------------------------

GOOS   ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

MOCKERY_ARCH := $(GOARCH)
ifeq (amd64,$(GOARCH))
MOCKERY_ARCH := x86_64
endif

YAMLFMT_ARCH := $(GOARCH)
ifeq (amd64,$(GOARCH))
YAMLFMT_ARCH := x86_64
endif

PROTOC_OS ?= osx

# ------------------------------------------------------------------------------
# Tool binary paths
# ------------------------------------------------------------------------------

GOLANGCI_LINT    := $(LOCALBIN)/golangci-lint
CONTROLLER_GEN   := $(LOCALBIN)/controller-gen
MOCKERY          := $(LOCALBIN)/mockery
YAMLFMT          := $(LOCALBIN)/yamlfmt
HELM             := $(LOCALBIN)/helm
PROTOC           := $(LOCALBIN)/protoc
PROTOC_GEN_GO    := $(LOCALBIN)/protoc-gen-go
PROTOC_GEN_GO_GRPC := $(LOCALBIN)/protoc-gen-go-grpc
KUBEBUILDER      := $(LOCALBIN)/kubebuilder
DATADOG_CI       := $(LOCALBIN)/datadog-ci
WATCHEXEC        := $(LOCALBIN)/watchexec

PROTOC_INCLUDE   := $(CURDIR)/.tools/include

# ------------------------------------------------------------------------------
# Stamp file paths (keyed by version)
# ------------------------------------------------------------------------------

GOLANGCI_LINT_STAMP    := $(LOCALSTAMP)/golangci-lint-$(GOLANGCI_LINT_VERSION)
CONTROLLER_GEN_STAMP   := $(LOCALSTAMP)/controller-gen-$(CONTROLLER_GEN_VERSION)
MOCKERY_STAMP          := $(LOCALSTAMP)/mockery-$(MOCKERY_VERSION)
YAMLFMT_STAMP          := $(LOCALSTAMP)/yamlfmt-$(YAMLFMT_VERSION)
HELM_STAMP             := $(LOCALSTAMP)/helm-$(HELM_VERSION)
PROTOC_STAMP           := $(LOCALSTAMP)/protoc-$(PROTOC_VERSION)
PROTOC_GEN_GO_STAMP    := $(LOCALSTAMP)/protoc-gen-go-$(PROTOC_GEN_GO_VERSION)
PROTOC_GEN_GO_GRPC_STAMP := $(LOCALSTAMP)/protoc-gen-go-grpc-$(PROTOC_GEN_GO_GRPC_VERSION)
KUBEBUILDER_STAMP      := $(LOCALSTAMP)/kubebuilder
DATADOG_CI_STAMP       := $(LOCALSTAMP)/datadog-ci
WATCHEXEC_STAMP        := $(LOCALSTAMP)/watchexec

# ------------------------------------------------------------------------------
# golangci-lint
# ------------------------------------------------------------------------------

$(GOLANGCI_LINT_STAMP): | $(LOCALBIN) $(LOCALSTAMP)
	@rm -f $(LOCALSTAMP)/golangci-lint-*
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(LOCALBIN) v$(GOLANGCI_LINT_VERSION)
	@touch $@

$(GOLANGCI_LINT): $(GOLANGCI_LINT_STAMP)

.PHONY: install-golangci-lint
install-golangci-lint: $(GOLANGCI_LINT) ## Install golangci-lint

# ------------------------------------------------------------------------------
# controller-gen
# ------------------------------------------------------------------------------

$(CONTROLLER_GEN_STAMP): | $(LOCALBIN) $(LOCALSTAMP)
	@rm -f $(LOCALSTAMP)/controller-gen-*
	GOBIN=$(LOCALBIN) CGO_ENABLED=0 go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_GEN_VERSION)
	@touch $@

$(CONTROLLER_GEN): $(CONTROLLER_GEN_STAMP)

.PHONY: install-controller-gen
install-controller-gen: $(CONTROLLER_GEN) ## Install controller-gen

# ------------------------------------------------------------------------------
# mockery
# ------------------------------------------------------------------------------

$(MOCKERY_STAMP): | $(LOCALBIN) $(LOCALSTAMP)
	@rm -f $(LOCALSTAMP)/mockery-*
	curl -sSLo /tmp/mockery.tar.gz https://github.com/vektra/mockery/releases/download/v$(MOCKERY_VERSION)/mockery_$(MOCKERY_VERSION)_$(GOOS)_$(MOCKERY_ARCH).tar.gz
	tar -xzf /tmp/mockery.tar.gz --directory=$(LOCALBIN) mockery
	rm -f /tmp/mockery.tar.gz
	@touch $@

$(MOCKERY): $(MOCKERY_STAMP)

.PHONY: install-mockery
install-mockery: $(MOCKERY) ## Install mockery

# ------------------------------------------------------------------------------
# yamlfmt
# ------------------------------------------------------------------------------

$(YAMLFMT_STAMP): | $(LOCALBIN) $(LOCALSTAMP)
	@rm -f $(LOCALSTAMP)/yamlfmt-*
	curl -sSLo /tmp/yamlfmt.tar.gz https://github.com/google/yamlfmt/releases/download/v$(YAMLFMT_VERSION)/yamlfmt_$(YAMLFMT_VERSION)_$(GOOS)_$(YAMLFMT_ARCH).tar.gz
	tar -xzf /tmp/yamlfmt.tar.gz --directory=$(LOCALBIN) yamlfmt
	rm -f /tmp/yamlfmt.tar.gz
	@touch $@

$(YAMLFMT): $(YAMLFMT_STAMP)

.PHONY: install-yamlfmt
install-yamlfmt: $(YAMLFMT) ## Install yamlfmt

# ------------------------------------------------------------------------------
# helm
# ------------------------------------------------------------------------------

$(HELM_STAMP): | $(LOCALBIN) $(LOCALSTAMP)
	@rm -f $(LOCALSTAMP)/helm-*
	curl -sSLo /tmp/helm.tar.gz "https://get.helm.sh/helm-$(HELM_VERSION)-$(GOOS)-$(GOARCH).tar.gz"
	tar -xzf /tmp/helm.tar.gz --directory=$(LOCALBIN) --strip-components=1 $(GOOS)-$(GOARCH)/helm
	rm -f /tmp/helm.tar.gz
	@touch $@

$(HELM): $(HELM_STAMP)

.PHONY: install-helm
install-helm: $(HELM) ## Install helm

# ------------------------------------------------------------------------------
# protoc
# ------------------------------------------------------------------------------

PROTOC_ZIP := protoc-$(PROTOC_VERSION)-$(PROTOC_OS)-x86_64.zip

$(PROTOC_STAMP): | $(LOCALBIN) $(LOCALSTAMP)
	@rm -f $(LOCALSTAMP)/protoc-*
	curl -sSLo /tmp/$(PROTOC_ZIP) https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOC_VERSION)/$(PROTOC_ZIP)
	unzip -o /tmp/$(PROTOC_ZIP) -d $(CURDIR)/.tools bin/protoc
	unzip -o /tmp/$(PROTOC_ZIP) -d $(CURDIR)/.tools 'include/*'
	rm -f /tmp/$(PROTOC_ZIP)
	@touch $@

$(PROTOC): $(PROTOC_STAMP)

.PHONY: install-protobuf
install-protobuf: $(PROTOC) ## Install protoc compiler

# ------------------------------------------------------------------------------
# protoc-gen-go
# ------------------------------------------------------------------------------

$(PROTOC_GEN_GO_STAMP): | $(LOCALBIN) $(LOCALSTAMP)
	@rm -f $(LOCALSTAMP)/protoc-gen-go-*
	GOBIN=$(LOCALBIN) go install google.golang.org/protobuf/cmd/protoc-gen-go@$(PROTOC_GEN_GO_VERSION)
	@touch $@

$(PROTOC_GEN_GO): $(PROTOC_GEN_GO_STAMP)

# ------------------------------------------------------------------------------
# protoc-gen-go-grpc
# ------------------------------------------------------------------------------

$(PROTOC_GEN_GO_GRPC_STAMP): | $(LOCALBIN) $(LOCALSTAMP)
	@rm -f $(LOCALSTAMP)/protoc-gen-go-grpc-*
	GOBIN=$(LOCALBIN) go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@$(PROTOC_GEN_GO_GRPC_VERSION)
	@touch $@

$(PROTOC_GEN_GO_GRPC): $(PROTOC_GEN_GO_GRPC_STAMP)

# ------------------------------------------------------------------------------
# kubebuilder + setup-envtest
# ------------------------------------------------------------------------------

$(KUBEBUILDER_STAMP): | $(LOCALBIN) $(LOCALSTAMP)
	@rm -f $(LOCALSTAMP)/kubebuilder
	curl -sSLo $(LOCALBIN)/kubebuilder https://go.kubebuilder.io/dl/latest/$(GOOS)/$(GOARCH)
	chmod u+x $(LOCALBIN)/kubebuilder
	GOBIN=$(LOCALBIN) go install -v sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
	@touch $@

$(KUBEBUILDER): $(KUBEBUILDER_STAMP)

.PHONY: install-kubebuilder
install-kubebuilder: $(KUBEBUILDER) ## Install kubebuilder and setup-envtest

# ------------------------------------------------------------------------------
# datadog-ci
# ------------------------------------------------------------------------------

$(DATADOG_CI_STAMP): | $(LOCALBIN) $(LOCALSTAMP)
	@rm -f $(LOCALSTAMP)/datadog-ci
	curl -L --fail "https://github.com/DataDog/datadog-ci/releases/latest/download/datadog-ci_$(GOOS)-x64" --output "$(LOCALBIN)/datadog-ci"
	chmod u+x $(LOCALBIN)/datadog-ci
	@touch $@

$(DATADOG_CI): $(DATADOG_CI_STAMP)

.PHONY: install-datadog-ci
install-datadog-ci: $(DATADOG_CI) ## Install datadog-ci

# ------------------------------------------------------------------------------
# watchexec (macOS only via brew; falls back to checking PATH)
# ------------------------------------------------------------------------------

$(WATCHEXEC_STAMP): | $(LOCALBIN) $(LOCALSTAMP)
	@rm -f $(LOCALSTAMP)/watchexec
	@if ! command -v watchexec >/dev/null 2>&1; then \
		echo "installing watchexec via brew..."; \
		brew install watchexec; \
	fi
	@touch $@

.PHONY: install-watchexec
install-watchexec: $(WATCHEXEC_STAMP) ## Install watchexec

# ------------------------------------------------------------------------------
# install-go (delegates to scripts/install-go)
# ------------------------------------------------------------------------------

.PHONY: install-go
install-go: ## Install Go (for Docker/CI builds)
	BUILDGOVERSION=$(BUILDGOVERSION) ./scripts/install-go

# ------------------------------------------------------------------------------
# install-tools / clean-tools
# ------------------------------------------------------------------------------

.PHONY: install-tools
install-tools: $(GOLANGCI_LINT) $(CONTROLLER_GEN) $(MOCKERY) $(YAMLFMT) $(HELM) $(PROTOC) $(PROTOC_GEN_GO) $(PROTOC_GEN_GO_GRPC) $(KUBEBUILDER) $(DATADOG_CI) ## Install all tools

.PHONY: clean-tools
clean-tools: ## Remove all locally installed tools
	rm -rf $(CURDIR)/.tools
