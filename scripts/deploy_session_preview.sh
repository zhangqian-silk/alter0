#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: scripts/deploy_session_preview.sh <session_id> [repo_path]

Build the current frontend dist and register it on the shared alter0 runtime so
https://<session_short_hash>.alter0.cn serves this workspace's /chat frontend.

Environment:
  ALTER0_PREVIEW_BASE_URL       Shared runtime base URL. Default: https://alter0.cn
  ALTER0_WEB_LOGIN_PASSWORD     Login password. If unset, script will try /etc/alter0/alter0.env
  ALTER0_PREVIEW_SKIP_BUILD     Set to 1 to skip npm run build
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

SESSION_ID="${1:-}"
if [[ -z "${SESSION_ID}" ]]; then
  usage >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_PATH="${2:-$(cd "${SCRIPT_DIR}/.." && pwd)}"
REPO_PATH="$(cd "${REPO_PATH}" && pwd)"
BASE_URL="${ALTER0_PREVIEW_BASE_URL:-https://alter0.cn}"

if [[ ! -d "${REPO_PATH}/.git" && ! -f "${REPO_PATH}/.git" ]]; then
  echo "repository path is not a git workspace: ${REPO_PATH}" >&2
  exit 1
fi

if [[ "${ALTER0_PREVIEW_SKIP_BUILD:-0}" != "1" ]]; then
  (
    cd "${REPO_PATH}/internal/interfaces/web/frontend"
    npm run build
  )
fi

DIST_INDEX="${REPO_PATH}/internal/interfaces/web/static/dist/index.html"
if [[ ! -f "${DIST_INDEX}" ]]; then
  echo "missing built dist: ${DIST_INDEX}" >&2
  exit 1
fi

PASSWORD="${ALTER0_WEB_LOGIN_PASSWORD:-}"
if [[ -z "${PASSWORD}" && -r /etc/alter0/alter0.env ]]; then
  set -a
  # shellcheck disable=SC1091
  source /etc/alter0/alter0.env
  set +a
  PASSWORD="${ALTER0_WEB_LOGIN_PASSWORD:-}"
fi
if [[ -z "${PASSWORD}" ]]; then
  echo "ALTER0_WEB_LOGIN_PASSWORD is required" >&2
  exit 1
fi

COOKIE_JAR="$(mktemp /tmp/alter0-preview-cookie.XXXXXX)"
HEADERS_FILE="$(mktemp /tmp/alter0-preview-headers.XXXXXX)"
BODY_FILE="$(mktemp /tmp/alter0-preview-body.XXXXXX)"
cleanup() {
  rm -f "${COOKIE_JAR}" "${HEADERS_FILE}" "${BODY_FILE}"
}
trap cleanup EXIT

curl -sS \
  -c "${COOKIE_JAR}" \
  -b "${COOKIE_JAR}" \
  -X POST \
  --data-urlencode "password=${PASSWORD}" \
  --data-urlencode "next=/api/control/previews/${SESSION_ID}" \
  "${BASE_URL}/login" \
  -D "${HEADERS_FILE}" \
  -o /dev/null

curl -sS \
  -b "${COOKIE_JAR}" \
  -X PUT \
  -H "Content-Type: application/json" \
  -d "{\"repository_path\":\"${REPO_PATH//\"/\\\"}\"}" \
  "${BASE_URL}/api/control/previews/${SESSION_ID}" \
  -o "${BODY_FILE}"

cat "${BODY_FILE}"
echo
