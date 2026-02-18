#!/usr/bin/env bash
set -euo pipefail

GOBIN="${GOBIN:-$(go env GOBIN)}"
GOBIN="${GOBIN:-$(go env GOPATH)/bin}"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')

echo "Installing datadog-ci..."
curl -L --fail "https://github.com/DataDog/datadog-ci/releases/latest/download/datadog-ci_${OS}-x64" --output "$GOBIN/datadog-ci"
chmod u+x "$GOBIN/datadog-ci"
