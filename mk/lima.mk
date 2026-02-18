# mk/lima.mk — Lima cluster management targets.

.PHONY: lima-all
lima-all: lima-start lima-install-datadog-agent lima-install-cert-manager lima-push-all lima-install ## Create Lima cluster and deploy chaos-controller
	kubens chaos-engineering

.PHONY: lima-redeploy
lima-redeploy: lima-push-all lima-install lima-restart ## Rebuild images, re-install chart, restart pods

.PHONY: lima-install-cert-manager
lima-install-cert-manager: ## Install cert-manager chart
	$(KUBECTL) apply -f https://github.com/jetstack/cert-manager/releases/download/v1.9.1/cert-manager.yaml
	$(KUBECTL) -n cert-manager rollout status deployment/cert-manager-webhook --timeout=180s

.PHONY: lima-install-demo
lima-install-demo: ## Deploy demo workloads
	$(KUBECTL) apply -f - < ./examples/namespace.yaml
	$(KUBECTL) apply -f - < ./examples/demo.yaml
	$(KUBECTL) -n chaos-demo rollout status deployment/demo-curl --timeout=60s
	$(KUBECTL) -n chaos-demo rollout status deployment/demo-nginx --timeout=60s

.PHONY: lima-install
lima-install: manifests ## Install CRDs and controller into Lima k3s cluster
	helm template \
		--set=controller.version=$(CONTAINER_VERSION) \
		--set=controller.metricsSink=$(LIMA_INSTALL_SINK) \
		--set=controller.profilerSink=$(LIMA_INSTALL_SINK) \
		--set=controller.tracerSink=$(LIMA_INSTALL_SINK) \
		--values ./chart/values/$(HELM_VALUES) \
		./chart | $(KUBECTL) apply -f -
ifneq (local.yaml,$(HELM_VALUES))
	$(KUBECTL) -n chaos-engineering rollout status deployment/chaos-controller --timeout=60s
endif

.PHONY: lima-uninstall
lima-uninstall: ## Uninstall CRDs and controller from Lima k3s cluster
	helm template --set=skipNamespace=true --values ./chart/values/$(HELM_VALUES) ./chart | $(KUBECTL) delete -f -

.PHONY: lima-restart
lima-restart: ## Restart the chaos-controller pod
ifneq (local.yaml,$(HELM_VALUES))
	$(KUBECTL) -n chaos-engineering rollout restart deployment/chaos-controller
	$(KUBECTL) -n chaos-engineering rollout status deployment/chaos-controller --timeout=60s
endif

.PHONY: lima-kubectx-clean
lima-kubectx-clean: ## Remove Lima references from kubectl config
	-kubectl config delete-cluster ${LIMA_PROFILE}
	-kubectl config delete-context ${LIMA_PROFILE}
	-kubectl config delete-user ${LIMA_PROFILE}
	kubectl config unset current-context

.PHONY: lima-kubectx
lima-kubectx:
	limactl shell $(LIMA_INSTANCE) sudo sed 's/default/lima/g' /etc/rancher/k3s/k3s.yaml > ~/.kube/config_lima
	KUBECONFIG=${KUBECONFIG}:~/.kube/config:~/.kube/config_lima kubectl config view --flatten > /tmp/config
	rm ~/.kube/config_lima
	mv /tmp/config ~/.kube/config
	chmod 600 ~/.kube/config
	kubectx ${LIMA_PROFILE}

.PHONY: lima-stop
lima-stop: ## Stop and delete the Lima cluster
	limactl stop -f $(LIMA_INSTANCE)
	limactl delete $(LIMA_INSTANCE)
	$(MAKE) lima-kubectx-clean

.PHONY: lima-start
lima-start: lima-kubectx-clean ## Start the Lima cluster
	LIMA_CGROUPS=${LIMA_CGROUPS} LIMA_CONFIG=${LIMA_CONFIG} LIMA_INSTANCE=${LIMA_INSTANCE} ./scripts/lima_start.sh
	$(MAKE) lima-kubectx

# Longhorn provides an alternative StorageClass for reliable disk throttling.
# Default local-path uses tmpfs which leads to virtual unnamed devices (major 0)
# that blkio does not support.
.PHONY: lima-install-longhorn
lima-install-longhorn: ## Install Longhorn storage driver
	$(KUBECTL) apply -f https://raw.githubusercontent.com/longhorn/longhorn/v1.4.0/deploy/longhorn.yaml

# ------------------------------------------------------------------------------
# Local development helpers
# ------------------------------------------------------------------------------

.PHONY: _pre_local
_pre_local: generate manifests
	@$(shell $(KUBECTL) get deploy chaos-controller 2> /dev/null)
ifeq (0,$(.SHELLSTATUS))
	-$(MAKE) lima-uninstall HELM_VALUES=dev.yaml
	$(MAKE) lima-install HELM_VALUES=local.yaml
	$(KUBECTL) -n chaos-engineering get cm chaos-controller -oyaml | yq '.data["config.yaml"]' > .local.yaml
	yq -i '.controller.webhook.certDir = "chart/certs"' .local.yaml
else
	@echo "Chaos controller is not installed, skipped!"
endif

.PHONY: debug
debug: _pre_local ## Prepare environment for IDE debugging
	@echo "now you can launch through vs-code or your favorite IDE a controller in debug with appropriate configuration (--config=chart/values/local.yaml + CONTROLLER_NODE_NAME=local)"

.PHONY: run
run: ## Run the controller locally
	CONTROLLER_NODE_NAME=local go run . --config=.local.yaml

.PHONY: watch
watch: _pre_local install-watchexec ## Watch for changes and auto-rebuild
	watchexec make SKIP_EBPF=true lima-push-injector run
