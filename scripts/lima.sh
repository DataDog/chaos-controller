#!/usr/bin/env bash
set -euo pipefail

# Default configuration — all overridable via env vars
LIMA_INSTANCE="${LIMA_INSTANCE:-$(whoami | tr '.' '-')}"
LIMA_PROFILE="${LIMA_PROFILE:-lima}"
LIMA_CONFIG="${LIMA_CONFIG:-lima}"
LIMA_CGROUPS="${LIMA_CGROUPS:-v1}"
KUBECTL="${KUBECTL:-limactl shell ${LIMA_INSTANCE} sudo kubectl}"
HELM_VALUES="${HELM_VALUES:-dev.yaml}"
CONTAINER_VERSION="${CONTAINER_VERSION:-$(git rev-parse HEAD)$(git diff --quiet || echo '-dirty')}"
STAGING_DATADOG_API_KEY="${STAGING_DATADOG_API_KEY:-}"
STAGING_DATADOG_APP_KEY="${STAGING_DATADOG_APP_KEY:-}"
STAGING_DD_SITE="${STAGING_DD_SITE:-}"

# KUBECTL wrapper using array splitting (avoids eval)
kctl() {
  local -a cmd
  read -ra cmd <<< "$KUBECTL"
  "${cmd[@]}" "$@"
}

# Determine if datadog agent should be installed
install_datadog_agent() {
  [ -n "$STAGING_DATADOG_API_KEY" ] && [ -n "$STAGING_DATADOG_APP_KEY" ]
}

# Determine metrics/profiler/tracer sink
get_sink() {
  if install_datadog_agent; then
    echo "datadog"
  else
    echo "noop"
  fi
}

cmd_start() {
  cmd_kubectx_clean

  # Start the lima instance
  limactl start --tty=false --name="${LIMA_INSTANCE}" - <"./${LIMA_CONFIG}.yaml"

  # For cgroups v1, reconfigure grub and restart the instance
  if [ "${LIMA_CGROUPS}" = "v1" ]; then
    echo "Reconfiguring lima instance with cgroups v1"
    limactl shell "${LIMA_INSTANCE}" sudo sed -i 's/GRUB_CMDLINE_LINUX=""/GRUB_CMDLINE_LINUX="systemd.unified_cgroup_hierarchy=0"/' /etc/default/grub
    limactl shell "${LIMA_INSTANCE}" sudo update-grub
    limactl shell "${LIMA_INSTANCE}" sudo reboot
    echo "Waiting for instance to reboot, it might take a while"
    sleep 10
    limactl stop "${LIMA_INSTANCE}"
    limactl start "${LIMA_INSTANCE}"
  fi

  cmd_kubectx
}

cmd_stop() {
  limactl stop -f "${LIMA_INSTANCE}"
  limactl delete "${LIMA_INSTANCE}"
  cmd_kubectx_clean
}

cmd_kubectx_clean() {
  kubectl config delete-cluster "${LIMA_PROFILE}" 2>/dev/null || true
  kubectl config delete-context "${LIMA_PROFILE}" 2>/dev/null || true
  kubectl config delete-user "${LIMA_PROFILE}" 2>/dev/null || true
  kubectl config unset current-context
}

cmd_kubectx() {
  limactl shell "${LIMA_INSTANCE}" sudo sed 's/default/lima/g' /etc/rancher/k3s/k3s.yaml > ~/.kube/config_lima
  KUBECONFIG="${KUBECONFIG:-}:~/.kube/config:~/.kube/config_lima" kubectl config view --flatten > /tmp/config
  rm ~/.kube/config_lima
  mv /tmp/config ~/.kube/config
  chmod 600 ~/.kube/config
  kubectx "${LIMA_PROFILE}"
}

cmd_install() {
  local sink
  sink="$(get_sink)"

  helm template \
    --set="controller.version=${CONTAINER_VERSION}" \
    --set="controller.metricsSink=${sink}" \
    --set="controller.profilerSink=${sink}" \
    --set="controller.tracerSink=${sink}" \
    --values "./chart/values/${HELM_VALUES}" \
    ./chart | kctl apply -f -

  # We can only wait for a controller if it exists, local.yaml does not deploy the controller
  if [ "${HELM_VALUES}" != "local.yaml" ]; then
    kctl -n chaos-engineering rollout status deployment/chaos-controller --timeout=60s
  fi
}

