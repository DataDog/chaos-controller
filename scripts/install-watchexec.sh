#!/usr/bin/env bash
set -euo pipefail

GOBIN="${GOBIN:-$(go env GOBIN)}"
GOBIN="${GOBIN:-$(go env GOPATH)/bin}"

if [ -f "${GOBIN}/watchexec" ] || command -v watchexec &>/dev/null; then
  exit 0
fi

echo "Installing watchexec..."
brew install watchexec
