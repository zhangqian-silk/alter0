#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

export GOTOOLCHAIN="${GOTOOLCHAIN:-auto}"
export GOSUMDB="${GOSUMDB:-sum.golang.org}"

THRESHOLD_HISTORY_PATH="${THRESHOLD_HISTORY_PATH:-output/cost/threshold-history-latest.json}"
CONFIG_PATH="${CONFIG_PATH:-config/config.json}"
OUTPUT_PATH="${THRESHOLD_RECONCILE_OUTPUT_PATH:-output/cost/threshold-reconcile-latest.json}"
MAX_SHARE_STEP="${THRESHOLD_RECONCILE_MAX_SHARE_STEP:-0.10}"
MAX_RATIO_STEP="${THRESHOLD_RECONCILE_MAX_RATIO_STEP:-2.0}"
APPLY_MODE="${THRESHOLD_RECONCILE_APPLY:-false}"

cd "$ROOT_DIR"

args=(
  --history "$THRESHOLD_HISTORY_PATH"
  --config "$CONFIG_PATH"
  --output "$OUTPUT_PATH"
  --max-share-step "$MAX_SHARE_STEP"
  --max-ratio-step "$MAX_RATIO_STEP"
)

if [ "$APPLY_MODE" = "true" ]; then
  args+=(--apply)
fi

go run ./app/cmd/cost-threshold-reconcile "${args[@]}"

echo "cost threshold reconcile checks passed"
