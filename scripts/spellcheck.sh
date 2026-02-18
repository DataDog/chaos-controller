#!/usr/bin/env bash
set -euo pipefail

ensure_deps() {
  if ! command -v npm &>/dev/null; then
    echo "please install npm or run 'make spellcheck-docker' for a slow but platform-agnostic run"
    exit 1
  fi
  if ! command -v mdspell &>/dev/null; then
    echo "installing mdspell through npm -g... (might require sudo run)"
    npm -g i markdown-spellcheck
  fi
}

find_markdown() {
  find . -name vendor -prune -o -name '*.md' -print
}

cmd="${1:-}"
shift || true

case "$cmd" in
  check)
    ensure_deps
    mdspell --en-us --ignore-acronyms --ignore-numbers $(find_markdown)
    ;;
  report)
    ensure_deps
    mdspell --en-us --ignore-acronyms --ignore-numbers --report $(find_markdown)
    ;;
  docker)
    docker run --rm -ti -v "$(pwd)":/workdir tmaier/markdown-spellcheck:latest \
      --ignore-numbers --ignore-acronyms --en-us $(find_markdown)
    ;;
  format-spelling)
    sort < .spelling | uniq | grep -v '^-' | tee .spelling.tmp > /dev/null && mv .spelling.tmp .spelling
    ;;
  *)
    echo "Usage: $0 {check|report|docker|format-spelling}"
    exit 1
    ;;
esac
