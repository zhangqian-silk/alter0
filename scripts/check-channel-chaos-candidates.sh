#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

export GOTOOLCHAIN="${GOTOOLCHAIN:-auto}"
export GOSUMDB="${GOSUMDB:-sum.golang.org}"

TRACE_BASE="${TRACE_BASE:-output/trace}"
WINDOW="${WINDOW:-24h}"
MIN_ERRORS="${MIN_ERRORS:-2}"
MAX_CANDIDATES="${MAX_CANDIDATES:-5}"
MAX_EVENTS="${MAX_EVENTS:-8}"
OUTPUT_PATH="${OUTPUT_PATH:-output/channel-chaos/candidates-latest.json}"

cd "$ROOT_DIR"

go run ./app/cmd/channel-chaos-candidates \
  --trace-base "$TRACE_BASE" \
  --window "$WINDOW" \
  --min-errors "$MIN_ERRORS" \
  --max-candidates "$MAX_CANDIDATES" \
  --max-events "$MAX_EVENTS" \
  --output "$OUTPUT_PATH"

echo "channel chaos candidate checks passed"
