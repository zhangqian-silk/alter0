#!/usr/bin/env bash
set -euo pipefail

SCRIPT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ROOT_DIR="${ALTER0_ROOT:-$SCRIPT_ROOT}"
DB_FILE="$ROOT_DIR/output/db/alter0.db"
CONFIG_FILE="$ROOT_DIR/config/config.json"
BACKUP_DIR="$ROOT_DIR/output/backups"
TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"
ARCHIVE_PATH="$BACKUP_DIR/alter0-backup-$TIMESTAMP.tar.gz"

if [[ ! -f "$DB_FILE" ]]; then
  echo "error: database file not found: $DB_FILE" >&2
  exit 1
fi

if [[ ! -f "$CONFIG_FILE" ]]; then
  echo "error: config file not found: $CONFIG_FILE" >&2
  exit 1
fi

mkdir -p "$BACKUP_DIR"

TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

mkdir -p "$TMP_DIR/output/db" "$TMP_DIR/config"
cp "$DB_FILE" "$TMP_DIR/output/db/alter0.db"
cp "$CONFIG_FILE" "$TMP_DIR/config/config.json"

cat >"$TMP_DIR/manifest.txt" <<EOF
created_at_utc=$TIMESTAMP
db_path=output/db/alter0.db
config_path=config/config.json
EOF

tar -C "$TMP_DIR" -czf "$ARCHIVE_PATH" .

echo "$ARCHIVE_PATH"
