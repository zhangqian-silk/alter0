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

# Keep node/npm under the service account HOME so alter0 can use a private,
# reproducible toolchain without relying on host-level npm or mutating /usr/local.
LOCAL_DIR="${HOME_DIR}/.local"
LOCAL_BIN_DIR="${LOCAL_DIR}/bin"
NODE_INSTALL_ROOT="${ALTER0_NODE_INSTALL_ROOT:-${LOCAL_DIR}/nodejs}"
PLAYWRIGHT_PROJECT_DIR="${ALTER0_PLAYWRIGHT_PROJECT_DIR:-/srv/alter0/app/internal/interfaces/web}"
INSTALL_E2E_DEPS="${ALTER0_INSTALL_E2E_DEPS:-true}"
PREPARE_PNPM="${ALTER0_PREPARE_PNPM:-true}"

detect_node_version() {
  if [[ -n "${ALTER0_NODE_VERSION:-}" ]]; then
    printf '%s\n' "${ALTER0_NODE_VERSION#v}"
    return 0
  fi
  if [[ -x /usr/local/bin/node ]]; then
    /usr/local/bin/node --version | sed 's/^v//'
    return 0
  fi
  if command -v node >/dev/null 2>&1; then
    node --version | sed 's/^v//'
    return 0
  fi
  printf '%s\n' "22.22.0"
}

detect_platform() {
  case "$(uname -s)" in
    Linux) printf '%s\n' "linux" ;;
    Darwin) printf '%s\n' "darwin" ;;
    *)
      echo "unsupported platform: $(uname -s)" >&2
      exit 1
      ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) printf '%s\n' "x64" ;;
    aarch64|arm64) printf '%s\n' "arm64" ;;
    *)
      echo "unsupported architecture: $(uname -m)" >&2
      exit 1
      ;;
  esac
}

NODE_VERSION="$(detect_node_version)"
NODE_PLATFORM="$(detect_platform)"
NODE_ARCH="$(detect_arch)"
NODE_DISTRO="node-v${NODE_VERSION}-${NODE_PLATFORM}-${NODE_ARCH}"
NODE_TARBALL="${NODE_DISTRO}.tar.xz"
NODE_URL="https://nodejs.org/dist/v${NODE_VERSION}/${NODE_TARBALL}"
NODE_DEST="${NODE_INSTALL_ROOT}/${NODE_DISTRO}"

install -d -m 750 -o "${RUN_AS}" -g "${RUN_GROUP}" "${HOME_DIR}" "${LOCAL_DIR}" "${LOCAL_BIN_DIR}" "${NODE_INSTALL_ROOT}"

if [[ ! -x "${NODE_DEST}/bin/npm" ]]; then
  tmp_dir="$(mktemp -d)"
  trap 'rm -rf "${tmp_dir}"' EXIT
  archive_path="${tmp_dir}/${NODE_TARBALL}"
  echo "downloading ${NODE_URL}"
  curl -fsSL "${NODE_URL}" -o "${archive_path}"
  rm -rf "${NODE_DEST}"
  tar -xJf "${archive_path}" -C "${NODE_INSTALL_ROOT}"
  chown -R "${RUN_AS}:${RUN_GROUP}" "${NODE_DEST}"
  chmod 750 "${NODE_INSTALL_ROOT}" "${NODE_DEST}" "${NODE_DEST}/bin"
fi

for tool in node npm npx corepack; do
  if [[ ! -x "${NODE_DEST}/bin/${tool}" ]]; then
    echo "missing ${tool} in ${NODE_DEST}/bin" >&2
    exit 1
  fi
  # Expose stable command paths from $HOME/.local/bin so service children only
  # need PATH to include one directory, even if the node version changes later.
  rm -f "${LOCAL_BIN_DIR}/${tool}"
  ln -s "${NODE_DEST}/bin/${tool}" "${LOCAL_BIN_DIR}/${tool}"
done

if [[ "${PREPARE_PNPM}" == "true" ]]; then
  rm -f "${LOCAL_BIN_DIR}/pnpm" "${LOCAL_BIN_DIR}/pnpx"
  su -s /bin/bash "${RUN_AS}" -c "export HOME='${HOME_DIR}' PATH='${LOCAL_BIN_DIR}:/usr/local/bin:/usr/bin:/bin'; '${LOCAL_BIN_DIR}/corepack' enable --install-directory '${LOCAL_BIN_DIR}' pnpm"
fi

if [[ "${INSTALL_E2E_DEPS}" == "true" && -d "${PLAYWRIGHT_PROJECT_DIR}" ]]; then
  if [[ -d "${PLAYWRIGHT_PROJECT_DIR}/node_modules" ]]; then
    chown -R "${RUN_AS}:${RUN_GROUP}" "${PLAYWRIGHT_PROJECT_DIR}/node_modules"
  fi
  su -s /bin/bash "${RUN_AS}" -c "export HOME='${HOME_DIR}' PATH='${LOCAL_BIN_DIR}:/usr/local/bin:/usr/bin:/bin'; cd '${PLAYWRIGHT_PROJECT_DIR}' && npm ci && npx playwright install chromium"
fi

echo "runtime node prepared for ${RUN_AS} at ${HOME_DIR}"
