# mk/ci.mk — CI-specific targets, Python tasks, dependency management, and Datadog agent.

# ------------------------------------------------------------------------------
# Minikube (CI)
# ------------------------------------------------------------------------------

MINIKUBE_CPUS   ?= 6
MINIKUBE_MEMORY ?= 28672

.PHONY: ci-install-minikube
ci-install-minikube: ## Install and start Minikube (CI only)
	curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube_latest_amd64.deb
	sudo dpkg -i minikube_latest_amd64.deb
	minikube start --cpus='$(MINIKUBE_CPUS)' --memory='$(MINIKUBE_MEMORY)' --vm-driver=docker --container-runtime=containerd --kubernetes-version=${KUBERNETES_VERSION}
	minikube status

# ------------------------------------------------------------------------------
# Python venv and invoke tasks
# ------------------------------------------------------------------------------

.PHONY: venv
venv:
	test -d .venv || python3 -m venv .venv
	. .venv/bin/activate; pip install -qr tasks/requirements.txt

.PHONY: header
header: venv ## Check license headers
	. .venv/bin/activate; inv header-check

.PHONY: header-fix
header-fix: ## Fix license headers (run twice to converge)
	-$(MAKE) header
	$(MAKE) header

.PHONY: license
license: venv ## Check third-party licenses
	. .venv/bin/activate; inv license-check

# ------------------------------------------------------------------------------
# Dependency management
# ------------------------------------------------------------------------------

.PHONY: godeps
godeps: ## Tidy and vendor Go dependencies
	go mod tidy; go mod vendor

.PHONY: update-deps
update-deps: ## Update pinned Python dependencies
	@echo "Updating Python dependencies..."
	@pip install -q uv
	@uv pip compile --python-platform linux tasks/requirements.in -o tasks/requirements.txt
	@echo "Updated tasks/requirements.txt"
	@echo "Please commit both tasks/requirements.in and tasks/requirements.txt"

.PHONY: deps
deps: godeps license ## Tidy Go deps and check licenses

# ------------------------------------------------------------------------------
# Release
# ------------------------------------------------------------------------------

.SILENT: release
.PHONY: release
release: ## Create a release
	VERSION=$(VERSION) ./tasks/release.sh

# ------------------------------------------------------------------------------
# Datadog coverage upload
# ------------------------------------------------------------------------------

.PHONY: coverage-upload
coverage-upload: $(DATADOG_CI) ## Upload Go coverage report to Datadog
	$(DATADOG_CI) coverage upload --format go-coverprofile --flags "type:unit-tests" cover.profile

# ------------------------------------------------------------------------------
# Datadog agent (Lima)
# ------------------------------------------------------------------------------

EXISTING_NAMESPACE = $(shell $(KUBECTL) get ns datadog-agent -oname 2>/dev/null || echo "")

.PHONY: lima-install-datadog-agent
lima-install-datadog-agent: ## Install Datadog agent into Lima cluster
ifeq (true,$(INSTALL_DATADOG_AGENT))
ifeq (,$(EXISTING_NAMESPACE))
	$(KUBECTL) create ns datadog-agent
	helm repo add --force-update datadoghq https://helm.datadoghq.com
	helm install -n datadog-agent my-datadog-operator datadoghq/datadog-operator
	$(KUBECTL) create secret -n datadog-agent generic datadog-secret --from-literal api-key=${STAGING_DATADOG_API_KEY} --from-literal app-key=${STAGING_DATADOG_APP_KEY}
endif
	$(KUBECTL) apply -f - < examples/datadog-agent.yaml
endif

.PHONY: open-dd
open-dd: ## Open Datadog infrastructure page for Lima host
ifeq (true,$(INSTALL_DATADOG_AGENT))
ifdef STAGING_DD_SITE
	open "${STAGING_DD_SITE}/infrastructure?host=lima-$(LIMA_INSTANCE)&tab=details"
else
	@echo "You need to define STAGING_DD_SITE in your .zshrc or similar to use this feature"
endif
endif
