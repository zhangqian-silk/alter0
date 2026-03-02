#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

export GOTOOLCHAIN="${GOTOOLCHAIN:-auto}"
export GOSUMDB="${GOSUMDB:-sum.golang.org}"

CONFIG_PATH="${CONFIG_PATH:-config/config.json}"
OUTPUT_PATH="${OUTPUT_PATH:-output/config/validate-latest.json}"
ALLOW_MISSING_CONFIG="${ALLOW_MISSING_CONFIG:-true}"

cd "$ROOT_DIR"

go run ./app/cmd/config-validate-preflight \
  --config "$CONFIG_PATH" \
  --output "$OUTPUT_PATH" \
  --allow-missing-config="$ALLOW_MISSING_CONFIG"

echo "config preflight validation checks passed"
