#!/usr/bin/env bash
set -euo pipefail

umask 027

ENV_FILE="${ALTER0_ENV_FILE:-/etc/alter0/alter0.env}"
if [[ -f "$ENV_FILE" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "$ENV_FILE"
  set +a
fi

UNIT="alter0.service"
while [[ $# -gt 0 ]]; do
  case "$1" in
    --unit)
      UNIT="${2:-}"
      shift 2
      ;;
    *)
      echo "unknown arg: $1" >&2
      exit 2
      ;;
  esac
done

if [[ -z "$UNIT" ]]; then
  echo "unit is required" >&2
  exit 2
fi

case "$(uname -s)" in
  Linux) ;;
  *)
    echo "relaunch_service.sh only supports Linux/systemd" >&2
    exit 3
    ;;
esac

SYSTEMCTL="$(command -v systemctl || true)"
if [[ -z "$SYSTEMCTL" ]]; then
  echo "systemctl not found" >&2
  exit 4
fi

if [[ "${ALTER0_RUNTIME_MANAGER:-}" != "systemd" ]]; then
  echo "ALTER0_RUNTIME_MANAGER is not systemd" >&2
  exit 5
fi

ALLOWED_UNIT="${ALTER0_SYSTEMD_UNIT:-alter0.service}"
if [[ "$UNIT" != "$ALLOWED_UNIT" ]]; then
  echo "refusing to restart unmanaged unit: $UNIT" >&2
  exit 6
fi

if ! "$SYSTEMCTL" cat "$UNIT" >/dev/null 2>&1; then
  echo "systemd unit not registered: $UNIT" >&2
  exit 7
fi

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOCK_DIR="${ALTER0_DEPLOY_LOCK_DIR:-/var/lib/alter0}"
mkdir -p "$LOCK_DIR"
LOCK_FILE="$LOCK_DIR/relaunch.lock"
exec 9>"$LOCK_FILE"
if ! flock -n 9; then
  echo "relaunch already in progress" >&2
  exit 8
fi

BRANCH="${ALTER0_RELAUNCH_BRANCH:-master}"
REMOTE="${ALTER0_RELAUNCH_REMOTE:-origin}"
BUILD_OUTPUT="${ALTER0_BUILD_OUTPUT:-$REPO_DIR/bin/alter0}"
mkdir -p "$(dirname "$BUILD_OUTPUT")"

cd "$REPO_DIR"
if command -v git >/dev/null 2>&1; then
  fetch_ok=0
  for delay in 0 2 4; do
    if [[ "$delay" -gt 0 ]]; then
      sleep "$delay"
    fi
    if git -c http.version=HTTP/1.1 fetch "$REMOTE"; then
      fetch_ok=1
      break
    fi
  done
  if [[ "$fetch_ok" -ne 1 ]]; then
    echo "git fetch failed after retries" >&2
    exit 10
  fi
  git checkout "$BRANCH"
  git reset --hard "$REMOTE/$BRANCH"
fi

env GOSUMDB="${GOSUMDB:-sum.golang.org}" GOTOOLCHAIN="${GOTOOLCHAIN:-auto}" go build -o "$BUILD_OUTPUT" ./cmd/alter0

if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  if command -v sudo >/dev/null 2>&1; then
    exec sudo "$0" --unit "$UNIT"
  fi
  echo "root or passwordless sudo is required to restart $UNIT" >&2
  exit 9
fi

"$SYSTEMCTL" restart "$UNIT"
