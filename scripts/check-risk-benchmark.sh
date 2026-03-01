#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

export GOTOOLCHAIN="${GOTOOLCHAIN:-auto}"
export GOSUMDB="${GOSUMDB:-sum.golang.org}"

WATCHLIST_PATH="${WATCHLIST_PATH:-config/risk-watchlist.json}"
RUNBOOK_PATH="${RUNBOOK_PATH:-docs/runbooks/risk-drift-triage.md}"
SCENARIO_MATRIX_PATH="${SCENARIO_MATRIX_PATH:-config/scenario-benchmark-matrix.json}"
COMPETITOR_TRACKING_PATH="${COMPETITOR_TRACKING_PATH:-config/competitor-tracking.json}"
THRESHOLD_HISTORY_PATH="${THRESHOLD_HISTORY_PATH:-output/cost/threshold-history-latest.json}"
OUTPUT_PATH="${OUTPUT_PATH:-output/risk/benchmark-latest.json}"
MAX_STALE_HOURS="${MAX_STALE_HOURS:-96}"
SCENARIO_MAX_STALE_DAYS="${SCENARIO_MAX_STALE_DAYS:-45}"
COMPETITOR_MAX_STALE_DAYS="${COMPETITOR_MAX_STALE_DAYS:-31}"
THRESHOLD_MAX_STALE_DAYS="${THRESHOLD_MAX_STALE_DAYS:-8}"
ALLOW_OVERDUE="${ALLOW_OVERDUE:-false}"
ENABLE_SCENARIO_MATRIX="${ENABLE_SCENARIO_MATRIX:-true}"
ENABLE_COMPETITOR_TRACKING="${ENABLE_COMPETITOR_TRACKING:-true}"
ENABLE_THRESHOLD_HISTORY="${ENABLE_THRESHOLD_HISTORY:-true}"

cd "$ROOT_DIR"

if [ "$ENABLE_THRESHOLD_HISTORY" = "true" ]; then
  THRESHOLD_HISTORY_OUTPUT_PATH="$THRESHOLD_HISTORY_PATH" \
    "$ROOT_DIR/scripts/check-cost-threshold-history.sh"
fi

args=(
  --watchlist "$WATCHLIST_PATH"
  --runbook "$RUNBOOK_PATH"
  --scenario-matrix "$SCENARIO_MATRIX_PATH"
  --competitor-tracking "$COMPETITOR_TRACKING_PATH"
  --threshold-history "$THRESHOLD_HISTORY_PATH"
  --output "$OUTPUT_PATH"
  --max-stale-hours "$MAX_STALE_HOURS"
  --scenario-max-stale-days "$SCENARIO_MAX_STALE_DAYS"
  --competitor-max-stale-days "$COMPETITOR_MAX_STALE_DAYS"
  --threshold-max-stale-days "$THRESHOLD_MAX_STALE_DAYS"
)

if [ "$ALLOW_OVERDUE" = "true" ]; then
  args+=(--allow-overdue)
fi
if [ "$ENABLE_SCENARIO_MATRIX" != "true" ]; then
  args+=(--skip-scenario)
fi
if [ "$ENABLE_COMPETITOR_TRACKING" != "true" ]; then
  args+=(--skip-competitor)
fi
if [ "$ENABLE_THRESHOLD_HISTORY" != "true" ]; then
  args+=(--skip-threshold-history)
fi

go run ./app/cmd/risk-benchmark "${args[@]}"

echo "risk benchmark checks passed"
