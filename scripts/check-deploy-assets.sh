#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

assert_contains() {
  local path="$1"
  local needle="$2"
  if ! grep -Fq "$needle" "$path"; then
    echo "missing '$needle' in $path" >&2
    exit 1
  fi
}

assert_contains "$ROOT_DIR/Dockerfile" "FROM golang:1.25.7 AS builder"
assert_contains "$ROOT_DIR/Dockerfile" "ENTRYPOINT [\"/usr/local/bin/alter0\"]"
assert_contains "$ROOT_DIR/deploy/systemd/alter0.service" "Restart=always"
assert_contains "$ROOT_DIR/deploy/systemd/alter0.service" "WorkingDirectory=/opt/alter0"
assert_contains "$ROOT_DIR/deploy/systemd/alter0.service" "ExecStart=/usr/local/bin/alter0"

echo "deployment assets look good"
