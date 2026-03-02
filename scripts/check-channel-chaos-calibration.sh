#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

export GOTOOLCHAIN="${GOTOOLCHAIN:-auto}"
export GOSUMDB="${GOSUMDB:-sum.golang.org}"

CANDIDATE_ARCHIVE_ROOT="${CANDIDATE_ARCHIVE_ROOT:-output/channel-chaos/candidates}"
MATRIX_PATH="${MATRIX_PATH:-config/channel-chaos-matrix.json}"
THRESHOLD_HISTORY_ROOT="${THRESHOLD_HISTORY_ROOT:-output/cost/threshold-history}"
THRESHOLD_RECONCILE_ROOT="${THRESHOLD_RECONCILE_ROOT:-output/cost/threshold-reconcile}"
WINDOW="${WINDOW:-336h}"
OUTPUT_PATH="${OUTPUT_PATH:-output/channel-chaos/calibration-latest.json}"
ARCHIVE_ROOT="${ARCHIVE_ROOT:-output/channel-chaos/calibration}"

cd "$ROOT_DIR"

go run ./app/cmd/channel-chaos-calibration \
  --candidate-archive-root "$CANDIDATE_ARCHIVE_ROOT" \
  --matrix "$MATRIX_PATH" \
  --threshold-history-root "$THRESHOLD_HISTORY_ROOT" \
  --threshold-reconcile-root "$THRESHOLD_RECONCILE_ROOT" \
  --window "$WINDOW" \
  --output "$OUTPUT_PATH" \
  --archive-root "$ARCHIVE_ROOT"

echo "channel chaos calibration checks passed"
