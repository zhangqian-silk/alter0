#!/usr/bin/env bash
set -euo pipefail

umask 027

ENV_FILE="${ALTER0_ENV_FILE:-/etc/alter0/alter0.env}"
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

RUN_AS="${ALTER0_RUN_AS:-alter0}"
if ! id "${RUN_AS}" >/dev/null 2>&1; then
  echo "user not found: ${RUN_AS}" >&2
  exit 1
fi
RUN_GROUP="$(id -gn "${RUN_AS}")"

RUNTIME_ROOT="${ALTER0_RUNTIME_ROOT:-/var/lib/alter0}"
HOME_DIR="${ALTER0_HOME:-${RUNTIME_ROOT}}"
if [[ "${HOME_DIR}" == "${RUNTIME_ROOT}/codex-home" ]]; then
  HOME_DIR="${RUNTIME_ROOT}"
fi

GIT_USER_NAME="${ALTER0_GIT_USER_NAME:-alter0-silk[bot]}"
GIT_USER_EMAIL="${ALTER0_GIT_USER_EMAIL:-264108545+alter0-silk[bot]@users.noreply.github.com}"
REPO_DIR="${ALTER0_REPO_DIR:-/srv/alter0/app}"
SOURCE_SIGNING_KEY="${ALTER0_GIT_SSH_SIGNING_KEY_SOURCE:-/root/.ssh/id_ed25519}"
SOURCE_SIGNING_PUB="${ALTER0_GIT_SSH_SIGNING_PUB_SOURCE:-${SOURCE_SIGNING_KEY}.pub}"
TARGET_SIGNING_KEY="${HOME_DIR}/.ssh/id_ed25519"
TARGET_SIGNING_PUB="${HOME_DIR}/.ssh/id_ed25519.pub"
ALLOWED_SIGNERS_PATH="${HOME_DIR}/.ssh/allowed_signers"
GITCONFIG_PATH="${HOME_DIR}/.gitconfig"
LOCAL_BIN_DIR="${HOME_DIR}/.local/bin"
GIT_HELPER_PATH="${LOCAL_BIN_DIR}/git-credential-github-app-token"
GH_WRAPPER_PATH="${LOCAL_BIN_DIR}/gh"
GH_HOSTS_PATH="${HOME_DIR}/.config/gh/hosts.yml"
GITHUB_APP_DIR="${ALTER0_GITHUB_APP_DIR:-/etc/github-app}"

if [[ ! -x /usr/local/bin/github-app-token ]]; then
  echo "missing /usr/local/bin/github-app-token" >&2
  exit 1
fi

if [[ ! -f "${SOURCE_SIGNING_KEY}" || ! -f "${SOURCE_SIGNING_PUB}" ]]; then
  echo "missing signing key source: ${SOURCE_SIGNING_KEY}" >&2
  exit 1
fi

install -d -m 750 -o "${RUN_AS}" -g "${RUN_GROUP}" "${HOME_DIR}" "${HOME_DIR}/.local" "${LOCAL_BIN_DIR}"
install -d -m 700 -o "${RUN_AS}" -g "${RUN_GROUP}" "${HOME_DIR}/.ssh"
install -d -m 750 -o "${RUN_AS}" -g "${RUN_GROUP}" "${HOME_DIR}/.config" "${HOME_DIR}/.config/gh"
rm -f "${GH_HOSTS_PATH}"

if [[ -d "${GITHUB_APP_DIR}" ]]; then
  chgrp "${RUN_GROUP}" "${GITHUB_APP_DIR}"
  chmod 750 "${GITHUB_APP_DIR}"
  find "${GITHUB_APP_DIR}" -maxdepth 1 -type f -exec chgrp "${RUN_GROUP}" {} +
  find "${GITHUB_APP_DIR}" -maxdepth 1 -type f -exec chmod 640 {} +
fi

install -m 600 -o "${RUN_AS}" -g "${RUN_GROUP}" "${SOURCE_SIGNING_KEY}" "${TARGET_SIGNING_KEY}"
install -m 644 -o "${RUN_AS}" -g "${RUN_GROUP}" "${SOURCE_SIGNING_PUB}" "${TARGET_SIGNING_PUB}"
printf '%s %s\n' "${GIT_USER_EMAIL}" "$(cat "${TARGET_SIGNING_PUB}")" >"${ALLOWED_SIGNERS_PATH}"
chown "${RUN_AS}:${RUN_GROUP}" "${ALLOWED_SIGNERS_PATH}"
chmod 644 "${ALLOWED_SIGNERS_PATH}"

cat >"${GIT_HELPER_PATH}" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

action="${1:-get}"
case "${action}" in
  get) ;;
  store|erase)
    exit 0
    ;;
  *)
    exit 0
    ;;
esac

protocol=""
host=""
while IFS='=' read -r key value; do
  case "${key}" in
    protocol) protocol="${value}" ;;
    host) host="${value}" ;;
  esac
done

if [[ "${protocol}" != "https" || "${host}" != "github.com" ]]; then
  exit 0
fi

token="$(/usr/local/bin/github-app-token)"
if [[ -z "${token}" ]]; then
  exit 1
fi

printf 'username=%s\n' "x-access-token"
printf 'password=%s\n' "${token}"
EOF
chown "${RUN_AS}:${RUN_GROUP}" "${GIT_HELPER_PATH}"
chmod 750 "${GIT_HELPER_PATH}"

cat >"${GH_WRAPPER_PATH}" <<EOF
#!/usr/bin/env bash
set -euo pipefail

token="\$(${GIT_HELPER_PATH} get <<'INPUT' | sed -n 's/^password=//p'
protocol=https
host=github.com
INPUT
)"
if [[ -z "\${token}" ]]; then
  echo "failed to resolve GitHub token" >&2
  exit 1
fi

export GH_TOKEN="\${token}"
export GITHUB_TOKEN="\${token}"
exec /usr/bin/gh "\$@"
EOF
chown "${RUN_AS}:${RUN_GROUP}" "${GH_WRAPPER_PATH}"
chmod 750 "${GH_WRAPPER_PATH}"

touch "${GITCONFIG_PATH}"
chown "${RUN_AS}:${RUN_GROUP}" "${GITCONFIG_PATH}"
chmod 640 "${GITCONFIG_PATH}"

tmp_gitconfig="$(mktemp)"
cat >"${tmp_gitconfig}" <<EOF
[safe]
	directory = /srv/alter0/app
[user]
	name = ${GIT_USER_NAME}
	email = ${GIT_USER_EMAIL}
	signingkey = ${TARGET_SIGNING_PUB}
[gpg]
	format = ssh
[gpg "ssh"]
	allowedSignersFile = ${ALLOWED_SIGNERS_PATH}
[commit]
	gpgsign = true
[credential]
	helper = !${GIT_HELPER_PATH}
EOF
install -m 640 -o "${RUN_AS}" -g "${RUN_GROUP}" "${tmp_gitconfig}" "${GITCONFIG_PATH}"
rm -f "${tmp_gitconfig}"

if [[ -d "${REPO_DIR}/.git" ]]; then
  git -C "${REPO_DIR}" config user.name "${GIT_USER_NAME}"
  git -C "${REPO_DIR}" config user.email "${GIT_USER_EMAIL}"
fi

echo "runtime auth prepared for ${RUN_AS} at ${HOME_DIR}"
