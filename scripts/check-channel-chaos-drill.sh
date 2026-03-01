#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

export GOTOOLCHAIN="${GOTOOLCHAIN:-auto}"
export GOSUMDB="${GOSUMDB:-sum.golang.org}"

MATRIX_PATH="${MATRIX_PATH:-config/channel-chaos-matrix.json}"
OUTPUT_PATH="${OUTPUT_PATH:-output/channel-chaos/drill-latest.json}"

cd "$ROOT_DIR"

go run ./app/cmd/channel-chaos-drill \
  --matrix "$MATRIX_PATH" \
  --output "$OUTPUT_PATH"

echo "channel chaos drill checks passed"
