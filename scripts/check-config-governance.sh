#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

export GOTOOLCHAIN="${GOTOOLCHAIN:-auto}"
export GOSUMDB="${GOSUMDB:-sum.golang.org}"

CONFIG_PATH="${CONFIG_PATH:-config/config.json}"
OUTPUT_PATH="${OUTPUT_PATH:-output/config/governance-latest.json}"

cd "$ROOT_DIR"

go run ./app/cmd/config-governance \
  --config "$CONFIG_PATH" \
  --output "$OUTPUT_PATH"

echo "config governance checks passed"
