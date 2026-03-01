#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

export GOTOOLCHAIN="${GOTOOLCHAIN:-auto}"
export GOSUMDB="${GOSUMDB:-sum.golang.org}"

cd "$ROOT_DIR"

echo "==> service dependency boundary"
forbidden_deps="$(go list -f '{{.ImportPath}} {{join .Imports " "}}' ./app/core/interaction/... ./app/core/runtime/... | grep 'alter0/app/core/orchestrator/schedule' || true)"
if [[ -n "$forbidden_deps" ]]; then
  echo "interaction/runtime must not depend on app/core/orchestrator/schedule directly"
  echo "$forbidden_deps"
  exit 1
fi

echo "==> service boundary package tests"
go test ./app/core/interaction/http ./app/core/runtime ./app/core/service/schedule -count=1

echo "service boundary checks passed"
