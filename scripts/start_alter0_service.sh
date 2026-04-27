#!/usr/bin/env bash
set -euo pipefail

umask 027

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

ENV_FILE="${ALTER0_ENV_FILE:-/etc/alter0/alter0.env}"
if [[ -f "${ENV_FILE}" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "${ENV_FILE}"
  set +a
fi

WEB_ADDR="${ALTER0_WEB_ADDR:-127.0.0.1:18088}"
WEB_BIND_LOCALHOST_ONLY="${ALTER0_WEB_BIND_LOCALHOST_ONLY:-true}"
WEB_LOGIN_PASSWORD="${ALTER0_WEB_LOGIN_PASSWORD:-}"
RUN_AS="${ALTER0_RUN_AS:-alter0}"

RUNTIME_ROOT="${ALTER0_RUNTIME_ROOT:-/var/lib/alter0}"
STORAGE_DIR="${ALTER0_STORAGE_DIR:-${RUNTIME_ROOT}/storage}"
HOME_DIR="${ALTER0_HOME:-}"
if [[ -z "${HOME_DIR}" ]]; then
  case "${HOME:-}" in
    "${RUNTIME_ROOT}"|"${RUNTIME_ROOT}/"*)
      HOME_DIR="${HOME}"
      ;;
  esac
fi
if [[ "${HOME_DIR}" == "${RUNTIME_ROOT}/codex-home" ]]; then
  HOME_DIR="${RUNTIME_ROOT}"
fi
if [[ -z "${HOME_DIR}" ]]; then
  HOME_DIR="${RUNTIME_ROOT}"
fi
LOG_FILE="${ALTER0_LOG_FILE:-/var/log/alter0/alter0.log}"
LOCK_FILE="${RUNTIME_ROOT}/run.lock"
ALTER0_RUNTIME_MANAGER="${ALTER0_RUNTIME_MANAGER:-systemd}"
ALTER0_SYSTEMD_UNIT="${ALTER0_SYSTEMD_UNIT:-alter0.service}"
BUILD_OUTPUT="${ALTER0_BUILD_OUTPUT:-${REPO_DIR}/bin/alter0}"
CODEX_COMMAND="${ALTER0_CODEX_COMMAND:-}"
if [[ -z "${CODEX_COMMAND}" && -x /usr/local/bin/codex ]]; then
  CODEX_COMMAND="/usr/local/bin/codex"
fi

mkdir -p "${RUNTIME_ROOT}" "${STORAGE_DIR}" "${HOME_DIR}" "$(dirname "${LOG_FILE}")" "${REPO_DIR}/.alter0" "$(dirname "${BUILD_OUTPUT}")"
chmod 750 "${RUNTIME_ROOT}" "${HOME_DIR}" || true
chmod 700 "${STORAGE_DIR}" "${REPO_DIR}/.alter0" || true
touch "${LOG_FILE}"
chmod 640 "${LOG_FILE}" || true
export HOME="${HOME_DIR}"

if [[ "$(id -un)" == "root" ]]; then
  echo "start_alter0_service.sh should run as ${RUN_AS}, not root" >&2
  echo "configure systemd User=${RUN_AS} Group=${RUN_AS}" >&2
  exit 1
fi

exec 9>"${LOCK_FILE}"
flock -n 9 || {
  echo "alter0 service is already running"
  exit 1
}

if [[ -z "${WEB_LOGIN_PASSWORD}" ]]; then
  echo "ALTER0_WEB_LOGIN_PASSWORD is required; anonymous web access is disabled." >&2
  exit 1
fi

env GOSUMDB="${GOSUMDB:-sum.golang.org}" GOTOOLCHAIN="${GOTOOLCHAIN:-auto}" go build -o "${BUILD_OUTPUT}" ./cmd/alter0

CMD="env \
GOSUMDB='${GOSUMDB:-sum.golang.org}' \
GOTOOLCHAIN='${GOTOOLCHAIN:-auto}' \
HOME='${HOME_DIR}' \
ALTER0_RUNTIME_MANAGER='${ALTER0_RUNTIME_MANAGER}' \
ALTER0_SYSTEMD_UNIT='${ALTER0_SYSTEMD_UNIT}' \
ALTER0_BUILD_OUTPUT='${BUILD_OUTPUT}' \
ALTER0_CODEX_COMMAND='${CODEX_COMMAND}' \
'${BUILD_OUTPUT}' \
-codex-command '${CODEX_COMMAND}' \
-web-addr '${WEB_ADDR}' \
-web-bind-localhost-only='${WEB_BIND_LOCALHOST_ONLY}' \
-web-login-password '${WEB_LOGIN_PASSWORD}' \
-daily-memory-dir '${STORAGE_DIR}/memory' \
-long-term-memory-path '${STORAGE_DIR}/memory/long-term/MEMORY.md'"

cd "${REPO_DIR}"
exec bash -lc "${CMD} >>'${LOG_FILE}' 2>&1"
