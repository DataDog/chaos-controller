#!/usr/bin/env bash
# Build chaosli for the Dogbrew platforms (darwin-arm64, linux-amd64, linux-arm64).
#
# Usage: scripts/dogbrew/build.sh <version> [out-dir]
set -euo pipefail

version="${1:?usage: build.sh <version> [out-dir]}"

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/../.." && pwd)"
out_dir="${2:-${repo_root}/dist/dogbrew}"

cd "${repo_root}"
mkdir -p "${out_dir}"

ldflags="-s -w -X github.com/DataDog/chaos-controller/cli/chaosli/cmd.Version=${version}"

# Dogbrew does not support Intel Mac (darwin-amd64).
for platform in darwin/arm64 linux/amd64 linux/arm64; do
  os="${platform%/*}"
  arch="${platform#*/}"
  GOOS="${os}" GOARCH="${arch}" CGO_ENABLED=0 \
    go build -trimpath -buildvcs=false -ldflags "${ldflags}" \
    -o "${out_dir}/chaosli-${os}-${arch}" \
    ./cli/chaosli
done
