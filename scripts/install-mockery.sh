#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/versions.env"

GOBIN="${GOBIN:-$(go env GOBIN)}"
GOBIN="${GOBIN:-$(go env GOPATH)/bin}"

if command -v mockery &>/dev/null; then
  INSTALLED=$(mockery --version --quiet --config="" 2>/dev/null || echo "")
  if [ "$INSTALLED" = "v${MOCKERY_VERSION}" ]; then
    exit 0
  fi
fi

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH="x86_64" ;;
  aarch64|arm64) ARCH="arm64" ;;
esac

echo "Installing mockery v${MOCKERY_VERSION}..."
curl -sSLo /tmp/mockery.tar.gz "https://github.com/vektra/mockery/releases/download/v${MOCKERY_VERSION}/mockery_${MOCKERY_VERSION}_${OS}_${ARCH}.tar.gz"
tar -xzf /tmp/mockery.tar.gz --directory="$GOBIN" mockery
rm /tmp/mockery.tar.gz
