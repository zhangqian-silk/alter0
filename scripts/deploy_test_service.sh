#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: scripts/deploy_test_service.sh <session_id> [service_name] [options]

Deploy or refresh a session-scoped test service on the shared alter0 gateway.

Options:
  --service-type <frontend_dist|http>  Deployment type. Default web deploy uses http full-stack preview.
  --repo-path <path>                   Git workspace path. Defaults to the current repo.
  --upstream-url <url>                 Existing HTTP upstream for service_type=http.
  --command <cmd>                      Start command for service_type=http. PORT is injected.
  --workdir <path>                     Working directory for --command.
  --port <number>                      Fixed local port for --command.
  --health-path <path>                 Health probe path after --command. Default: /
  --skip-build                         Skip the frontend build step when the deploy mode needs static assets.
  -h, --help                           Show this help.

Environment:
  ALTER0_GATEWAY_BASE_URL       Shared runtime base URL. Default: https://alter0.cn
  ALTER0_WEB_LOGIN_PASSWORD     Login password. If unset, the script will try /etc/alter0/alter0.env
  ALTER0_TEST_SERVICE_GOCACHE   Go build cache for default web full-stack preview. Default: /tmp/alter0-go-cache
  ALTER0_TEST_SERVICE_WEB_COMMAND
                                 Override default web full-stack start command.
EOF
}

default_web_start_command() {
  local go_cache="${ALTER0_TEST_SERVICE_GOCACHE:-/tmp/alter0-go-cache}"
  printf 'unset ALTER0_WEB_LOGIN_PASSWORD; export GOCACHE=%q; exec go run ./cmd/alter0 --internal-runtime-child --web-addr "127.0.0.1:${PORT}"' "${go_cache}"
}

base_host_from_url() {
  python3 - "$1" <<'PY'
import sys
from urllib.parse import urlparse

parsed = urlparse(sys.argv[1].strip())
print(parsed.hostname or "")
PY
}

merge_no_proxy_entries() {
  python3 - "$@" <<'PY'
import sys

seen = set()
merged = []
for raw in sys.argv[1:]:
    for item in raw.split(","):
        value = item.strip()
        if not value:
            continue
        lower = value.lower()
        if lower in seen:
            continue
        seen.add(lower)
        merged.append(value)
print(",".join(merged))
PY
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
shift
SERVICE_NAME="web"
if [[ $# -gt 0 && "${1#-}" == "${1}" ]]; then
  SERVICE_NAME="$1"
  shift
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_PATH="$(cd "${SCRIPT_DIR}/.." && pwd)"
SERVICE_TYPE=""
UPSTREAM_URL=""
START_COMMAND=""
WORKDIR=""
PORT=""
HEALTH_PATH="/"
SKIP_BUILD=0
DEFAULT_WEB_HTTP=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --service-type)
      SERVICE_TYPE="${2:-}"
      shift 2
      ;;
    --repo-path)
      REPO_PATH="${2:-}"
      shift 2
      ;;
    --upstream-url)
      UPSTREAM_URL="${2:-}"
      shift 2
      ;;
    --command)
      START_COMMAND="${2:-}"
      shift 2
      ;;
    --workdir)
      WORKDIR="${2:-}"
      shift 2
      ;;
    --port)
      PORT="${2:-}"
      shift 2
      ;;
    --health-path)
      HEALTH_PATH="${2:-}"
      shift 2
      ;;
    --skip-build)
      SKIP_BUILD=1
      shift 1
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

REPO_PATH="$(cd "${REPO_PATH}" && pwd)"
if [[ -z "${SERVICE_TYPE}" ]]; then
  if [[ -n "${UPSTREAM_URL}" || -n "${START_COMMAND}" ]]; then
    SERVICE_TYPE="http"
  elif [[ "${SERVICE_NAME}" == "web" ]]; then
    SERVICE_TYPE="http"
  else
    SERVICE_TYPE="frontend_dist"
  fi
fi

if [[ "${SERVICE_TYPE}" != "frontend_dist" && "${SERVICE_TYPE}" != "http" ]]; then
  echo "service type must be frontend_dist or http" >&2
  exit 1
fi

BASE_URL="${ALTER0_GATEWAY_BASE_URL:-https://alter0.cn}"
BASE_HOST="$(base_host_from_url "${BASE_URL}")"
CURL_NO_PROXY="$(merge_no_proxy_entries "${NO_PROXY:-${no_proxy:-}}" "${BASE_HOST}" ".${BASE_HOST}" "127.0.0.1" "localhost")"
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

RUNTIME_DIR=""
LOG_FILE=""
PID=""

if [[ "${SERVICE_TYPE}" == "http" && -z "${UPSTREAM_URL}" && -z "${START_COMMAND}" && "${SERVICE_NAME}" == "web" ]]; then
  DEFAULT_WEB_HTTP=1
  START_COMMAND="${ALTER0_TEST_SERVICE_WEB_COMMAND:-$(default_web_start_command)}"
  if [[ "${HEALTH_PATH}" == "/" ]]; then
    HEALTH_PATH="/readyz"
  fi
