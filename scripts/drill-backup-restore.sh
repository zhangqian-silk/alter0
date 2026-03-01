#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SOURCE_DB="$ROOT_DIR/output/db/alter0.db"
SOURCE_CONFIG="$ROOT_DIR/config/config.json"

if [[ ! -f "$SOURCE_DB" ]]; then
  echo "missing source db: $SOURCE_DB" >&2
  exit 1
fi

if [[ ! -f "$SOURCE_CONFIG" ]]; then
  echo "missing source config: $SOURCE_CONFIG" >&2
  exit 1
fi

WORK_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "$WORK_DIR"
}
trap cleanup EXIT

mkdir -p "$WORK_DIR/output/db" "$WORK_DIR/config"
cp "$SOURCE_DB" "$WORK_DIR/output/db/alter0.db"
cp "$SOURCE_CONFIG" "$WORK_DIR/config/config.json"

before_db="$(sha256sum "$WORK_DIR/output/db/alter0.db" | awk '{print $1}')"
before_cfg="$(sha256sum "$WORK_DIR/config/config.json" | awk '{print $1}')"

archive_path="$(ALTER0_ROOT="$WORK_DIR" "$ROOT_DIR/scripts/backup-local.sh")"

printf 'tamper-%s\n' "$(date -u +%s)" > "$WORK_DIR/output/db/alter0.db"
printf '{"broken":true}\n' > "$WORK_DIR/config/config.json"

ALTER0_ROOT="$WORK_DIR" "$ROOT_DIR/scripts/restore-local.sh" "$archive_path" >/dev/null

after_db="$(sha256sum "$WORK_DIR/output/db/alter0.db" | awk '{print $1}')"
after_cfg="$(sha256sum "$WORK_DIR/config/config.json" | awk '{print $1}')"

if [[ "$before_db" != "$after_db" ]]; then
  echo "rollback drill failed: db checksum mismatch" >&2
  exit 1
fi

if [[ "$before_cfg" != "$after_cfg" ]]; then
  echo "rollback drill failed: config checksum mismatch" >&2
  exit 1
fi

echo "backup/restore drill passed"
