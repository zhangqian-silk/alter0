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

run "config dependency boundary gate" "$ROOT_DIR/scripts/check-config-boundary.sh"
run "service dependency boundary gate" "$ROOT_DIR/scripts/check-service-boundary.sh"
run "config parameter governance gate" "$ROOT_DIR/scripts/check-config-governance.sh"
run "test stability gate" "$ROOT_DIR/scripts/check-test-stability.sh"
run "integration matrix" "$ROOT_DIR/scripts/run-integration-matrix.sh"
run "risk benchmark gate" "$ROOT_DIR/scripts/check-risk-benchmark.sh"
run "channel chaos drill gate" "$ROOT_DIR/scripts/check-channel-chaos-drill.sh"
run "channel chaos candidate extraction" "$ROOT_DIR/scripts/check-channel-chaos-candidates.sh"
run "channel chaos calibration gate" "$ROOT_DIR/scripts/check-channel-chaos-calibration.sh"
run "backup/restore rollback drill" "$ROOT_DIR/scripts/drill-backup-restore.sh"
run "deployment asset checks" "$ROOT_DIR/scripts/check-deploy-assets.sh"
run "documentation sync gate" "$ROOT_DIR/scripts/check-doc-sync.sh" "$BASE_REF"

echo "release gates passed"
