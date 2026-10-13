#!/usr/bin/env bash
# Package chaosli binaries into one Dogbrew tarball per platform and print each
# tarball path on its own line (the publish job's TARBALLS).
#
# Usage: scripts/dogbrew/package.sh <version> [dist-dir]
set -euo pipefail

version="${1:?usage: package.sh <version> [dist-dir]}"

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/../.." && pwd)"
dist_dir="${2:-${repo_root}/dist/dogbrew}"

# Dogbrew requires SemVer 2.0 with no leading "v".
semver='^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$'
[[ "${version}" =~ ${semver} ]] || { echo "error: version is not SemVer: ${version}" >&2; exit 1; }

work_dir="$(mktemp -d)"
trap 'rm -rf "${work_dir}"' EXIT

for platform in darwin-arm64 linux-amd64 linux-arm64; do
  binary="chaosli-${platform}"
  platform_dir="${work_dir}/${platform}"
  tarball="${dist_dir}/${binary}.tar.gz"

  [[ -s "${dist_dir}/${binary}" ]] || { echo "error: missing or empty binary: ${dist_dir}/${binary}" >&2; exit 1; }
  mkdir -p "${platform_dir}"
  sed "s/^version:.*/version: ${version}/" "${repo_root}/cli/chaosli/cli.yaml" > "${platform_dir}/cli.yaml"
  cp "${dist_dir}/${binary}" "${platform_dir}/${binary}"
  chmod +x "${platform_dir}/${binary}"
  tar -czf "${tarball}" -C "${platform_dir}" cli.yaml "${binary}"
  printf '%s\n' "${tarball}"
done
