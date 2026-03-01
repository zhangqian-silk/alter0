#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

export GOTOOLCHAIN="${GOTOOLCHAIN:-auto}"
export GOSUMDB="${GOSUMDB:-sum.golang.org}"

run() {
  local label="$1"
  shift
  echo "==> $label"
  "$@"
}

cd "$ROOT_DIR"

run "full test suite (cache disabled)" go test ./... -count=1
run "flaky hotspot stress pass" go test ./app/core/interaction/gateway ./app/core/interaction/http ./app/core/orchestrator/task ./app/core/runtime -count=20
run "windows regression compile check" env GOOS=windows GOARCH=amd64 go test ./... -run TestDoesNotExist -count=1 -exec=true

echo "test stability checks passed"
