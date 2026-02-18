# mk/test.mk — Test targets (unit, e2e, chaosli).

# https://onsi.github.io/ginkgo/#recommended-continuous-integration-configuration
GINKGO_PROCS ?= 4

# Additional args to provide to test runner (ginkgo).
# Examples:
#   make test TEST_ARGS=--until-it-fails
#   make test TEST_ARGS=injector
TEST_ARGS ?=

DD_ENV := local
ifeq (true,$(CI))
DD_ENV := ci
endif

SKIP_DEPLOY ?=

# Internal ginkgo runner — not meant to be called directly.
.PHONY: _ginkgo_test
_ginkgo_test:
	-go run github.com/onsi/ginkgo/v2/ginkgo --fail-on-pending --keep-going --vv \
		--cover --coverprofile=cover.profile --randomize-all \
		--race --trace --json-report=report-$(GO_TEST_REPORT_NAME).json --junit-report=report-$(GO_TEST_REPORT_NAME).xml \
		--compilers=$(GINKGO_PROCS) --procs=$(GINKGO_PROCS) \
		--poll-progress-after=10s --poll-progress-interval=10s \
		$(GINKGO_TEST_ARGS) \
			&& touch report-$(GO_TEST_REPORT_NAME)-succeed
ifneq (true,$(GO_TEST_SKIP_UPLOAD))
ifdef DATADOG_API_KEY
ifneq (,$(shell which datadog-ci))
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
	[ -f report-$(GO_TEST_REPORT_NAME)-succeed ] && rm -f report-$(GO_TEST_REPORT_NAME)-succeed || exit 1

.PHONY: test
test: generate manifests ## Run unit tests
	$(if $(GOPATH),,$(error GOPATH is not set. Please set GOPATH before running make test))
	$(MAKE) _ginkgo_test GO_TEST_REPORT_NAME=$@ \
		GINKGO_TEST_ARGS="-r --skip-package=controllers --randomize-suites --timeout=10m $(TEST_ARGS)"

.PHONY: e2e-test
e2e-test: generate manifests ## Run e2e tests (against a real cluster)
ifneq (true,$(SKIP_DEPLOY))
	$(MAKE) lima-install HELM_VALUES=ci.yaml
endif
	E2E_TEST_CLUSTER_NAME=$(E2E_TEST_CLUSTER_NAME) E2E_TEST_KUBECTL_CONTEXT=$(E2E_TEST_KUBECTL_CONTEXT) $(MAKE) _ginkgo_test GO_TEST_REPORT_NAME=$@ \
		GINKGO_TEST_ARGS="--flake-attempts=3 --timeout=25m controllers"

.PHONY: chaosli-test
chaosli-test: ## Test chaosli API portability
	docker buildx build -f ./cli/chaosli/chaosli.DOCKERFILE -t test-chaosli-image .
