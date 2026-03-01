#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

export GOTOOLCHAIN="${GOTOOLCHAIN:-auto}"
export GOSUMDB="${GOSUMDB:-sum.golang.org}"

WATCHLIST_PATH="${WATCHLIST_PATH:-config/risk-watchlist.json}"
RUNBOOK_PATH="${RUNBOOK_PATH:-docs/runbooks/risk-drift-triage.md}"
OUTPUT_PATH="${OUTPUT_PATH:-output/risk/benchmark-latest.json}"
MAX_STALE_HOURS="${MAX_STALE_HOURS:-96}"
ALLOW_OVERDUE="${ALLOW_OVERDUE:-false}"

cd "$ROOT_DIR"

args=(
  --watchlist "$WATCHLIST_PATH"
  --runbook "$RUNBOOK_PATH"
  --output "$OUTPUT_PATH"
  --max-stale-hours "$MAX_STALE_HOURS"
)

if [ "$ALLOW_OVERDUE" = "true" ]; then
  args+=(--allow-overdue)
fi

go run ./app/cmd/risk-benchmark "${args[@]}"

echo "risk benchmark checks passed"