cmd_uninstall() {
  helm template --set=skipNamespace=true --values "./chart/values/${HELM_VALUES}" ./chart | kctl delete -f -
}

cmd_restart() {
  # We can only restart a controller if it exists, local.yaml does not deploy the controller
  if [ "${HELM_VALUES}" != "local.yaml" ]; then
    kctl -n chaos-engineering rollout restart deployment/chaos-controller
    kctl -n chaos-engineering rollout status deployment/chaos-controller --timeout=60s
  fi
}

cmd_install_cert_manager() {
  kctl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.9.1/cert-manager.yaml
  kctl -n cert-manager rollout status deployment/cert-manager-webhook --timeout=180s
}

cmd_install_demo() {
  kctl apply -f - < ./examples/namespace.yaml
  kctl apply -f - < ./examples/demo.yaml
  kctl -n chaos-demo rollout status deployment/demo-curl --timeout=60s
  kctl -n chaos-demo rollout status deployment/demo-nginx --timeout=60s
}

cmd_install_datadog_agent() {
  if ! install_datadog_agent; then
    return 0
  fi

  # Create namespace if it doesn't already exist
  if ! kctl get ns datadog-agent &>/dev/null; then
    kctl create ns datadog-agent
    helm repo add --force-update datadoghq https://helm.datadoghq.com
    helm install -n datadog-agent my-datadog-operator datadoghq/datadog-operator
    kctl create secret -n datadog-agent generic datadog-secret \
      --from-literal "api-key=${STAGING_DATADOG_API_KEY}" \
      --from-literal "app-key=${STAGING_DATADOG_APP_KEY}"
  fi
  kctl apply -f - < examples/datadog-agent.yaml
}

cmd_install_longhorn() {
  kctl apply -f https://raw.githubusercontent.com/longhorn/longhorn/v1.4.0/deploy/longhorn.yaml
}

cmd_pre_local() {
  if kctl get deploy chaos-controller -n chaos-engineering &>/dev/null; then
    # Uninstall using a non local value to ensure deployment is deleted
    (HELM_VALUES=dev.yaml; cmd_uninstall) || true
    (HELM_VALUES=local.yaml; cmd_install)
    kctl -n chaos-engineering get cm chaos-controller -oyaml | yq '.data["config.yaml"]' > .local.yaml
    yq -i '.controller.webhook.certDir = "chart/certs"' .local.yaml
  else
    echo "Chaos controller is not installed, skipped!"
  fi
}

cmd_open_dd() {
  if ! install_datadog_agent; then
    return 0
  fi

  if [ -z "${STAGING_DD_SITE}" ]; then
    echo "You need to define STAGING_DD_SITE in your .zshrc or similar to use this feature"
    return 0
  fi

  open "${STAGING_DD_SITE}/infrastructure?host=lima-${LIMA_INSTANCE}&tab=details"
}

# Main dispatch
cmd="${1:-}"
shift || true

case "$cmd" in
  start)                cmd_start ;;
  stop)                 cmd_stop ;;
  kubectx)              cmd_kubectx ;;
  kubectx-clean)        cmd_kubectx_clean ;;
  install)              cmd_install ;;
  uninstall)            cmd_uninstall ;;
  restart)              cmd_restart ;;
  install-cert-manager) cmd_install_cert_manager ;;
  install-demo)         cmd_install_demo ;;
  install-datadog-agent) cmd_install_datadog_agent ;;
  install-longhorn)     cmd_install_longhorn ;;
  pre-local)            cmd_pre_local ;;
  open-dd)              cmd_open_dd ;;
  *)
    echo "Usage: $0 {start|stop|kubectx|kubectx-clean|install|uninstall|restart|install-cert-manager|install-demo|install-datadog-agent|install-longhorn|pre-local|open-dd}"
    exit 1
    ;;
esac
