#!/usr/bin/env bash
set -euo pipefail

umask 027

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
ENV_FILE="${ALTER0_ENV_FILE:-/etc/alter0/alter0.env}"
SUDOERS_TARGET="${ALTER0_SUDOERS_TARGET:-/etc/sudoers.d/alter0-relaunch}"
RUN_AS="${ALTER0_RUN_AS:-alter0}"
UNIT="${ALTER0_SYSTEMD_UNIT:-alter0.service}"
RELAUNCH_SCRIPT="${ALTER0_RELAUNCH_SCRIPT:-${REPO_DIR}/scripts/relaunch_service.sh}"

if [[ -f "${ENV_FILE}" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "${ENV_FILE}"
  set +a
fi

if [[ "$(id -u)" -ne 0 ]]; then
  echo "must run as root" >&2
  exit 1
fi

if [[ ! -x "${RELAUNCH_SCRIPT}" ]]; then
  echo "relaunch script not found or not executable: ${RELAUNCH_SCRIPT}" >&2
  exit 1
fi

cat >"${SUDOERS_TARGET}" <<EOF
Cmnd_Alias ALTER0_RELAUNCH = ${RELAUNCH_SCRIPT} --unit ${UNIT}
${RUN_AS} ALL=(root) NOPASSWD: ALTER0_RELAUNCH
Defaults!ALTER0_RELAUNCH secure_path=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/bin
EOF

chmod 440 "${SUDOERS_TARGET}"
visudo -cf "${SUDOERS_TARGET}"
