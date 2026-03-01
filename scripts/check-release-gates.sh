#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BASE_REF="${BASE_REF:-origin/master}"

export GOTOOLCHAIN="${GOTOOLCHAIN:-auto}"
export GOSUMDB="${GOSUMDB:-sum.golang.org}"

run() {
  local label="$1"
  shift
  echo "==> $label"
  "$@"
}

cd "$ROOT_DIR"

run "full regression" go test ./...
run "integration matrix" "$ROOT_DIR/scripts/run-integration-matrix.sh"
run "backup/restore rollback drill" "$ROOT_DIR/scripts/drill-backup-restore.sh"
run "deployment asset checks" "$ROOT_DIR/scripts/check-deploy-assets.sh"
run "documentation sync gate" "$ROOT_DIR/scripts/check-doc-sync.sh" "$BASE_REF"

echo "release gates passed"
