#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/versions.env"

GOBIN="${GOBIN:-$(go env GOBIN)}"
GOBIN="${GOBIN:-$(go env GOPATH)/bin}"

if command -v golangci-lint &>/dev/null; then
  INSTALLED=$(golangci-lint --version 2>/dev/null | sed -E 's/.*version ([^ ]+).*/\1/' || echo "")
  if [ "$INSTALLED" = "$GOLANGCI_LINT_VERSION" ]; then
    exit 0
  fi
fi

echo "Installing golangci-lint v${GOLANGCI_LINT_VERSION}..."
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "$GOBIN" "v${GOLANGCI_LINT_VERSION}"
