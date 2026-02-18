#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/versions.env"

GOBIN="${GOBIN:-$(go env GOBIN)}"
GOBIN="${GOBIN:-$(go env GOPATH)/bin}"

if [ -x "$GOBIN/yamlfmt" ]; then
  exit 0
fi

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH="x86_64" ;;
  aarch64|arm64) ARCH="arm64" ;;
esac

echo "Installing yamlfmt v${YAMLFMT_VERSION}..."
curl -sSLo /tmp/yamlfmt.tar.gz "https://github.com/google/yamlfmt/releases/download/v${YAMLFMT_VERSION}/yamlfmt_${YAMLFMT_VERSION}_${OS}_${ARCH}.tar.gz"
tar -xzf /tmp/yamlfmt.tar.gz --directory="$GOBIN" yamlfmt
rm /tmp/yamlfmt.tar.gz
