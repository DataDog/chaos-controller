# mk/lint.mk — Formatting, vetting, and linting targets.

.PHONY: fmt
fmt: ## Run go fmt
	go fmt ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: lint
lint: $(GOLANGCI_LINT) ## Run golangci-lint
	GOOS=linux $(GOLANGCI_LINT) run -E ginkgolinter ./...
	GOOS=linux $(GOLANGCI_LINT) run

# ------------------------------------------------------------------------------
# Spellcheck
# ------------------------------------------------------------------------------

.PHONY: spellcheck-deps
spellcheck-deps:
ifeq (, $(shell which npm))
	@{ \
		echo "please install npm or run 'make spellcheck-docker' for a slow but platform-agnostic run"; \
		exit 1; \
	}
endif
ifeq (, $(shell which mdspell))
	@{ \
		echo "installing mdspell through npm -g... (might require sudo run)"; \
		npm -g i markdown-spellcheck; \
	}
endif

.PHONY: spellcheck
spellcheck: spellcheck-deps ## Spellcheck markdown files
	mdspell --en-us --ignore-acronyms --ignore-numbers \
		$(shell find . -name vendor -prune -o -name '*.md' -print);

.PHONY: spellcheck-docker
spellcheck-docker: ## Spellcheck via Docker (no local npm needed)
	docker run --rm -ti -v $(shell pwd):/workdir tmaier/markdown-spellcheck:latest \
		--ignore-numbers --ignore-acronyms --en-us \
		$(shell find . -name vendor -prune -o -name '*.md' -print);
