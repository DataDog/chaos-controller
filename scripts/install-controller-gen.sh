#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/versions.env"

if command -v controller-gen &>/dev/null; then
  INSTALLED=$(controller-gen --version 2>/dev/null | awk '{ print $2 }' || echo "")
  if [ "$INSTALLED" = "$CONTROLLER_GEN_VERSION" ]; then
    exit 0
  fi
fi

echo "Installing controller-gen ${CONTROLLER_GEN_VERSION}..."
TMP_DIR=$(mktemp -d)
cd "$TMP_DIR"
go mod init tmp
CGO_ENABLED=0 go install "sigs.k8s.io/controller-tools/cmd/controller-gen@${CONTROLLER_GEN_VERSION}"
rm -rf "$TMP_DIR"
