# Makefile — Thin entry point. All logic lives in mk/*.mk modules.
#
# Usage:
#   make help          Show all available targets
#   make test          Run unit tests
#   make lint          Run linter
#   make docker-build-all  Build all container images
#
# Tools are automatically installed into .tools/bin/ on first use.
# Run `make clean-tools` to remove them.

.DEFAULT_GOAL := help

# ------------------------------------------------------------------------------
# Go / build metadata
# ------------------------------------------------------------------------------

NOW_ISO8601 := $(shell date -u +"%Y-%m-%dT%H:%M:%S")

# Change also github actions go build version if you change the version below
# https://github.com/DataDog/chaos-controller/blob/main/.github/workflows/ci.yml
BUILDGOVERSION := 1.25.6

# ------------------------------------------------------------------------------
# Container images
# ------------------------------------------------------------------------------

# Lima requires images built on a specific namespace to be shared to the
# Kubernetes cluster when using containerd runtime.
# https://github.com/abiosoft/colima#interacting-with-image-registry
CONTAINER_REGISTRY     ?= k8s.io
CONTAINER_TAG          ?= latest
CONTAINER_VERSION      ?= $(shell git rev-parse HEAD)$(shell git diff --quiet || echo '-dirty')
CONTAINER_BUILD_EXTRA_ARGS ?=

SIGN_IMAGE    ?= false
SKIP_GENERATE ?= false

MANAGER_IMAGE  ?= $(CONTAINER_REGISTRY)/chaos-controller
INJECTOR_IMAGE ?= $(CONTAINER_REGISTRY)/chaos-injector
HANDLER_IMAGE  ?= $(CONTAINER_REGISTRY)/chaos-handler

# ------------------------------------------------------------------------------
# Lima / Kubernetes
# ------------------------------------------------------------------------------

LIMA_PROFILE  ?= lima
LIMA_CONFIG   ?= lima
LIMA_INSTANCE ?= $(shell whoami | tr "." "-")

E2E_TEST_CLUSTER_NAME   ?= lima-$(LIMA_INSTANCE)
E2E_TEST_KUBECTL_CONTEXT ?= lima

KUBECTL ?= limactl shell $(LIMA_INSTANCE) sudo kubectl

KUBERNETES_MAJOR_VERSION ?= 1.28
KUBERNETES_VERSION       ?= v$(KUBERNETES_MAJOR_VERSION).0
USE_VOLUMES              ?= false

HELM_VALUES ?= dev.yaml

INSTALL_DATADOG_AGENT := false
LIMA_INSTALL_SINK     := noop
ifdef STAGING_DATADOG_API_KEY
ifdef STAGING_DATADOG_APP_KEY
INSTALL_DATADOG_AGENT := true
LIMA_INSTALL_SINK     := datadog
endif
endif

LIMA_CGROUPS := v1
ifeq (v2,$(CGROUPS))
LIMA_CGROUPS := v2
endif

# ------------------------------------------------------------------------------
# Local tool bin (isolated from system)
# ------------------------------------------------------------------------------

LOCALBIN   := $(CURDIR)/.tools/bin
LOCALSTAMP := $(CURDIR)/.tools/stamps
export PATH := $(LOCALBIN):$(PATH)

# ------------------------------------------------------------------------------
# Include modules
# ------------------------------------------------------------------------------

include mk/tools.mk
include mk/docker.mk
include mk/test.mk
include mk/generate.mk
include mk/lint.mk
include mk/lima.mk
include mk/ci.mk

# ------------------------------------------------------------------------------
# Help
# ------------------------------------------------------------------------------

.PHONY: help
help: ## Show this help
	@grep -hE '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-30s\033[0m %s\n", $$1, $$2}'
