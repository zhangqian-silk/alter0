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
LOG_FILE="${ALTER0_LOG_FILE:-/var/log/alter0/alter0.log}"
LOCK_FILE="${RUNTIME_ROOT}/run.lock"
ALTER0_RUNTIME_MANAGER="${ALTER0_RUNTIME_MANAGER:-systemd}"
ALTER0_SYSTEMD_UNIT="${ALTER0_SYSTEMD_UNIT:-alter0.service}"
BUILD_OUTPUT="${ALTER0_BUILD_OUTPUT:-${REPO_DIR}/bin/alter0}"

mkdir -p "${RUNTIME_ROOT}" "${STORAGE_DIR}" "$(dirname "${LOG_FILE}")" "${REPO_DIR}/.alter0" "$(dirname "${BUILD_OUTPUT}")"
chmod 750 "${RUNTIME_ROOT}"
chmod 700 "${STORAGE_DIR}" "${REPO_DIR}/.alter0"
touch "${LOG_FILE}"
chmod 640 "${LOG_FILE}"

exec 9>"${LOCK_FILE}"
flock -n 9 || {
  echo "alter0 service is already running"
  exit 1
}

if [[ -z "${WEB_LOGIN_PASSWORD}" ]]; then
  echo "[WARN] ALTER0_WEB_LOGIN_PASSWORD is empty, login page is disabled."
fi

if [[ ! -x "${BUILD_OUTPUT}" ]]; then
  env GOSUMDB="${GOSUMDB:-sum.golang.org}" GOTOOLCHAIN="${GOTOOLCHAIN:-auto}" go build -o "${BUILD_OUTPUT}" ./cmd/alter0
fi

CMD=(
  env
  "GOSUMDB=${GOSUMDB:-sum.golang.org}"
  "GOTOOLCHAIN=${GOTOOLCHAIN:-auto}"
  "ALTER0_RUNTIME_MANAGER=${ALTER0_RUNTIME_MANAGER}"
  "ALTER0_SYSTEMD_UNIT=${ALTER0_SYSTEMD_UNIT}"
  "ALTER0_BUILD_OUTPUT=${BUILD_OUTPUT}"
  "${BUILD_OUTPUT}"
  -web-addr "${WEB_ADDR}"
  -web-bind-localhost-only="${WEB_BIND_LOCALHOST_ONLY}"
  -web-login-password "${WEB_LOGIN_PASSWORD}"
  -daily-memory-dir "${STORAGE_DIR}/memory"
  -long-term-memory-path "${STORAGE_DIR}/memory/long-term/MEMORY.md"
)

if [[ "$(id -u)" -eq 0 ]] && id -u "${RUN_AS}" >/dev/null 2>&1 && [[ "${RUN_AS}" != "root" ]]; then
  chown -R "${RUN_AS}:${RUN_AS}" "${RUNTIME_ROOT}" "$(dirname "${LOG_FILE}")" "${REPO_DIR}/.alter0" "$(dirname "${BUILD_OUTPUT}")"
  exec su -s /bin/bash -c "cd '${REPO_DIR}' && exec \"\$@\" >>'${LOG_FILE}' 2>&1" "${RUN_AS}" bash "${CMD[@]}"
fi

cd "${REPO_DIR}"
exec "${CMD[@]}" >>"${LOG_FILE}" 2>&1
