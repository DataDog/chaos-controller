# mk/docker.mk — Docker build, save, and push targets for all components.

# Set container names for each component
docker-build-injector docker-build-only-injector: CONTAINER_NAME=$(INJECTOR_IMAGE)
docker-build-handler docker-build-only-handler: CONTAINER_NAME=$(HANDLER_IMAGE)
docker-build-manager docker-build-only-manager: CONTAINER_NAME=$(MANAGER_IMAGE)

lima-push-injector lima-push-handler lima-push-manager: FAKE_FOR=COMPLETION

# Generate manifests before building manager unless skipped (useful in CI)
ifneq ($(SKIP_GENERATE),true)
docker-build-manager docker-build-only-manager: generate
endif

# Template for per-component docker targets.
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

TARGETS := injector handler manager

$(foreach tgt,$(TARGETS),$(eval $(call TARGET_template,$(tgt))))

# Aggregate targets
.PHONY: docker-build-all docker-build-only-all lima-push-all minikube-load-all
docker-build-all: $(addprefix docker-build-,$(TARGETS)) ## Build and save all Docker images
docker-build-only-all: $(addprefix docker-build-only-,$(TARGETS)) ## Build all Docker images (no save)
lima-push-all: $(addprefix lima-push-,$(TARGETS)) ## Push all images to Lima
minikube-load-all: $(addprefix minikube-load-,$(TARGETS)) ## Load all images into Minikube

# Build chaosli CLI
.PHONY: chaosli
chaosli: ## Build chaosli CLI binary
	GOOS=darwin GOARCH=$(GOARCH) CGO_ENABLED=0 go build \
		-ldflags="-X github.com/DataDog/chaos-controller/cli/chaosli/cmd.Version=$(VERSION)" \
		-o bin/chaosli/chaosli_darwin_$(GOARCH) ./cli/chaosli/
