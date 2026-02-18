#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/versions.env"

GOBIN="${GOBIN:-$(go env GOBIN)}"
GOBIN="${GOBIN:-$(go env GOPATH)/bin}"

if command -v helm &>/dev/null; then
  INSTALLED=$(helm version --template="{{ .Version }}" 2>/dev/null | awk '{ print $1 }' || echo "")
  if [ "$INSTALLED" = "$HELM_VERSION" ]; then
    exit 0
  fi
fi

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
esac

echo "Installing helm ${HELM_VERSION}..."
curl -sSLo /tmp/helm.tar.gz "https://get.helm.sh/helm-${HELM_VERSION}-${OS}-${ARCH}.tar.gz"
tar -xzf /tmp/helm.tar.gz --directory="$GOBIN" --strip-components=1 "${OS}-${ARCH}/helm"
rm /tmp/helm.tar.gz
