#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

export GOTOOLCHAIN="${GOTOOLCHAIN:-auto}"
export GOSUMDB="${GOSUMDB:-sum.golang.org}"

cd "$ROOT_DIR"

echo "==> configs dependency boundary"
core_deps="$(go list -deps ./app/configs | grep '^alter0/app/core/' || true)"
if [[ -n "$core_deps" ]]; then
  echo "configs package must not depend on app/core packages"
  echo "$core_deps"
  exit 1
fi

echo "==> configs package tests"
go test ./app/configs -count=1

echo "config dependency boundary checks passed"
