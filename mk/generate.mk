# mk/generate.mk — Code generation: CRDs, deepcopy, protobuf, mocks.

# ------------------------------------------------------------------------------
# Manifests (CRD, RBAC)
# ------------------------------------------------------------------------------

.PHONY: manifests
manifests: $(CONTROLLER_GEN) $(YAMLFMT) ## Generate CRD and RBAC manifests
	$(CONTROLLER_GEN) rbac:roleName=chaos-controller crd:crdVersions=v1 \
		paths="./..." \
		output:crd:dir=./chart/templates/generated/ \
		output:rbac:dir=./chart/templates/generated/
	$(YAMLFMT) chart/templates/generated

# ------------------------------------------------------------------------------
# controller-gen object generation (deepcopy)
# ------------------------------------------------------------------------------

.PHONY: generate
generate: $(CONTROLLER_GEN) ## Generate deepcopy helpers
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths="./..."

# ------------------------------------------------------------------------------
# Protobuf generation (DRY template for all proto packages)
# ------------------------------------------------------------------------------

PROTOBUF_PACKAGES := grpc/disruptionlistener dogfood/chaosdogfood

define protobuf_target
.PHONY: generate-$(notdir $(1))-protobuf
generate-$(notdir $(1))-protobuf: $(PROTOC) $(PROTOC_GEN_GO) $(PROTOC_GEN_GO_GRPC)
	cd $(1) && \
	PATH="$(LOCALBIN):$(PATH)" $(PROTOC) \
		--proto_path=. \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		$(notdir $(wildcard $(1)/*.proto))
endef

$(foreach pkg,$(PROTOBUF_PACKAGES),$(eval $(call protobuf_target,$(pkg))))

.PHONY: generate-protobuf
generate-protobuf: $(foreach p,$(PROTOBUF_PACKAGES),generate-$(notdir $(p))-protobuf) ## Generate all protobuf stubs

# ------------------------------------------------------------------------------
# Mocks
# ------------------------------------------------------------------------------

.PHONY: clean-mocks
clean-mocks: ## Remove generated mocks
	find . -type f -name "*mock*.go" -not -path "./vendor/*" -exec rm {} \;
	rm -rf mocks/

.PHONY: generate-mocks
generate-mocks: clean-mocks $(MOCKERY) ## Regenerate all mocks
	go generate ./...
	$(MAKE) header-fix
