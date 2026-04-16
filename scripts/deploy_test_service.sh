#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: scripts/deploy_test_service.sh <session_id> [service_name] [options]

Deploy or refresh a session-scoped test service on the shared alter0 gateway.

Options:
  --service-type <frontend_dist|http>  Deployment type. Auto-detected by flags when omitted.
  --repo-path <path>                   Git workspace path. Defaults to the current repo.
  --upstream-url <url>                 Existing HTTP upstream for service_type=http.
  --command <cmd>                      Start command for service_type=http. PORT is injected.
  --workdir <path>                     Working directory for --command.
  --port <number>                      Fixed local port for --command.
  --health-path <path>                 Health probe path after --command. Default: /
  --skip-build                         Skip npm run build for frontend_dist.
  -h, --help                           Show this help.

Environment:
  ALTER0_GATEWAY_BASE_URL       Shared runtime base URL. Default: https://alter0.cn
  ALTER0_WEB_LOGIN_PASSWORD     Login password. If unset, the script will try /etc/alter0/alter0.env
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
  else
    SERVICE_TYPE="frontend_dist"
  fi
fi

if [[ "${SERVICE_TYPE}" != "frontend_dist" && "${SERVICE_TYPE}" != "http" ]]; then
  echo "service type must be frontend_dist or http" >&2
  exit 1
fi

BASE_URL="${ALTER0_GATEWAY_BASE_URL:-https://alter0.cn}"
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

if [[ "${SERVICE_TYPE}" == "frontend_dist" ]]; then
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
else
  if [[ -n "${START_COMMAND}" ]]; then
    if [[ -z "${WORKDIR}" ]]; then
      WORKDIR="${REPO_PATH}"
    fi
    WORKDIR="$(cd "${WORKDIR}" && pwd)"
    RUNTIME_DIR="${WORKDIR}/.alter0/test-services/${SESSION_ID}/${SERVICE_NAME}"
    mkdir -p "${RUNTIME_DIR}"
    LOG_FILE="${RUNTIME_DIR}/service.log"
    PID_FILE="${RUNTIME_DIR}/service.pid"
    if [[ -r "${PID_FILE}" ]]; then
      PREVIOUS_PID="$(cat "${PID_FILE}")"
      if [[ -n "${PREVIOUS_PID}" ]] && kill -0 "${PREVIOUS_PID}" 2>/dev/null; then
        kill "${PREVIOUS_PID}" 2>/dev/null || true
      fi
    fi
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
    : > "${LOG_FILE}"
    (
      cd "${WORKDIR}"
      nohup env PORT="${PORT}" ALTER0_SERVICE_PORT="${PORT}" bash -lc "${START_COMMAND}" >>"${LOG_FILE}" 2>&1 &
      echo $! > "${PID_FILE}"
    )
    PID="$(cat "${PID_FILE}")"
    UPSTREAM_URL="http://127.0.0.1:${PORT}"
    HEALTH_URL="${UPSTREAM_URL}${HEALTH_PATH}"
    HEALTHY=0
    for _ in $(seq 1 40); do
      if curl -fsS --max-time 2 "${HEALTH_URL}" >/dev/null 2>&1; then
        HEALTHY=1
        break
      fi
      sleep 0.5
    done
    if [[ "${HEALTHY}" != "1" ]]; then
      if [[ -n "${PID}" ]] && kill -0 "${PID}" 2>/dev/null; then
        kill "${PID}" 2>/dev/null || true
      fi
      echo "service failed health probe: ${HEALTH_URL}" >&2
      exit 1
    fi
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

curl -sS \
  -c "${COOKIE_JAR}" \
  -b "${COOKIE_JAR}" \
  -X POST \
  --data-urlencode "password=${PASSWORD}" \
  --data-urlencode "next=/api/control/workspace-services/${SESSION_ID}/${SERVICE_NAME}" \
  "${BASE_URL}/login" \
  -D "${HEADERS_FILE}" \
  -o /dev/null

PAYLOAD="$(python3 - "${SERVICE_TYPE}" "${REPO_PATH}" "${UPSTREAM_URL}" <<'PY'
import json
import sys

service_type = sys.argv[1]
repo_path = sys.argv[2]
upstream_url = sys.argv[3]
payload = {"service_type": service_type}
if service_type == "frontend_dist":
    payload["repository_path"] = repo_path
else:
    payload["upstream_url"] = upstream_url
print(json.dumps(payload))
PY
)"

HTTP_CODE="$(curl -sS \
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
