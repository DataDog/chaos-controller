#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/versions.env"

MINIKUBE_CPUS="${MINIKUBE_CPUS:-6}"
MINIKUBE_MEMORY="${MINIKUBE_MEMORY:-28672}"
KUBERNETES_MAJOR_VERSION="${KUBERNETES_MAJOR_VERSION:-1.28}"
KUBERNETES_VERSION="${KUBERNETES_VERSION:-v${KUBERNETES_MAJOR_VERSION}.0}"

curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube_latest_amd64.deb
sudo dpkg -i minikube_latest_amd64.deb
minikube start \
  --cpus="${MINIKUBE_CPUS}" \
  --memory="${MINIKUBE_MEMORY}" \
  --vm-driver=docker \
  --container-runtime=containerd \
  --kubernetes-version="${KUBERNETES_VERSION}"
minikube status
