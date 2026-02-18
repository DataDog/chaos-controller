#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/versions.env"

GOPATH="${GOPATH:-$(go env GOPATH)}"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  darwin) PROTOC_OS="osx" ;;
  linux)  PROTOC_OS="linux" ;;
  *)      echo "Unsupported OS: $OS"; exit 1 ;;
esac

PROTOC_ZIP="protoc-${PROTOC_VERSION}-${PROTOC_OS}-x86_64.zip"

echo "Installing protoc ${PROTOC_VERSION}..."
curl -sSLo "/tmp/${PROTOC_ZIP}" "https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/${PROTOC_ZIP}"
unzip -o "/tmp/${PROTOC_ZIP}" -d "$GOPATH" bin/protoc
unzip -o "/tmp/${PROTOC_ZIP}" -d "$GOPATH" 'include/*'
rm -f "/tmp/${PROTOC_ZIP}"

echo "Installing protoc-gen-go ${PROTOC_GEN_GO_VERSION}..."
go install "google.golang.org/protobuf/cmd/protoc-gen-go@${PROTOC_GEN_GO_VERSION}"

echo "Installing protoc-gen-go-grpc ${PROTOC_GEN_GO_GRPC_VERSION}..."
go install "google.golang.org/grpc/cmd/protoc-gen-go-grpc@${PROTOC_GEN_GO_GRPC_VERSION}"