fi

if [[ "${SERVICE_TYPE}" == "frontend_dist" || "${DEFAULT_WEB_HTTP}" == "1" ]]; then
  if [[ ! -d "${REPO_PATH}/.git" && ! -f "${REPO_PATH}/.git" ]]; then
    echo "repository path is not a git workspace: ${REPO_PATH}" >&2
    exit 1
  fi
  if [[ "${SKIP_BUILD}" != "1" ]]; then
    (
      cd "${REPO_PATH}/internal/interfaces/web/frontend"
      npm run build
    )
  fi
  if [[ ! -f "${REPO_PATH}/internal/interfaces/web/static/dist/index.html" ]]; then
    echo "missing built dist: ${REPO_PATH}/internal/interfaces/web/static/dist/index.html" >&2
    exit 1
  fi
fi

if [[ "${SERVICE_TYPE}" == "http" ]]; then
  if [[ -n "${START_COMMAND}" ]]; then
    if [[ -z "${WORKDIR}" ]]; then
      WORKDIR="${REPO_PATH}"
    fi
    WORKDIR="$(cd "${WORKDIR}" && pwd)"
    if [[ -z "${PORT}" ]]; then
      PORT="$(python3 - <<'PY'
import socket
s = socket.socket()
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
PY
)"
    fi
    UPSTREAM_URL="http://127.0.0.1:${PORT}"
  elif [[ -z "${UPSTREAM_URL}" ]]; then
    echo "http deployments require --upstream-url or --command" >&2
    exit 1
  fi
fi

COOKIE_JAR="$(mktemp /tmp/alter0-service-cookie.XXXXXX)"
HEADERS_FILE="$(mktemp /tmp/alter0-service-headers.XXXXXX)"
BODY_FILE="$(mktemp /tmp/alter0-service-body.XXXXXX)"
cleanup() {
  rm -f "${COOKIE_JAR}" "${HEADERS_FILE}" "${BODY_FILE}"
}
trap cleanup EXIT

env NO_PROXY="${CURL_NO_PROXY}" no_proxy="${CURL_NO_PROXY}" HTTPS_PROXY= HTTP_PROXY= https_proxy= http_proxy= \
curl -sS \
  -c "${COOKIE_JAR}" \
  -b "${COOKIE_JAR}" \
  -X POST \
  --data-urlencode "password=${PASSWORD}" \
  --data-urlencode "next=/api/control/workspace-services/${SESSION_ID}/${SERVICE_NAME}" \
  "${BASE_URL}/login" \
  -D "${HEADERS_FILE}" \
  -o /dev/null

PAYLOAD="$(python3 - "${SERVICE_TYPE}" "${REPO_PATH}" "${UPSTREAM_URL}" "${START_COMMAND}" "${WORKDIR}" "${PORT}" "${HEALTH_PATH}" <<'PY'
import json
import sys

service_type = sys.argv[1]
repo_path = sys.argv[2]
upstream_url = sys.argv[3]
start_command = sys.argv[4]
workdir = sys.argv[5]
port = sys.argv[6]
health_path = sys.argv[7]
payload = {"service_type": service_type}
if service_type == "frontend_dist":
    payload["repository_path"] = repo_path
else:
    if start_command:
        payload["start_command"] = start_command
        payload["workdir"] = workdir
        payload["port"] = int(port)
        payload["health_path"] = health_path
    else:
        payload["upstream_url"] = upstream_url
print(json.dumps(payload))
PY
)"

HTTP_CODE="$(env NO_PROXY="${CURL_NO_PROXY}" no_proxy="${CURL_NO_PROXY}" HTTPS_PROXY= HTTP_PROXY= https_proxy= http_proxy= \
  curl -sS \
  -b "${COOKIE_JAR}" \
  -X PUT \
  -H "Content-Type: application/json" \
  -d "${PAYLOAD}" \
  "${BASE_URL}/api/control/workspace-services/${SESSION_ID}/${SERVICE_NAME}" \
  -o "${BODY_FILE}" \
  -w '%{http_code}')"

if [[ "${HTTP_CODE}" -lt 200 || "${HTTP_CODE}" -ge 300 ]]; then
  cat "${BODY_FILE}" >&2
  exit 1
fi

python3 - "${BODY_FILE}" "${RUNTIME_DIR}" "${LOG_FILE}" "${PID}" <<'PY'
import json
import pathlib
import sys

body_path = pathlib.Path(sys.argv[1])
runtime_dir = sys.argv[2]
log_path = sys.argv[3]
pid = sys.argv[4]
payload = json.loads(body_path.read_text())
if runtime_dir:
    payload["runtime_dir"] = runtime_dir
if log_path:
    payload["log_path"] = log_path
if pid:
    payload["pid"] = int(pid)
if "status" not in payload:
    payload["status"] = "deployed"
print(json.dumps(payload))
PY
