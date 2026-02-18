#!/usr/bin/env bash
set -euo pipefail

GOBIN="${GOBIN:-$(go env GOBIN)}"
GOBIN="${GOBIN:-$(go env GOPATH)/bin}"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
esac

echo "Installing kubebuilder..."
curl -sSLo "$GOBIN/kubebuilder" "https://go.kubebuilder.io/dl/latest/${OS}/${ARCH}"
chmod u+x "$GOBIN/kubebuilder"

echo "Installing setup-envtest..."
go install -v sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
