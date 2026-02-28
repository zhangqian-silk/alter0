#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <backup-archive.tar.gz>" >&2
  exit 1
fi

ARCHIVE_PATH="$1"
if [[ ! -f "$ARCHIVE_PATH" ]]; then
  echo "error: archive not found: $ARCHIVE_PATH" >&2
  exit 1
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TARGET_DB="$ROOT_DIR/output/db/alter0.db"
TARGET_CONFIG="$ROOT_DIR/config/config.json"

TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

tar -C "$TMP_DIR" -xzf "$ARCHIVE_PATH"

RESTORE_DB="$TMP_DIR/output/db/alter0.db"
RESTORE_CONFIG="$TMP_DIR/config/config.json"

if [[ ! -f "$RESTORE_DB" ]]; then
  echo "error: invalid archive, missing output/db/alter0.db" >&2
  exit 1
fi

if [[ ! -f "$RESTORE_CONFIG" ]]; then
  echo "error: invalid archive, missing config/config.json" >&2
  exit 1
fi

mkdir -p "$(dirname "$TARGET_DB")" "$(dirname "$TARGET_CONFIG")"
cp "$RESTORE_DB" "$TARGET_DB"
cp "$RESTORE_CONFIG" "$TARGET_CONFIG"

echo "restored db and config from $ARCHIVE_PATH"
