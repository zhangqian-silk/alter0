#!/usr/bin/env bash
set -euo pipefail

BASE_REF="${1:-origin/master}"

if ! git rev-parse --verify "$BASE_REF" >/dev/null 2>&1; then
  echo "base ref not found: $BASE_REF" >&2
  exit 2
fi

changed_files="$(git diff --name-only "$BASE_REF"...HEAD)"

if [ -z "$changed_files" ]; then
  echo "doc sync check skipped: no changes between $BASE_REF and HEAD"
  exit 0
fi

has_features=0
has_readme=0
has_arch=0

while IFS= read -r file; do
  case "$file" in
    docs/features.md) has_features=1 ;;
    README.md) has_readme=1 ;;
    ARCHITECTURE.md) has_arch=1 ;;
  esac
done <<< "$changed_files"

if [ "$has_features" -eq 0 ]; then
  echo "doc sync check passed: docs/features.md not changed"
  exit 0
fi

missing=()
if [ "$has_readme" -eq 0 ]; then
  missing+=("README.md")
fi
if [ "$has_arch" -eq 0 ]; then
  missing+=("ARCHITECTURE.md")
fi

if [ "${#missing[@]}" -gt 0 ]; then
  echo "doc sync check failed: docs/features.md changed but missing updates in: ${missing[*]}" >&2
  exit 1
fi

echo "doc sync check passed: docs/features.md, README.md and ARCHITECTURE.md are updated together"
