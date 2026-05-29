#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

BUILD_OUTPUT="${ALTER0_BUILD_OUTPUT:-${REPO_DIR}/bin/alter0}"
FRONTEND_DIR="${ALTER0_FRONTEND_PROJECT_DIR:-${REPO_DIR}/internal/interfaces/web/frontend}"
DIST_INDEX="${REPO_DIR}/internal/interfaces/web/static/dist/index.html"
SKIP_FRONTEND_BUILD="${ALTER0_SKIP_FRONTEND_BUILD:-0}"
FRONTEND_INSTALL_DEPS="${ALTER0_FRONTEND_INSTALL_DEPS:-auto}"

mkdir -p "$(dirname "${BUILD_OUTPUT}")"

if [[ "${SKIP_FRONTEND_BUILD}" != "1" ]]; then
  if [[ ! -f "${FRONTEND_DIR}/package.json" ]]; then
    echo "missing frontend package: ${FRONTEND_DIR}/package.json" >&2
    exit 1
  fi
  (
    cd "${FRONTEND_DIR}"
    if [[ ! -d node_modules && "${FRONTEND_INSTALL_DEPS}" != "never" ]]; then
      npm ci
    fi
    npm run build
  )
fi

if [[ ! -f "${DIST_INDEX}" ]]; then
  echo "missing built frontend dist: ${DIST_INDEX}" >&2
  exit 1
fi
if ! grep -Eq '/assets/index-[^"]+\.js' "${DIST_INDEX}"; then
  echo "frontend dist index does not reference a hashed JavaScript asset" >&2
  exit 1
fi
if ! grep -Eq '/assets/index-[^"]+\.css' "${DIST_INDEX}"; then
  echo "frontend dist index does not reference a hashed CSS asset" >&2
  exit 1
fi

cd "${REPO_DIR}"
env GOSUMDB="${GOSUMDB:-sum.golang.org}" GOTOOLCHAIN="${GOTOOLCHAIN:-auto}" go build -o "${BUILD_OUTPUT}" ./cmd/alter0
